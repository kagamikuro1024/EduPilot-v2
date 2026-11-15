package attendance

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// DeviceIn là `device` của POST …/checkins: WEB_KEY {public_key (SPKI), signature} hoặc ANDROID_KEY {key_id, signature}. Chuỗi base64 (chuẩn hoặc url).
type DeviceIn struct {
	Kind      string `json:"kind"`
	PublicKey string `json:"public_key"`
	KeyID     string `json:"key_id"`
	Signature string `json:"signature"`
}

// GeoIn là `geo`: chỉ dùng khi lớp khai geofence; toạ độ bị bỏ ngay sau khi tính khoảng cách (không vào DB, log, audit).
type GeoIn struct {
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	AccuracyM float64 `json:"accuracy_m"`
	Mock      bool    `json:"mock"`
	Denied    bool    `json:"denied"`
}

// AppIn chỉ để chẩn đoán (không đổi quyết định).
type AppIn struct {
	Version string `json:"version"`
}

// CheckinIn là thân `POST /courses/{id}/checkins`: đúng MỘT trong `token`, `code`.
type CheckinIn struct {
	Token  string    `json:"token"`
	Code   string    `json:"code"`
	Device *DeviceIn `json:"device"`
	Geo    *GeoIn    `json:"geo"`
	App    *AppIn    `json:"app"`
}

// CheckinOut là phản hồi cho sinh viên: KHÔNG cờ, KHÔNG mức tin cậy (so byte ở TestCheckinResponseHidesFlags).
type CheckinOut struct {
	CheckedInAt time.Time `json:"checked_in_at"`
	Round       int       `json:"round"`
}

func tokenInvalid() *apierr.Error {
	return apierr.New(http.StatusUnprocessableEntity, apierr.CheckinTokenInvalid)
}

func codeInvalid() *apierr.Error {
	return apierr.New(http.StatusUnprocessableEntity, apierr.CheckinTokenInvalid).WithMessage("Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu.")
}

func tokenExpired() *apierr.Error {
	return apierr.New(http.StatusUnprocessableEntity, apierr.CheckinTokenExpired)
}

func decodeB64(s string) ([]byte, bool) {
	if s == "" {
		return nil, false
	}
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if b, err := enc.DecodeString(s); err == nil {
			return b, true
		}
	}
	return nil, false
}

// signals là tín hiệu thu được trước khi ghi; không có tín hiệu nào làm lượt bị từ chối (FR-10, FR-11).
type signals struct {
	deviceID   *uuid.UUID
	flags      []string
	assurance  int16
	geoDist    *int32
	geoAcc     *int32
	inCampus   *bool
	channel    store.CheckinChannel
	attestedOK bool // chữ ký ANDROID_KEY hợp lệ của khoá đã chứng thực
}

// Checkin ghi một lượt điểm danh của sinh viên `student` (= `sub` của JWT) vào lớp `courseID` (FR-10…FR-13). clientIP là IP máy khách theo hàm client-IP của
// middleware hạn mức (chỉ để tính `in_campus`, không lưu). Trả (kết quả, mới?) — gửi lại cùng lượt → mới=false (HTTP 200, cùng bản ghi cũ).
func (s *Service) Checkin(ctx context.Context, student, courseID uuid.UUID, in CheckinIn, clientIP string) (CheckinOut, bool, error) {
	if err := s.rate(ctx, "checkin", student, s.Cfg.RatePerMin); err != nil {
		return CheckinOut{}, false, err
	}
	if (in.Token == "") == (in.Code == "") {
		return CheckinOut{}, false, invalid("token", "exactly_one", "Cần đúng một trong mã quét hoặc mã nhập tay.")
	}
	if !s.Cfg.Enabled {
		return CheckinOut{}, false, conflict(apierr.SelfCheckinDisabled)
	}
	q := store.New(s.Pool)
	c, pol, err := s.course(ctx, q, courseID)
	if err != nil {
		return CheckinOut{}, false, err
	}
	if c.Status != store.CourseStatusACTIVE {
		return CheckinOut{}, false, apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	if !pol.SelfCheckin {
		return CheckinOut{}, false, conflict(apierr.SelfCheckinDisabled)
	}
	now := s.now()
	nowStep := func(rot int) int64 { return Step(now.Unix(), rot) }

	var (
		w    store.CheckinWindow
		step int64
		norm string
	)
	if in.Token != "" { // đường token
		p, ok := ParseToken(in.Token)
		if !ok {
			return CheckinOut{}, false, tokenInvalid()
		}
		w, err = q.AttWindowByID(ctx, store.AttWindowByIDParams{CourseID: courseID, ID: p.WindowID})
		if err != nil {
			if isNoRows(err) {
				return CheckinOut{}, false, tokenInvalid() // cửa sổ lạ / của lớp khác: không phân biệt
			}
			return CheckinOut{}, false, wrap("đọc cửa sổ", err)
		}
		if !VerifyMAC(s.Cfg.TokenKey, p) {
			return CheckinOut{}, false, tokenInvalid()
		}
		if w.ClosedAt != nil || !w.ClosesAt.After(now) {
			return CheckinOut{}, false, conflict(apierr.WindowClosed)
		}
		if !StepAccepted(p.Step, nowStep(int(w.RotateSeconds))) {
			return CheckinOut{}, false, tokenExpired()
		}
		step = p.Step
	} else { // đường mã ngắn: dò cửa sổ của lớp
		nc, ok := NormalizeCode(in.Code)
		if !ok || len(nc) != s.Cfg.CodeLen {
			if err := s.countCodeFail(ctx, student); err != nil {
				return CheckinOut{}, false, err
			}
			return CheckinOut{}, false, codeInvalid()
		}
		norm = nc
		if err := s.checkCodeFail(ctx, student); err != nil {
			return CheckinOut{}, false, err
		}
		ws, err := q.AttWindowsForCode(ctx, store.AttWindowsForCodeParams{CourseID: courseID, Since: now.Add(-time.Duration(s.Cfg.CodeLookbackMin) * time.Minute)})
		if err != nil {
			return CheckinOut{}, false, wrap("dò cửa sổ", err)
		}
		found := false
		for _, cand := range ws {
			ns := nowStep(int(cand.RotateSeconds))
			for _, st := range []int64{ns, ns - 1} {
				if st >= 0 && CodeMatches(s.Cfg.TokenKey, cand.ID, st, s.Cfg.CodeLen, norm) {
					w, step, found = cand, st, true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			if err := s.countCodeFail(ctx, student); err != nil {
				return CheckinOut{}, false, err
			}
			return CheckinOut{}, false, codeInvalid()
		}
		if w.ClosedAt != nil || !w.ClosesAt.After(now) {
			return CheckinOut{}, false, conflict(apierr.WindowClosed)
		}
	}

	// gửi lại (F5, hai tab, mạng chập chờn): trả bản ghi cũ, không ghi gì
	if old, err := q.AttCheckinFind(ctx, store.AttCheckinFindParams{CourseID: courseID, WindowID: w.ID, StudentID: student}); err == nil {
		return CheckinOut{CheckedInAt: old.CreatedAt, Round: int(w.Round)}, false, nil
	} else if !isNoRows(err) {
		return CheckinOut{}, false, wrap("đọc lượt cũ", err)
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return CheckinOut{}, false, wrap("mở giao dịch", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qt := store.New(tx)
	sg, err := s.evaluate(ctx, qt, student, courseID, w, step, now, norm, in, pol, clientIP)
	if err != nil {
		return CheckinOut{}, false, err
	}
	row, err := qt.AttCheckinInsert(ctx, store.AttCheckinInsertParams{
		CourseID: courseID, SessionID: w.SessionID, WindowID: w.ID, StudentID: student, DeviceID: sg.deviceID, Channel: sg.channel, Assurance: sg.assurance,
		TokenAgeMs: int32(TokenAgeMs(now.UnixMilli(), step, int(w.RotateSeconds))), GeoDistanceM: sg.geoDist, GeoAccuracyM: sg.geoAcc, InCampus: sg.inCampus, Flags: sg.flags, CreatedAt: now,
	})
	if err != nil {
		if isNoRows(err) { // đua: lượt khác của chính người này vừa ghi
			_ = tx.Rollback(ctx)
			old, ferr := q.AttCheckinFind(ctx, store.AttCheckinFindParams{CourseID: courseID, WindowID: w.ID, StudentID: student})
			if ferr != nil {
				return CheckinOut{}, false, wrap("đọc lượt vừa ghi", ferr)
			}
			return CheckinOut{CheckedInAt: old.CreatedAt, Round: int(w.Round)}, false, nil
		}
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23503" && strings.Contains(pg.ConstraintName, "student_fkey") {
			return CheckinOut{}, false, apierr.New(http.StatusForbidden, apierr.Forbidden).WithDetails(map[string]string{"reason": "course"}) // bị mời ra giữa guard và INSERT: như người ngoài lớp
		}
		return CheckinOut{}, false, wrap("ghi lượt", err)
	}
	if slices.Contains(sg.flags, FlagDeviceShared) && sg.deviceID != nil { // cờ cũng lên mọi lượt của những người dùng chung máy trong buổi (FR-12)
		others, err := qt.AttSharedDeviceStudents(ctx, store.AttSharedDeviceStudentsParams{CourseID: courseID, DeviceID: sg.deviceID, StudentID: student})
		if err != nil {
			return CheckinOut{}, false, wrap("người dùng chung máy", err)
		}
		if err := qt.AttFlagShared(ctx, store.AttFlagSharedParams{CourseID: courseID, SessionID: w.SessionID, StudentIds: idStrings(others)}); err != nil {
			return CheckinOut{}, false, wrap("gắn cờ dùng chung", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return CheckinOut{}, false, wrap("commit", err)
	}
	return CheckinOut{CheckedInAt: row.CreatedAt, Round: int(w.Round)}, true, nil
}

// evaluate dựng tín hiệu (thiết bị, vị trí, mạng) và `assurance`. KHÔNG bao giờ trả lỗi vì tín hiệu: chỉ gắn cờ.
func (s *Service) evaluate(ctx context.Context, q *store.Queries, student, courseID uuid.UUID, w store.CheckinWindow, step int64, now time.Time, normCode string, in CheckinIn, pol Policy, clientIP string) (signals, error) {
	sg := signals{channel: store.CheckinChannelWEB, assurance: 1, flags: []string{}}
	if in.Code != "" {
		sg.channel = store.CheckinChannelCODE
	} else if (in.Device != nil && in.Device.Kind == KindAndroidKey) || in.App != nil {
		sg.channel = store.CheckinChannelAPP
	}
	var tokenForSig string
	if in.Token != "" {
		tokenForSig = in.Token
	}
	msg := SignedMessage(tokenForSig, normCode, student)

	validSig, bootVerified := false, false
	var dev *store.UserDevice
	switch {
	case in.Device == nil:
		sg.flags = append(sg.flags, FlagNoDeviceKey)
	case in.Device.Kind == KindWebKey:
		spki, ok1 := decodeB64(in.Device.PublicKey)
		sig, ok2 := decodeB64(in.Device.Signature)
		pk, ok3 := ParseP256(spki)
		if !ok1 || !ok2 || !ok3 || !VerifySignature(pk, msg, sig) {
			sg.flags = append(sg.flags, FlagDeviceUntrusted)
			break
		}
		d, err := q.AttDeviceUpsertWeb(ctx, store.AttDeviceUpsertWebParams{KeySha256: keySum(spki), PublicKey: spki})
		if err != nil {
			return signals{}, wrap("ghi khoá thiết bị", err)
		}
		dev, validSig = &d, true
	case in.Device.Kind == KindAndroidKey:
		sig, ok := decodeB64(in.Device.Signature)
		kid, ok2 := decodeB64(in.Device.KeyID)
		if !ok || !ok2 || len(kid) != 32 {
			sg.flags = append(sg.flags, FlagDeviceUntrusted)
			break
		}
		d, err := q.AttDeviceByHashLock(ctx, kid)
		if err != nil {
			if isNoRows(err) { // khoá app chưa đăng ký
				sg.flags = append(sg.flags, FlagDeviceUntrusted)
				break
			}
			return signals{}, wrap("đọc khoá thiết bị", err)
		}
		pk, ok3 := ParseP256(d.PublicKey)
		if !ok3 || !VerifySignature(pk, msg, sig) {
			sg.flags = append(sg.flags, FlagDeviceUntrusted)
			break
		}
		dev, validSig = &d, true
		if d.Attested && d.Kind == store.DeviceKindANDROIDKEY {
			sg.attestedOK = true
			bootVerified = d.BootVerified != nil && *d.BootVerified
			if !bootVerified {
				sg.flags = append(sg.flags, FlagDeviceUntrusted) // máy mở khoá / không xác minh được khởi động (SRS 4.4)
			}
		}
	default:
		sg.flags = append(sg.flags, FlagDeviceUntrusted)
	}
	if dev != nil {
		id := dev.ID
		sg.deviceID = &id
		others, err := q.AttSharedDeviceStudents(ctx, store.AttSharedDeviceStudentsParams{CourseID: courseID, DeviceID: sg.deviceID, StudentID: student})
		if err != nil {
			return signals{}, wrap("kiểm dùng chung máy", err)
		}
		if len(others) > 0 {
			sg.flags = append(sg.flags, FlagDeviceShared)
		}
		sd, err := q.AttStudentDevices(ctx, store.AttStudentDevicesParams{CourseID: courseID, StudentID: student, DeviceID: sg.deviceID})
		if err != nil {
			return signals{}, wrap("kiểm máy mới", err)
		}
		if sd.Other && !sd.This {
			sg.flags = append(sg.flags, FlagDeviceNew)
		}
	}
	if TokenAgeMs(now.UnixMilli(), step, int(w.RotateSeconds)) > int64(w.RotateSeconds)*1000+int64(s.Cfg.TokenLateExtraMs) {
		sg.flags = append(sg.flags, FlagTokenLate)
	}
	if pol.Require == RequireApp && !sg.attestedOK {
		sg.flags = append(sg.flags, FlagNotApp)
	}

	// assurance (SRS 4.5, ARCHITECTURE 11.3)
	switch {
	case !validSig:
		sg.assurance = 1
	case sg.attestedOK && bootVerified:
		sg.assurance = 3
	default:
		sg.assurance = 2
	}

	// vị trí, mạng: chỉ khi lớp khai (FR-13); toạ độ / IP là biến cục bộ, bỏ ngay sau khi tính
	geoBad := false
	if pol.Geofence != nil {
		switch {
		case in.Geo == nil || in.Geo.Denied:
			sg.flags = append(sg.flags, FlagGeoDenied)
			geoBad = true
		default:
			d := Haversine(in.Geo.Lat, in.Geo.Lng, pol.Geofence.Lat, pol.Geofence.Lng)
			dm, am := int32(math.Round(d)), int32(math.Round(math.Max(in.Geo.AccuracyM, 0)))
			sg.geoDist, sg.geoAcc = &dm, &am
			if d-in.Geo.AccuracyM > float64(pol.Geofence.RadiusM) {
				sg.flags = append(sg.flags, FlagGeoOut)
				geoBad = true
			}
			if in.Geo.AccuracyM > float64(s.Cfg.GeoMaxAccuracyM) {
				sg.flags = append(sg.flags, FlagGeoWeak)
				geoBad = true
			}
			if in.Geo.Mock {
				sg.flags = append(sg.flags, FlagGeoMock)
				if sg.assurance == 3 {
					sg.assurance = 2
				}
			}
		}
	}
	if len(pol.CampusCIDRs) > 0 {
		if ip, err := netip.ParseAddr(strings.TrimSpace(clientIP)); err == nil {
			ip = ip.Unmap()
			in4 := false
			for _, c := range pol.CampusCIDRs {
				if pf, err := netip.ParsePrefix(c); err == nil && pf.Contains(ip) {
					in4 = true
					break
				}
			}
			sg.inCampus = &in4
			if !in4 {
				sg.flags = append(sg.flags, FlagOffCampus)
			}
		}
	}
	if geoBad {
		sg.assurance = 1
	}
	return sg, nil
}

// Haversine: khoảng cách mét giữa hai điểm (R = 6.371.000 m).
func Haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const r = 6371000.0
	rad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat, dLng := rad(lat2-lat1), rad(lng2-lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(rad(lat1))*math.Cos(rad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * r * math.Asin(math.Min(1, math.Sqrt(a)))
}

// checkCodeFail: ≥ CHECKIN_CODE_FAIL_PER_MIN lần nhập SAI trong 60 s → 429 (cả khi đúng sau đó, tới hết cửa sổ đếm). Redis nil / chết → cho qua.
func (s *Service) checkCodeFail(ctx context.Context, student uuid.UUID) error {
	if s.Redis == nil {
		return nil
	}
	n, err := s.Redis.Get(ctx, appredis.Key("att", "codefail", student.String())).Int()
	if err != nil {
		return nil // chưa có khoá (redis.Nil) hoặc Redis chết: cho qua
	}
	if n >= s.Cfg.CodeFailPerMin {
		ttl, terr := s.Redis.TTL(ctx, appredis.Key("att", "codefail", student.String())).Result()
		sec := 60
		if terr == nil && ttl > 0 {
			sec = int(math.Ceil(ttl.Seconds()))
		}
		return apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(sec)
	}
	return nil
}

// countCodeFail: chỉ lần nhập sai mới tăng bộ đếm (INCR + EXPIRE NX 60).
func (s *Service) countCodeFail(ctx context.Context, student uuid.UUID) error {
	if s.Redis == nil {
		return nil
	}
	key := appredis.Key("att", "codefail", student.String())
	pipe := s.Redis.TxPipeline()
	pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, 60*time.Second)
	if _, err := pipe.Exec(ctx); err != nil {
		s.log().WarnContext(ctx, "attendance: bộ đếm nhập sai lỗi, cho qua")
	}
	return nil
}

// MyCheckedIn trả giờ điểm danh sớm nhất theo buổi của một sinh viên (cho `GET …/me/attendance`, FR-17): chỉ thời điểm, không cờ.
func (s *Service) MyCheckedIn(ctx context.Context, courseID, student uuid.UUID) (map[uuid.UUID]time.Time, error) {
	rows, err := store.New(s.Pool).AttMyCheckins(ctx, store.AttMyCheckinsParams{CourseID: courseID, StudentID: student})
	if err != nil {
		return nil, fmt.Errorf("attendance: giờ điểm danh: %w", dbErr(err))
	}
	out := make(map[uuid.UUID]time.Time, len(rows))
	for _, r := range rows {
		out[r.SessionID] = r.CheckedInAt
	}
	return out, nil
}
