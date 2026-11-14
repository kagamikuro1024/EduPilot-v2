// PoC điểm danh tự phục vụ (research 2026-11-14-attendance-anti-cheat.md). Chỉ thư viện chuẩn Go.
// Chạy: cd docs/research/poc/attendance-checkin && go run .
//  1. Mã QR xoay vòng: HMAC theo cửa sổ điểm danh, không lưu trạng thái; kiểm sai lệch, hết hạn, giả mạo, sai cửa sổ.
//  2. Chữ ký khoá thiết bị ECDSA P-256 (WebCrypto trả r||s 64 byte, Android Keystore trả DER): kiểm cả hai dạng.
//  3. Android Key Attestation: kiểm chuỗi chứng chỉ tới gốc Google + đọc KeyDescription (rootOfTrust, gói ứng dụng)
//     bằng crypto/x509 + encoding/asn1, không thư viện ngoài. Dữ liệu mẫu: github.com/android/keyattestation/testdata.
//  4. Đo: 200 lượt check-in (kiểm mã + kiểm chữ ký) chạy song song.
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------- 1. Mã QR xoay vòng ----------

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

type window struct {
	ID       string // uuid cửa sổ điểm danh (một buổi có thể có vòng 1, vòng 2)
	OpenedAt time.Time
	Rotate   time.Duration
}

// key của cửa sổ suy ra từ khoá máy chủ: không lưu bí mật nào theo buổi, gateway nào cũng tính được.
func windowKey(master []byte, wid string) []byte {
	m := hmac.New(sha256.New, master)
	m.Write([]byte("att-v1|" + wid))
	return m.Sum(nil)
}

func mac(k []byte, step int64) string {
	m := hmac.New(sha256.New, k)
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(step))
	m.Write(b[:])
	return b32.EncodeToString(m.Sum(nil)[:10]) // 80 bit, 16 ký tự
}

func issue(master []byte, w window, now time.Time) string {
	step := int64(now.Sub(w.OpenedAt) / w.Rotate)
	return fmt.Sprintf("%s.%d.%s", w.ID, step, mac(windowKey(master, w.ID), step))
}

var errExpired, errBad = errors.New("TOKEN_EXPIRED"), errors.New("TOKEN_INVALID")

// verify nhận bước hiện tại và bước liền trước (tuổi mã tối đa 2×Rotate); trả tuổi mã để gắn cờ quét muộn.
func verify(master []byte, w window, tok string, now time.Time) (age time.Duration, err error) {
	p := strings.Split(tok, ".")
	if len(p) != 3 || p[0] != w.ID {
		return 0, errBad
	}
	step, err := strconv.ParseInt(p[1], 10, 64)
	if err != nil || !hmac.Equal([]byte(p[2]), []byte(mac(windowKey(master, w.ID), step))) {
		return 0, errBad
	}
	cur := int64(now.Sub(w.OpenedAt) / w.Rotate)
	if step > cur || cur-step > 1 {
		return 0, errExpired
	}
	return now.Sub(w.OpenedAt.Add(time.Duration(step) * w.Rotate)), nil
}

// ---------- 2. Chữ ký khoá thiết bị ----------

// verifySig nhận chữ ký DER (Android Keystore, iOS SecKey) hoặc r||s 64 byte (WebCrypto).
func verifySig(pub *ecdsa.PublicKey, msg, sig []byte) bool {
	h := sha256.Sum256(msg)
	if len(sig) == 64 {
		r, s := new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])
		return ecdsa.Verify(pub, h[:], r, s)
	}
	return ecdsa.VerifyASN1(pub, h[:], sig)
}

func checkinMsg(tok, uid string) []byte { return []byte("edupilot-checkin-v1|" + tok + "|" + uid) }

// ---------- 3. Android Key Attestation ----------

var oidKeyDesc = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 1, 17}

type keyDescription struct {
	AttestationVersion       int
	AttestationSecurityLevel asn1.Enumerated
	KeyMintVersion           int
	KeyMintSecurityLevel     asn1.Enumerated
	AttestationChallenge     []byte
	UniqueID                 []byte
	SoftwareEnforced         asn1.RawValue
	HardwareEnforced         asn1.RawValue
}

type rootOfTrust struct {
	VerifiedBootKey   []byte
	DeviceLocked      bool
	VerifiedBootState asn1.Enumerated // 0 Verified, 1 SelfSigned, 2 Unverified, 3 Failed
	VerifiedBootHash  []byte `asn1:"optional"`
}

type pkgInfo struct {
	Name    []byte
	Version int
}
type appID struct {
	Packages   []pkgInfo `asn1:"set"`
	Signatures [][]byte  `asn1:"set"`
}

// field trả phần tử có thẻ ngữ cảnh `tag` trong AuthorizationList (thẻ số lớn như 704, 709).
func field(list asn1.RawValue, tag int) (asn1.RawValue, bool) {
	rest := list.Bytes
	for len(rest) > 0 {
		var v asn1.RawValue
		var err error
		if rest, err = asn1.Unmarshal(rest, &v); err != nil {
			return v, false
		}
		if v.Class == asn1.ClassContextSpecific && v.Tag == tag {
			var inner asn1.RawValue
			_, err := asn1.Unmarshal(v.Bytes, &inner) // thẻ EXPLICIT bọc giá trị thật
			return inner, err == nil
		}
	}
	return asn1.RawValue{}, false
}

var bootState = map[asn1.Enumerated]string{0: "VERIFIED", 1: "SELF_SIGNED", 2: "UNVERIFIED", 3: "FAILED"}
var secLevel = map[asn1.Enumerated]string{0: "SOFTWARE", 1: "TRUSTED_ENVIRONMENT", 2: "STRONG_BOX"}

func attest(pemPath string, roots *x509.CertPool) error {
	raw, err := os.ReadFile(pemPath)
	if err != nil {
		return err
	}
	var chain []*x509.Certificate
	for b, rest := pem.Decode(raw); b != nil; b, rest = pem.Decode(rest) {
		c, err := x509.ParseCertificate(b.Bytes)
		if err != nil {
			return err
		}
		chain = append(chain, c)
	}
	leaf, inter := chain[0], x509.NewCertPool()
	for _, c := range chain[1:] {
		inter.AddCert(c)
	}
	// Lá Keystore có NotBefore = 1970; chứng chỉ trung gian RKP sống vài ngày. Sản phẩm kiểm tại time.Now() ngay khi đăng ký thiết bị;
	// PoC dùng dữ liệu mẫu cũ nên kiểm tại "ngay sau khi chuỗi được cấp" = NotBefore muộn nhất trong chuỗi + 1 phút.
	at := leaf.NotBefore
	for _, c := range chain {
		if c.NotBefore.After(at) {
			at = c.NotBefore
		}
	}
	_, verr := leaf.Verify(x509.VerifyOptions{Roots: roots, Intermediates: inter, CurrentTime: at.Add(time.Minute),
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}})
	var kd keyDescription
	for _, e := range leaf.Extensions {
		if e.Id.Equal(oidKeyDesc) {
			if _, err := asn1.Unmarshal(e.Value, &kd); err != nil {
				return fmt.Errorf("KeyDescription: %w", err)
			}
		}
	}
	var rot rootOfTrust
	if v, ok := field(kd.HardwareEnforced, 704); ok {
		if _, err := asn1.Unmarshal(v.FullBytes, &rot); err != nil {
			return fmt.Errorf("rootOfTrust: %w", err)
		}
	}
	var app appID
	if v, ok := field(kd.SoftwareEnforced, 709); ok { // OCTET STRING chứa DER của AttestationApplicationId
		if _, err := asn1.Unmarshal(v.Bytes, &app); err != nil {
			return fmt.Errorf("appId: %w", err)
		}
	}
	pkg := ""
	if len(app.Packages) > 0 {
		pkg = string(app.Packages[0].Name)
	}
	fmt.Printf("  %-32s chain→GoogleRoot=%v  level=%s  boot=%s locked=%v  challenge=%q  pkg=%s  sigs=%d\n",
		pemPath[strings.LastIndex(pemPath, "/")+1:], verr == nil, secLevel[kd.AttestationSecurityLevel],
		bootState[rot.VerifiedBootState], rot.DeviceLocked, kd.AttestationChallenge, pkg, len(app.Signatures))
	if verr != nil {
		fmt.Println("    verify:", verr)
	}
	return nil
}

func main() {
	master := make([]byte, 32)
	rand.Read(master)
	t0 := time.Date(2026, 11, 14, 7, 0, 0, 0, time.UTC)
	w := window{ID: "019a8c2e-7f00-7000-8000-000000000001", OpenedAt: t0, Rotate: 10 * time.Second}

	fmt.Println("1. Mã QR xoay vòng (Rotate = 10 s)")
	tok := issue(master, w, t0.Add(3*time.Second))
	url := "https://edupilot.example/a#" + tok
	fmt.Printf("  mã=%s  URL QR=%d ký tự\n", tok, len(url))
	for _, c := range []struct {
		name string
		tok  string
		at   time.Duration
	}{
		{"quét ngay (4 s)", tok, 4 * time.Second},
		{"quét ở bước sau (14 s)", tok, 14 * time.Second},
		{"ảnh gửi bạn, dùng sau 21 s", tok, 21 * time.Second},
		{"sửa MAC", tok[:len(tok)-1] + "A", 4 * time.Second},
		{"bước tương lai", fmt.Sprintf("%s.%d.%s", w.ID, 5, mac(windowKey(master, w.ID), 5)), 4 * time.Second},
		{"mã của cửa sổ khác", strings.Replace(tok, "0001", "0002", 1), 4 * time.Second},
	} {
		age, err := verify(master, w, c.tok, t0.Add(c.at))
		fmt.Printf("  %-28s → err=%v tuổi=%v\n", c.name, err, age)
	}

	fmt.Println("2. Chữ ký khoá thiết bị P-256 (DER và r||s)")
	dev, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	msg := checkinMsg(tok, "user-A")
	h := sha256.Sum256(msg)
	der, _ := ecdsa.SignASN1(rand.Reader, dev, h[:])
	r, s, _ := ecdsa.Sign(rand.Reader, dev, h[:])
	raw := append(r.FillBytes(make([]byte, 32)), s.FillBytes(make([]byte, 32))...)
	fmt.Printf("  DER ok=%v  r||s ok=%v  chữ ký đem sang tài khoản B ok=%v\n",
		verifySig(&dev.PublicKey, msg, der), verifySig(&dev.PublicKey, msg, raw), verifySig(&dev.PublicKey, checkinMsg(tok, "user-B"), der))

	fmt.Println("3. Android Key Attestation (gốc từ android.googleapis.com/attestation/root)")
	var rootPEMs []string
	rj, _ := os.ReadFile("testdata/roots.json")
	json.Unmarshal(rj, &rootPEMs)
	roots := x509.NewCertPool()
	for _, p := range rootPEMs {
		roots.AppendCertsFromPEM([]byte(p))
	}
	for _, f := range []string{"testdata/caiman_sdk36_SB_EC_RKP.pem", "testdata/akita_sdk34_TEE_EC_NONE.pem"} {
		if err := attest(f, roots); err != nil {
			fmt.Println("  lỗi:", err)
		}
	}
	// Chuỗi giả: cùng lá thật nhưng thiếu trung gian / gốc tự ký → không tới gốc Google.
	fk, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tpl := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: t0, NotAfter: t0.Add(time.Hour), IsCA: true, BasicConstraintsValid: true}
	fder, _ := x509.CreateCertificate(rand.Reader, tpl, tpl, &fk.PublicKey, fk)
	fake, _ := x509.ParseCertificate(fder)
	_, ferr := fake.Verify(x509.VerifyOptions{Roots: roots, CurrentTime: t0.Add(time.Minute)})
	fmt.Printf("  chứng chỉ tự ký giả làm 'thiết bị': chain→GoogleRoot=%v (%v)\n", ferr == nil, ferr)

	fmt.Println("4. Tải: 200 lượt check-in song song (kiểm mã + chữ ký), lặp 50 lần")
	type req struct {
		tok string
		sig []byte
		pub *ecdsa.PublicKey
		uid string
	}
	reqs := make([]req, 200)
	for i := range reqs {
		k, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		uid := fmt.Sprintf("u%03d", i)
		hh := sha256.Sum256(checkinMsg(tok, uid))
		sig, _ := ecdsa.SignASN1(rand.Reader, k, hh[:])
		reqs[i] = req{tok, sig, &k.PublicKey, uid}
	}
	best := time.Hour
	for range 50 {
		start := time.Now()
		var wg sync.WaitGroup
		for _, q := range reqs {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := verify(master, w, q.tok, t0.Add(5*time.Second)); err != nil || !verifySig(q.pub, checkinMsg(q.tok, q.uid), q.sig) {
					panic("verify")
				}
			}()
		}
		wg.Wait()
		best = min(best, time.Since(start))
	}
	fmt.Printf("  200 lượt: %v (tốt nhất trong 50 lần) → phần mật mã không phải nút cổ chai; chi phí thật là 1 INSERT / lượt\n", best)
}
