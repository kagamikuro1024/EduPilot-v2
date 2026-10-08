package contract

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// examResultScenarios: thao tác 26, 42, 44–50, 55, 56 (công bố, kết quả cho Staff, sửa điểm, chấm lại, phúc khảo — US-PE-08) với mọi status đã khai báo.
// 47 trả 202 chỉ với > 200 lượt GRADED: được miễn ở exempt.go (service `exam.TestOverrideRecomputes` + `TestOverrideLargeEnqueues`).
func (r *runner) examResultScenarios(x examRig) {
	db := r.rig.deps.DB
	ctx := context.Background()
	ex := "/api/v1/courses/" + x.cid + "/exams"
	now := time.Now().UTC()
	create := `{"title":"Bài có kết quả","opens_at":"` + now.Add(2*time.Hour).Format(time.RFC3339) + `","closes_at":"` + now.Add(4*time.Hour).Format(time.RFC3339) + `","duration_minutes":45}`
	_, b := r.must(call{method: "POST", path: ex, token: x.gv, headers: x.idem(), body: create}, 201)
	var e struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	_ = json.Unmarshal(b, &e)
	r.must(call{method: "PUT", path: ex + "/" + e.ID + "/items", token: x.gv, body: `{"items":[{"question_id":"` + x.q1 + `","points":"2"}],"version":` + itoa(e.Version) + `}`}, 200)
	r.must(call{method: "POST", path: ex + "/" + e.ID + "/schedule", token: x.gv}, 200)
	exec := func(sql string, args ...any) {
		if _, err := db.Exec(ctx, sql, args...); err != nil {
			r.t.Fatal(err)
		}
	}
	exec(`update exams set status='OPEN', opens_at=now() - interval '1 minute', closes_at=now() + interval '3 hours' where id=$1`, e.ID)
	one := ex + "/" + e.ID
	tab := uuid.NewString()
	hdr := func(idem bool) map[string]string {
		h := map[string]string{"X-Exam-Tab": tab}
		if idem {
			h["Idempotency-Key"] = "ct-" + uuid.NewString()
		}
		return h
	}
	_, b = r.must(call{method: "POST", path: one + "/attempts", token: x.sv, headers: hdr(true)}, 201)
	var st struct {
		Attempt struct {
			ID string `json:"id"`
		} `json:"attempt"`
		Items []struct {
			ItemID  string `json:"item_id"`
			Options []struct {
				ID string `json:"id"`
			} `json:"options"`
		} `json:"items"`
	}
	_ = json.Unmarshal(b, &st)
	att := one + "/attempts/" + st.Attempt.ID
	aid, item := st.Attempt.ID, st.Items[0].ItemID
	r.must(call{method: "PUT", path: att + "/answers", token: x.sv, headers: hdr(false), body: `{"items":[{"item_id":"` + item + `","answer":{"option_ids":["` + st.Items[0].Options[0].ID + `"]}}]}`}, 200)
	r.must(call{method: "POST", path: att + "/submit", token: x.sv, headers: hdr(true)}, 200)
	exec(`update exams set status='CLOSED', opens_at=now() - interval '4 hours', closes_at=now() - interval '1 minute' where id=$1`, e.ID)

	// 26: hoãn công bố.
	var d struct {
		Version int `json:"version"`
	}
	_, b = r.must(call{method: "GET", path: one, token: x.gv}, 200)
	_ = json.Unmarshal(b, &d)
	hold := one + "/publish-hold"
	_, b = r.must(call{method: "PUT", path: hold, token: x.gv, body: `{"hold":true,"version":` + itoa(d.Version) + `}`}, 200)
	_ = json.Unmarshal(b, &d)
	r.must(call{method: "PUT", path: hold, token: x.gv, body: `{"hold":true,"version":` + itoa(d.Version+7) + `}`}, 409)
	r.must(call{method: "PUT", path: hold, body: `{"hold":true,"version":1}`}, 401)
	for _, who := range []string{x.ta, x.sv, x.admin} {
		r.must(call{method: "PUT", path: hold, token: who, body: `{"hold":true,"version":1}`}, 403)
	}
	r.must(call{method: "PUT", path: ex + "/" + uuid.NewString() + "/publish-hold", token: x.gv, body: `{"hold":true,"version":1}`}, 404)
	r.must(call{method: "PUT", path: hold, token: x.gv, body: `{"hold":"co","version":1}`}, 422)

	// 44: bảng điểm.
	res := one + "/results"
	h1, _ := r.must(call{method: "GET", path: res, token: x.gv}, 200)
	r.must(call{method: "GET", path: res, token: x.gv, headers: map[string]string{"If-None-Match": h1.Get("ETag")}}, 304)
	r.must(call{method: "GET", path: res + "?sort=name&status=GRADED&limit=1", token: x.ta}, 200)
	r.must(call{method: "GET", path: res}, 401)
	r.must(call{method: "GET", path: res, token: x.sv}, 403)
	r.must(call{method: "GET", path: res, token: x.admin}, 403)
	r.must(call{method: "GET", path: ex + "/" + uuid.NewString() + "/results", token: x.gv}, 404)
	r.must(call{method: "GET", path: res + "?sort=dien-thoai", token: x.gv}, 422)

	// 45: chi tiết một lượt.
	rd := res + "/" + aid
	r.must(call{method: "GET", path: rd, token: x.gv}, 200)
	r.must(call{method: "GET", path: rd, token: x.ta}, 200)
	r.must(call{method: "GET", path: rd}, 401)
	r.must(call{method: "GET", path: rd, token: x.sv}, 403)
	r.must(call{method: "GET", path: res + "/" + uuid.NewString(), token: x.gv}, 404)

	// 49: thống kê.
	r.must(call{method: "GET", path: one + "/stats", token: x.gv}, 200)
	r.must(call{method: "GET", path: one + "/stats", token: x.ta}, 200)
	r.must(call{method: "GET", path: one + "/stats"}, 401)
	r.must(call{method: "GET", path: one + "/stats", token: x.sv}, 403)
	r.must(call{method: "GET", path: ex + "/" + uuid.NewString() + "/stats", token: x.gv}, 404)

	// 50: CSV.
	csv := one + "/results.csv"
	r.must(call{method: "GET", path: csv, token: x.gv}, 200)
	r.must(call{method: "GET", path: csv}, 401)
	r.must(call{method: "GET", path: csv, token: x.sv}, 403)
	r.must(call{method: "GET", path: ex + "/" + uuid.NewString() + "/results.csv", token: x.gv}, 404)
	// 422 EXPORT_TOO_LARGE cần > 5.000 sinh viên: được miễn (exam.TestResultsCSV + hằng `csvMaxRows`).

	// 46: sửa điểm tay.
	var cur struct {
		Version int `json:"version"`
	}
	_, b = r.must(call{method: "GET", path: rd, token: x.gv}, 200)
	_ = json.Unmarshal(b, &cur)
	sc := rd + "/score"
	_, b = r.must(call{method: "PUT", path: sc, token: x.gv, body: `{"score":"1.5","reason":"chấm tay","version":` + itoa(cur.Version) + `}`}, 200)
	_ = json.Unmarshal(b, &cur)
	r.must(call{method: "PUT", path: sc, token: x.gv, body: `{"score":"2","reason":"sai version","version":` + itoa(cur.Version+9) + `}`}, 409)
	r.must(call{method: "PUT", path: sc, token: x.gv, body: `{"score":"2","reason":"","version":` + itoa(cur.Version) + `}`}, 422)
	r.must(call{method: "PUT", path: sc, token: x.gv, body: `{"score":"99","reason":"vượt","version":` + itoa(cur.Version) + `}`}, 422)
	r.must(call{method: "PUT", path: sc, body: `{"score":null,"reason":"x","version":1}`}, 401)
	r.must(call{method: "PUT", path: sc, token: x.ta, body: `{"score":null,"reason":"x","version":1}`}, 403)
	r.must(call{method: "PUT", path: res + "/" + uuid.NewString() + "/score", token: x.gv, body: `{"score":null,"reason":"x","version":1}`}, 404)
	r.must(call{method: "PUT", path: sc, token: x.gv, body: `{"score":null,"reason":"gỡ điều chỉnh","version":` + itoa(cur.Version) + `}`}, 200)

	// 47: đổi đáp án / huỷ câu.
	ov := one + "/items/" + item + "/override"
	r.must(call{method: "PUT", path: ov, token: x.gv, body: `{"void":true,"reason":"đề sai"}`}, 200)
	r.must(call{method: "PUT", path: ov, token: x.gv, body: `{"reason":"thiếu nội dung"}`}, 422)
	r.must(call{method: "PUT", path: ov, body: `{"void":true,"reason":"x"}`}, 401)
	r.must(call{method: "PUT", path: ov, token: x.ta, body: `{"void":true,"reason":"x"}`}, 403)
	r.must(call{method: "PUT", path: one + "/items/" + uuid.NewString() + "/override", token: x.gv, body: `{"void":true,"reason":"x"}`}, 404)
	r.must(call{method: "PUT", path: ex + "/" + e.ID + "x/items/" + item + "/override", token: x.gv, body: `{"void":true,"reason":"x"}`}, 404)
	// 409: bài chưa đóng.
	_, b = r.must(call{method: "POST", path: ex, token: x.gv, headers: x.idem(), body: `{"title":"Bài chưa đóng"}`}, 201)
	var dr struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(b, &dr)
	r.must(call{method: "PUT", path: ex + "/" + dr.ID + "/items/" + item + "/override", token: x.gv, body: `{"void":true,"reason":"x"}`}, 409)

	// 48: chấm lại (bài chỉ có câu trắc nghiệm → 422 NO_CODE_ITEMS; bài chưa đóng → 409).
	rg := one + "/regrade"
	r.must(call{method: "POST", path: rg, token: x.gv, headers: x.idem(), body: `{"scope":"all","reason":"đổi test"}`}, 422)
	r.must(call{method: "POST", path: ex + "/" + dr.ID + "/regrade", token: x.gv, headers: x.idem(), body: `{"scope":"all","reason":"x"}`}, 409)
	r.must(call{method: "POST", path: rg, token: x.gv, body: `{"scope":"all","reason":"x"}`}, 422) // thiếu Idempotency-Key
	r.must(call{method: "POST", path: rg, headers: x.idem(), body: `{"scope":"all","reason":"x"}`}, 401)
	r.must(call{method: "POST", path: rg, token: x.ta, headers: x.idem(), body: `{"scope":"all","reason":"x"}`}, 403)
	r.must(call{method: "POST", path: ex + "/" + uuid.NewString() + "/regrade", token: x.gv, headers: x.idem(), body: `{"scope":"all","reason":"x"}`}, 404)
	// 202: bài có câu code đã đóng — dựng ở `examCodeScenarios` (cùng helper) không có lượt MCQ; ở đây đổi loại bài sang CODE trong DB để chạm nhánh 202.
	exec(`update exams set kind='MIXED' where id=$1`, e.ID)
	r.must(call{method: "POST", path: rg, token: x.gv, headers: x.idem(), body: `{"scope":"all","reason":"đổi test"}`}, 202)
	exec(`update exams set kind='MCQ', regrading=false where id=$1`, e.ID)

	// 42: phúc khảo — chưa công bố → 409; sau công bố một lần.
	ap := att + "/appeal"
	r.must(call{method: "POST", path: ap, token: x.sv, headers: x.idem(), body: `{"reason":"Câu 1 chấm sai"}`}, 409)
	exec(`update exams set status='PUBLISHED', published_at=now(), publish_hold=false where id=$1`, e.ID)
	r.must(call{method: "POST", path: ap, token: x.sv, headers: x.idem(), body: `{"reason":""}`}, 422)
	r.must(call{method: "POST", path: ap, token: x.sv, body: `{"reason":"x"}`}, 422) // thiếu Idempotency-Key
	r.must(call{method: "POST", path: ap, headers: x.idem(), body: `{"reason":"x"}`}, 401)
	r.must(call{method: "POST", path: ap, token: x.gv, headers: x.idem(), body: `{"reason":"x"}`}, 403)
	r.must(call{method: "POST", path: one + "/attempts/" + uuid.NewString() + "/appeal", token: x.sv, headers: x.idem(), body: `{"reason":"x"}`}, 404)
	_, b = r.must(call{method: "POST", path: ap, token: x.sv, headers: x.idem(), body: `{"reason":"Câu 1 chấm sai"}`}, 201)
	var apl struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	_ = json.Unmarshal(b, &apl)
	r.must(call{method: "POST", path: ap, token: x.sv, headers: x.idem(), body: `{"reason":"lần hai"}`}, 409)
	r.must(call{method: "GET", path: att + "/result", token: x.sv}, 200)

	// 55: danh sách phúc khảo.
	al := one + "/appeals"
	r.must(call{method: "GET", path: al, token: x.gv}, 200)
	r.must(call{method: "GET", path: al + "?status=OPEN&limit=1", token: x.ta}, 200)
	r.must(call{method: "GET", path: al}, 401)
	r.must(call{method: "GET", path: al, token: x.sv}, 403)
	r.must(call{method: "GET", path: ex + "/" + uuid.NewString() + "/appeals", token: x.gv}, 404)
	r.must(call{method: "GET", path: al + "?status=KHAC", token: x.gv}, 422)

	// 56: trả lời.
	an := al + "/" + apl.ID + "/answer"
	r.must(call{method: "POST", path: an, token: x.gv, body: `{"decision":"UPHELD","response":"ok","version":` + itoa(apl.Version+5) + `}`}, 409)
	r.must(call{method: "POST", path: an, token: x.gv, body: `{"decision":"ADJUSTED","response":"ok","version":` + itoa(apl.Version) + `}`}, 422)
	r.must(call{method: "POST", path: an, body: `{"decision":"UPHELD","response":"ok","version":1}`}, 401)
	r.must(call{method: "POST", path: an, token: x.ta, body: `{"decision":"UPHELD","response":"ok","version":1}`}, 403)
	r.must(call{method: "POST", path: al + "/" + uuid.NewString() + "/answer", token: x.gv, body: `{"decision":"UPHELD","response":"ok","version":1}`}, 404)
	r.must(call{method: "POST", path: an, token: x.gv, body: `{"decision":"UPHELD","response":"Đã xem lại, giữ nguyên điểm","version":` + itoa(apl.Version) + `}`}, 200)
}
