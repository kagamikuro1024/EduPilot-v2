import re,sys,collections
from rows1 import INS,SES,FLW,SCN
from rows2 import GEO,ERR,PRM,NET,LIF
from rows3 import UIW,SAF,TXT,THM,KBD,NAV,TGT,MSG
from rows4 import A11,PWA,SEC,DEV,PRF,DAT,ATT
GROUPS=[
("A","Cài đặt và vỏ app Android (APK, quyền, ký, biểu tượng, splash)","INS",INS),
("B","Phiên đăng nhập, vai và tài khoản trên điện thoại","SES",SES),
("C","Luồng điểm danh chính (liên kết, quét, nhập mã, lặp, chọn lớp)","FLW",FLW),
("D","Quét QR trong app (WebView, camera)","SCN",SCN),
("E","Vị trí / mạng trường trên điện thoại (US-AS-10)","GEO",GEO),
("F","Nhánh lỗi, QR lạ, hết hạn, lớp khác, chưa đăng nhập, phân quyền","ERR",ERR),
("G","Độ rộng 360 / 375 / 390 / 411 / 430 px và xoay ngang (US-AS-15 AC1, AC2)","UIW",UIW),
("H","Vùng an toàn: tai thỏ, đục lỗ, thanh cử chỉ, ba nút, Home (US-AS-15 AC3)","SAF",SAF),
("I","Cỡ chữ lớn và phóng (US-AS-15 AC4)","TXT",TXT),
("J","Chế độ sáng / tối của máy (US-AS-15 AC5)","THM",THM),
("K","Bàn phím che ô nhập (US-AS-15 AC6)","KBD",KBD),
("L","Back, vuốt quay lại (US-AS-15 AC7)","NAV",NAV),
("M","Quyền camera / vị trí: từ chối rồi cấp lại (US-AS-15 AC8)","PRM",PRM),
("N","Mạng yếu, mất mạng, khôi phục (US-AS-15 AC9)","NET",NET),
("O","Nền → mở lại, hệ thống giết tiến trình, vòng đời (US-AS-15 AC10)","LIF",LIF),
("P","PWA trên iPhone Safari (US-AS-09 AC6, US-AS-15 AC11)","PWA",PWA),
("Q","Trợ năng: TalkBack, VoiceOver, axe (US-AS-15 AC12)","A11",A11),
("R","Vùng chạm ≥ 44 px (US-AS-15 AC13)","TGT",TGT),
("S","Chữ lỗi tiếng Việt, không từ kỹ thuật (US-AS-15 AC14)","MSG",MSG),
("T","Bảo mật: token, khoá, log, WebView, liên kết (luật cứng 4, 5; US-AS-11 AC1, AC5, AC7)","SEC",SEC),
("U","Thiết bị và chứng thực khoá (US-AS-12; chỉ máy thật mới chứng minh mức 3)","DEV",DEV),
("V","Hiệu năng và nhịp tải","PRF",PRF),
("W","Seed, E2E, tải, dữ liệu 30 ngày nhìn từ điện thoại (US-AS-13, US-AS-14)","DAT",DAT),
("X","Staff trên điện thoại: `/attendance` (US-AS-15 phạm vi; US-AS-08)","ATT",ATT),
]
GROUPS=[(a,b,c,sorted(g,key=lambda r:r[0])) for a,b,c,g in GROUPS]
rows=[r for _,_,_,g in GROUPS for r in g]
ids=[r[0] for r in rows]; dup=[k for k,v in collections.Counter(ids).items() if v>1]
assert not dup,dup
assert all(len(r)==8 for r in rows),[r[0] for r in rows if len(r)!=8]
# ---- ma trận AC -> TC
acre=re.compile(r'US-AS-(\d\d)\s+((?:AC\d+[a-z]?(?:\([a-d]\))?(?:\s*[,;]\s*)?)+)|(AC-\d+[a-z]?)')
cov=collections.defaultdict(list)
for r in rows:
    txt=r[1]; cur=None
    for tok in txt.split(';'):
        m=re.search(r'US-AS-(\d\d)',tok)
        if m: cur=m.group(1)
        if cur:
            seg=re.sub(r'US-AS-\d\d','',tok)
            for a in re.findall(r'(?<![-\w])AC\d+[a-z]?',seg): cov[f"US-AS-{cur} {a}"].append(r[0])
    for a in re.findall(r'AC-\d+[a-z]?',txt): cov[a].append(r[0])
ACS={ '09':['AC1','AC2','AC3','AC4','AC5','AC6','AC7','AC8','AC9','AC10'],
 '10':['AC1','AC2','AC3','AC4','AC5','AC6','AC7'],
 '11':['AC1','AC2','AC3','AC3a','AC3b','AC4','AC5','AC6','AC7','AC8'],
 '12':['AC1','AC2','AC3','AC4','AC5','AC6','AC7','AC8','AC9'],
 '13':['AC1','AC2','AC3','AC4','AC5','AC6','AC7'],
 '14':['AC1','AC2','AC3','AC4','AC5','AC6'],
 '15':[f'AC{i}' for i in range(1,16)]}
OTHER={ # AC không có thiết bị: phủ ở tệp khác hoặc ghi chú
 'US-AS-10 AC6':'API (422 hình dạng toạ độ/geofence): `tc-US-AS-10-12-13.md` TC-AS10-xx; ở điện thoại không tạo được thân sai từ giao diện',
 'US-AS-10 AC7':'phân quyền Staff/SV: `tc-US-AS-10-12-13.md`; ở điện thoại: ATT-01…10, ERR-07/08',
 'US-AS-11 AC2':'DIFF: `tc-US-AS-11-14.md` TC-AS11-04; ở điện thoại: DEV-03/05 (cùng route)',
 'US-AS-11 AC8':'chính checklist này (`android-checklist.md`) + DEV-01…14',
 'US-AS-12 AC2':'API/Go `TestAttestRejects`: `tc-US-AS-10-12-13.md`; ở điện thoại DEV-01, DEV-03, DEV-12',
 'US-AS-13 AC1':'server-only (worker purge): `tc-US-AS-10-12-13.md` TC-AS13-01…',
 'US-AS-13 AC3':'server-only: `tc-US-AS-10-12-13.md`',
 'US-AS-13 AC4':'server-only (`ForgetStudent`): `tc-US-AS-10-12-13.md`',
 'US-AS-13 AC6':'server-only (log purge): `tc-US-AS-10-12-13.md`',
 'US-AS-13 AC7':'server-only: `tc-US-AS-10-12-13.md`',
 'US-AS-14 AC6':'bảng truy vết SRS mục 11: `tc-US-AS-11-14.md`; ở điện thoại: ERR + SEC',
}
mx=[]
for us,acs in ACS.items():
    for a in acs:
        k=f"US-AS-{us} {a}"
        t=sorted(set(cov.get(k,[])))
        mx.append((k,t))
missing=[k for k,t in mx if not t and k not in OTHER]
print('AC chưa có TC thiết bị và không có ghi chú:',missing,file=sys.stderr)
CLK=[f"AC-{n}" for n in ('0','1','2','3','3b','3c','4','4b','4c','5','5b','6','7','7b','8','9','9b','10','11','12','13','14','15','16','17','18','19')]
# AC-n của checklist 10.8 -> TC (khai tường minh nếu hàng chưa ghi AC-n)
CHECK_MAP={ 'AC-0':['TC-MOB-INS-01','TC-MOB-INS-02','TC-MOB-INS-04'],'AC-1':['TC-MOB-SES-01','TC-MOB-SES-02'],'AC-2':['TC-MOB-SES-05'],
 'AC-3':['TC-MOB-FLW-01','TC-MOB-SCN-01','TC-MOB-SCN-02'],'AC-3b':['TC-MOB-SCN-03'],'AC-3c':['TC-MOB-INS-13','TC-MOB-FLW-16'],
 'AC-4':['TC-MOB-SCN-11'],'AC-4b':['TC-MOB-PRM-01'],'AC-4c':['TC-MOB-SCN-12'],'AC-5':['TC-MOB-DEV-01'],'AC-5b':['TC-MOB-DEV-07'],'AC-6':['TC-MOB-DEV-02'],
 'AC-7':['TC-MOB-DEV-03'],'AC-7b':['TC-MOB-DEV-04'],'AC-8':['TC-MOB-DEV-05'],'AC-9':['TC-MOB-DEV-06'],'AC-9b':['TC-MOB-DEV-07'],
 'AC-10':['TC-MOB-GEO-01','TC-MOB-GEO-02','TC-MOB-GEO-03','TC-MOB-GEO-04'],'AC-11':['TC-MOB-GEO-10'],'AC-12':['TC-MOB-SES-07'],'AC-13':['TC-MOB-DEV-08'],
 'AC-14':['TC-MOB-SCN-04'],'AC-15':['TC-MOB-SCN-13','TC-MOB-NET-02','TC-MOB-NET-03'],'AC-16':['TC-MOB-SCN-08'],'AC-17':['TC-MOB-SCN-16','TC-MOB-PRF-02'],
 'AC-18':['TC-MOB-UIW-01','TC-MOB-UIW-02','TC-MOB-TGT-01'],'AC-19':['TC-MOB-INS-08','TC-MOB-DEV-13']}
for a,t in CHECK_MAP.items():
    for x in t: assert x in ids,(a,x)
# ---- đếm
nA=len(rows); nMT=sum(1 for r in rows if r[7].strip()); nW=sum(1 for r in rows if r[6].startswith('E2E') or r[6] in('AXE','DIFF','API','SQL'))
# ---- viết tệp
out=[]
w=out.append
w("# QC test case — ỨNG DỤNG ĐIỆN THOẠI (app Android + PWA iPhone) — US-AS-09…15 + `android-checklist.md`")
w("")
w("Viết 25/11/2026 (qc3) theo lệnh PM / chủ dự án: testcase ĐẦY ĐỦ cho app điện thoại, nhất là UI/UX; **CHẠY sau cùng**. **Chưa chạy gì** (không có kết quả P/F trong tệp này). Nguồn: `docs/specs/FEAT-attendance-selfcheck/{US,SRS}.md` (US-AS-09…15; US-AS-15 do BA `3018e3ce`, P-11-130 CHẤP NHẬN), `docs/design/DESIGN.md` §14.8.2, §16, §23, `docs/sprints/10.8/qc/android-checklist.md`, `docs/sprints/10.8/proposals.md` (P-108-1…12), mã `mobile/` và `frontend/src/features/checkin/` @`2c0dbe83` (đọc để biết chữ, nhãn, kích thước — ca vẫn viết hộp đen theo AC). Tệp này BỔ SUNG, không thay `tc-US-AS-09.md`, `tc-US-AS-10-12-13.md`, `tc-US-AS-11-14.md` (các tệp đó phủ API / web 375 px; ở đây phủ **thiết bị thật / giả lập** và mọi khía cạnh di động).")
w("")
w("**Tổng: %d TC** trong %d nhóm; %d ca đánh dấu «chỉ máy thật / điều kiện đặc biệt» (cột cuối)." % (nA,len(GROUPS),nMT))
w("")
w("## 1. Quy ước")
w("| Mục | Quy tắc |")
w("| --- | --- |")
w("| Kết quả | **P** (đạt, có bằng chứng), **F** (trượt, ghi DEF), **N** (KHÔNG KIỂM ĐƯỢC: nêu lý do, không tính F). Không ghi P khi chưa chạy. Ca ghi «dự kiến F» là quan sát tĩnh từ mã trước khi chạy, **không** phải kết quả |")
w("| Mã TC | `TC-MOB-<nhóm>-<số>`; cột «AC» ghi AC của US hoặc bước `AC-n` của `android-checklist.md` |")
w("| MT (môi trường) | **A** = Android emulator AVD `ep-smoke` API 35; **AR** = Android máy thật; **I** = iOS Simulator 27.2, Safari; **P** = iOS Simulator, PWA đã «Thêm vào Màn hình chính»; **IR** = iPhone thật; **W** = Playwright (thiết bị di động giả, `isMobile`, `hasTouch`); **API** = gọi `curl`/python thẳng vào gateway, không cần thiết bị |")
w("| Kiểm | **TAY** (người thao tác trên thiết bị), **E2E** (Playwright), **ADB** (lệnh `adb`), **SIMCTL** (lệnh `xcrun simctl`), **AXE** (`@axe-core/playwright`), **SQL** (psql chỉ đọc), **DIFF** (đọc mã / `go test`), ghép bằng `+` |")
w("| Cột cuối | Trống = chạy được trên emulator / simulator / Playwright. Có chữ = cần thứ thiếu (máy thật, TalkBack, hai thiết bị, quyền chặn egress…) → nếu không có thì ghi **N** kèm lý do |")
w("| Bằng chứng | Ảnh chụp: `shots-mobile/<mã TC>-<thiết bị>.png`; video ngắn `.mp4`/`.mov` cho ca có thời gian; trích log đã bỏ bí mật. KHÔNG dán token, khoá, mật khẩu, toạ độ thật, IP vào báo cáo |")
w("| Thời gian chờ | Mã hết hạn thật chờ ≥ 12 s với `rotate_seconds=5`; cửa sổ hết giờ `minutes=1`; không đổi giờ máy chủ. Đổi giờ máy điện thoại chỉ ở ERR-16 rồi trả về tự động |")
w("")
w("## 2. Môi trường và thiết bị")
w("| Mã | Thiết bị | Ghi chú |")
w("| --- | --- | --- |")
w("| A | Android Virtual Device `ep-smoke`, API 35 (ghi loại image: Google APIs hoặc Play Store), camera sau `virtualscene` (poster QR) hoặc `webcam0`; ghi bản WebView, bản Chrome | APK release dựng từ `mobile/README.md`; `APP_SERVER_URL` = HTTPS thật của VPS (dev đang dựng). Emulator **không** đạt mức 3 và **không** là lỗi |")
w("| AR | Máy Android thật (ghi hãng, model, bản Android, bản WebView, bootloader khoá/mở, StrongBox/TEE) | Chủ dự án KHÔNG có máy thật (P-108-12) → ca chỉ máy thật ghi **N** trừ khi có máy |")
w("| I | iOS Simulator 27.2 (qc2 đã dựng): iPhone SE (3rd gen) 375×667, iPhone 13 mini 375×812, iPhone 15 390×844, iPhone 15 Pro Max 430×932 | Safari. Simulator KHÔNG có camera: quét QR không kiểm được ở iOS (và iOS web không quét, chỉ camera hệ thống mở liên kết) |")
w("| P | iOS Simulator, Safari → Chia sẻ → «Thêm vào Màn hình chính» | PWA standalone; kho lưu trữ tách khỏi Safari |")
w("| IR | iPhone thật nếu có | VoiceOver, ITP 7 ngày, biểu tượng thật |")
w("| W | Playwright `Pixel 7` (412×915), `iPhone 15` (393×852) và các viewport 360/375/390/411/430 × 640–932 và ngang 667×375, 844×390, 915×412 | chạy trên bản build (`next start`) hoặc stack thử |")
w("")
w("| Độ rộng (CSS px) | Android (R-SIZE) | iOS | Playwright |")
w("| --- | --- | --- | --- |")
w("| 360 × 640–800 | `wm size 1080x2400`, `wm density 480` | – (không có) | 360×640, 360×800 |")
w("| 375 × 667–812 | `wm size 1125x2436`, `wm density 480` | iPhone SE 375×667, iPhone 13 mini 375×812 | 375×667, 375×812 |")
w("| 390 × 844 | `wm size 1170x2532`, `wm density 480` | iPhone 15 | 390×844 |")
w("| 411 × 891 | `wm size 1080x2340`, `wm density 420` | – | 411×891 |")
w("| 430 × 932 | `wm size 1290x2796`, `wm density 480` | iPhone 15 Pro Max | 430×932 |")
w("| Ngang | xoay `R-ROT` ở 667×375, 844×390, 915×412 | Simulator ⌘← / ⌘→ | cùng ba kích thước |")
w("")
w("## 3. Công thức lệnh (tham chiếu trong các bước)")
w("| Tên | Lệnh / thao tác |")
w("| --- | --- |")
for k,v in [
("R-ENV","`emulator -avd ep-smoke -no-snapshot-load -camera-back virtualscene`; cài APK: `cd mobile && npm ci && APP_SERVER_URL=https://{VPS} npm run install:device`"),
("R-CAM","poster QR cho camera ảo: thêm ảnh PNG vào `$ANDROID_HOME/emulator/resources/Toren1BD.posters` (mục `poster wall` → `default <ảnh>.png`); QR động của màn chiếu: `-camera-back webcam0` hướng vào màn laptop; ảnh poster dựng sẵn: Q-OK (đúng, đang hạn), Q-EXPIRED, Q-ALIEN, Q-OTHER-ORIGIN, Q-PATH, Q-NOUUID, Q-NOHASH, Q-OTHERCLASS, Q-CLOSED"),
("R-SIZE-w","`adb shell wm size <W×H px>`; `adb shell wm density <dpi>` theo bảng ở mục 2; trả: `wm size reset`; `wm density reset`"),
("R-DENSITY","`adb shell wm density 320` (hiển thị lớn) và `wm density 560` (nhỏ)"),
("R-FONT-x","`adb shell settings put system font_scale 1.3` rồi `2.0`; trả `1.0`"),
("R-DARK-on/off","`adb shell cmd uimode night yes` / `no`"),
("R-ROT-land","`adb shell settings put system accelerometer_rotation 0; adb shell settings put system user_rotation 1` (ngang); `0` (dọc)"),
("R-CUT-tall/corner/double/hole","`adb shell cmd overlay enable com.android.internal.display.cutout.emulation.<tall|corner|double|hole|narrow|wide>`; tắt: `overlay disable …`"),
("R-NAV-gesture / R-NAV-3btn","`adb shell cmd overlay enable com.android.internal.systemui.navbar.gestural` / `…navbar.threebutton`"),
("R-PERM","`adb shell pm revoke vn.edupilot.app android.permission.CAMERA`; `pm grant …`; `pm reset-permissions`; mở Cài đặt ứng dụng: `adb shell am start -a android.settings.APPLICATION_DETAILS_SETTINGS -d package:vn.edupilot.app`"),
("R-NET-off","`adb shell cmd connectivity airplane-mode enable` / `disable`"),
("R-NET-slow","`adb emu network speed 400:400` và `adb emu network delay 1000` (≈ RTT 2 s); trả: `network speed full`, `network delay none`"),
("R-NET-flaky","vòng lặp `adb shell svc wifi disable; sleep 2; adb shell svc wifi enable; sleep 2` ×5 (kèm `svc data`)"),
("R-NET-block-host","chặn máy chủ app bằng proxy phía máy chủ chạy (mitmproxy rule) hoặc nhờ dev tạm đóng cổng 443 của VPS thử; trả lại sau khi xong"),
("R-KILL","`am force-stop vn.edupilot.app`; nền: `input keyevent KEYCODE_HOME` rồi `am kill vn.edupilot.app`; ép bộ nhớ thấp: `am send-trim-memory vn.edupilot.app COMPLETE`; Doze: `dumpsys deviceidle force-idle` / `unforce`"),
("R-LINK","`adb shell am start -a android.intent.action.VIEW -d \"$U\" com.android.chrome` (như camera hệ thống mở liên kết)"),
("R-GEO","`adb emu geo fix <kinh độ> <vĩ độ>`; vị trí giả: `adb shell appops set <gói app GPS giả> android:mock_location allow`"),
("R-SHOT","`adb exec-out screencap -p > x.png`; quay: `adb shell screenrecord --time-limit 30 /sdcard/x.mp4`; chụp liên tiếp: vòng lặp `screencap` ≥ 5 ảnh/s"),
("R-DUMP","`adb shell uiautomator dump /sdcard/ui.xml` rồi `adb pull`; đọc `bounds`, `content-desc`, `text`"),
("R-LOGCAT","`adb logcat -d -v time`; lọc bí mật: `grep -iE 'access_token|refresh|ep_rt|signature|public_key|Bearer|token='`"),
("R-CALL","`adb emu gsm call 0901234567`; `adb emu gsm cancel 0901234567`"),
("S-SIM","`xcrun simctl list devices available`; `xcrun simctl boot \"iPhone 15\"` (và `iPhone SE (3rd generation)`, `iPhone 13 mini`, `iPhone 15 Pro Max`)"),
("S-URL","`xcrun simctl openurl booted \"$U\"` (như camera hệ thống mở liên kết; luôn mở Safari)"),
("S-DARK","`xcrun simctl ui booted appearance dark` / `light`"),
("S-CONTENT-SIZE","`xcrun simctl ui booted content_size accessibility-extra-extra-extra-large`; trả `large`"),
("S-PRIVACY","`xcrun simctl privacy booted grant|revoke|reset location com.apple.mobilesafari`"),
("S-LOC","`xcrun simctl location booted set <vĩ độ>,<kinh độ>`"),
("S-SHOT","`xcrun simctl io booted screenshot x.png`; quay: `xcrun simctl io booted recordVideo x.mov`; thanh trạng thái sạch: `xcrun simctl status_bar booted override --time 9:41 --batteryLevel 100`"),
("S-KILL","`xcrun simctl terminate booted com.apple.mobilesafari`"),
("S-NET","Network Link Conditioner trên máy Mac (hồ sơ 3G/100 % mất gói), hoặc Safari → Develop → Network → Throttling"),
("S-CA","chỉ khi VPS dùng CA nội bộ: `xcrun simctl keychain booted add-root-cert <gốc>.pem` (HTTPS thật của VPS thì không cần)"),
("E2E","`pnpm -C frontend exec playwright test e2e/attendance-checkin.spec.ts --project=mobile` (cần project `mobile` hoặc thêm dự án `isMobile`); script kiểm kích thước riêng: `e2e/mobile-ui.spec.ts` (QC viết khi chạy)"),
]: w("| %s | %s |"%(k,v))
w("")
w("## 4. Dữ liệu")
w("| Tên | Nội dung |")
w("| --- | --- |")
for k,v in [
("VPS","`https://{VPS}` = HTTPS thật do dev dựng (ghi host khi chạy; đổi host ⇒ dựng lại và cài lại APK). Mật khẩu seed lấy từ biến môi trường của đợt chạy, không ghi vào báo cáo"),
("Tài khoản","SV: `sv.gioi` (SV1), `sv.kha` (SV2), `sv.nguyco` (ngoài lớp), `sv.moi` (chưa vào lớp nào); Staff: `teacher@`, `ta@`; `admin@` (đều `@edupilot.local`)"),
("Lớp","**QCMOB1**: `self_checkin` bật, không geofence; **QCMOB2**: `geofence {lat,lng,radius_m}` bật + `campus_cidrs`; **QCMOB3**: `require=APP`; **QCMOB4**: lưu trữ; **QCMOB5**: `self_checkin` tắt. SV1, SV2 thuộc QCMOB1–3; `sv.nguyco` không thuộc lớp nào trong số đó. Không đụng 761987/761988 (trừ DAT-01)"),
("Cửa sổ","W: cửa sổ đang mở, `rotate_seconds=10` (đủ cho quét tay) hoặc `5` (cho ca hết hạn); W60: `rotate_seconds=60` (poster QR sống ≤ 120 s); W-closed: đã đóng; W-old: token cũ > 2 bước; vòng 2 mở sau vòng 1"),
("QR / liên kết","`U = https://{VPS}/checkin?c={course_id}#{token}`; mã ngắn 6 ký tự hiện dưới QR của màn chiếu (`/attendance/display` trên laptop ≥ 1280×720)"),
("Phụ trợ","app GPS giả (cho GEO-10), ảnh QR dựng sẵn (R-CAM), proxy chậm/chặn (R-NET-*), `psql` chỉ đọc vào `ep-qc` / stack thử"),
]: w("| %s | %s |"%(k,v))
w("")
w("## 5. Quan sát tĩnh TRƯỚC khi chạy (đọc mã / tài nguyên ngày 25/11; chưa phải kết quả)")
w("Những điểm sau suy ra từ tệp trong repo @`2c0dbe83`; ca tương ứng sẽ xác nhận hay bác bỏ khi chạy. Nếu xác nhận → mở DEF.")
w("| # | Quan sát | Ca xác nhận |")
w("| --- | --- | --- |")
for i,(a,b) in enumerate([
("Biểu tượng launcher (`mipmap-*/ic_launcher*.png`) và màn splash (`drawable*/splash.png`) là bộ mặc định của Capacitor (chữ X xanh), không phải logo EduPilot đỏ – trắng (đã mở ảnh đọc)","INS-05, INS-06, THM-05"),
("`AndroidManifest.xml` chỉ khai `ACCESS_FINE_LOCATION`, không khai `ACCESS_COARSE_LOCATION`; Android 12+ xin cả hai cùng lúc nên hộp xin vị trí có thể bị từ chối ngay trong app","GEO-05, GEO-08, PRM-10"),
("`layout.tsx` không đặt `viewport-fit=cover`, không `apple-touch-icon`, không `appleWebApp` / `theme-color` metadata (chỉ `manifest.ts` có icon PNG 192/512 và `theme_color`)","SAF-05, SAF-06, PWA-03, PWA-05, THM-07"),
("Toàn app chỉ giao diện sáng (`color-scheme: light`, D59); `error.html` không đặt nền (chữ `#1a1a1a`) — dễ chữ tối trên nền tối nếu WebView ép tối","THM-01…05"),
("`targetSdk 36`: từ Android 15 giao diện vẽ tràn dưới thanh trạng thái / thanh điều hướng (edge-to-edge) — mọi `env(safe-area-inset-*)` phải đúng","SAF-01…04"),
("Manifest không đặt `windowSoftInputMode`; dock `/checkin` là `position: fixed` — bàn phím có thể che nút `Điểm danh`","KBD-01, KBD-02, TXT-08"),
("Ngưỡng 44 px chỉ bật ở `(max-width: 719px), (pointer: coarse)`; điện thoại ngang ≥ 720 px vẫn có `pointer: coarse` nên đạt, nhưng khung chuyển sang thanh bên","TGT-06, UIW-17"),
("iOS Safari không có `BarcodeDetector` và Simulator không có camera: quét QR chỉ kiểm được trên Android; iOS dùng ô nhập mã + camera hệ thống mở Safari","FLW-04, PWA-08, PWA-09"),
],1): w("| %d | %s | %s |"%(i,a,b))
w("")
for g,title,code,lst in GROUPS:
    w("## 6.%s %s"%(g,title))
    w("| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |")
    w("| --- | --- | --- | --- | --- | --- | --- | --- |")
    for r in lst: w("| "+" | ".join(c.replace("\n"," ").replace("|","\\|") for c in r)+" |")
    w("")
w("## 7. Truy vết AC → TC")
w("| AC | TC thiết bị trong tệp này |")
w("| --- | --- |")
for k,t in mx:
    if t: w("| %s | %s |"%(k,", ".join(x.replace('TC-MOB-','') for x in t)))
    else: w("| %s | — (%s) |"%(k,OTHER.get(k,'chưa có')))
w("")
w("### Bước `AC-n` của `android-checklist.md` → TC")
w("| Bước | TC |")
w("| --- | --- |")
for a in CLK:
    t=CHECK_MAP.get(a)
    w("| %s | %s |"%(a,", ".join(x.replace('TC-MOB-','') for x in t) if t else "—"))
w("")
w("## 8. Thứ tự chạy gợi ý (PM xếp lượt cuối)")
w("1. **W + API + DIFF** (không cần thiết bị, ~1 giờ): E2E 5 độ rộng × 6 trạng thái, đo vùng chạm, axe, MSG-15, SEC-07/14/15/18, PWA-01/16, DEV-10…12, PRF-04, DAT-03…07.")
w("2. **Emulator A theo nhóm trạng thái** để đỡ đổi cấu hình: (a) cài đặt, phiên, luồng, quét, lỗi; (b) đổi độ rộng 360 → 375 → 390 → 411 → 430 (UIW, SAF, TXT, TGT); (c) ngang; (d) tối/sáng; (e) bàn phím; (f) quyền; (g) mạng; (h) vòng đời; (i) bảo mật; (j) thiết bị/chứng thực. Sau mỗi nhóm trả cấu hình (R-SIZE reset, font 1.0, night no, rotation 0, network full).")
w("3. **iOS Simulator I và P**: liên kết, nhập mã, vị trí, an toàn vùng, PWA, trợ năng (VoiceOver nếu bật được).")
w("4. **Máy thật** (nếu có): DEV-01/02/07/08, GEO-10, SEC-13, SCN-16, PWA-13, A11-10/11.")
w("Ước lượng: W ~1 giờ; emulator ~6 giờ; iOS ~2 giờ; máy thật ~2 giờ. Số ca: %d." % nA)
w("")
w("## 9. Bảng kết quả (điền khi chạy)")
w("| Mã | Thiết bị / OS / WebView | KQ (P/F/N) | Ghi chú / ảnh / DEF |")
w("| --- | --- | --- | --- |")
w("| (mỗi mã ở mục 6) | | | |")
w("")
w("## 10. Ghi chú")
w("- Chữ UI theo luật cứng PO (09/10/2026): ca KHÔNG đòi câu giải thích mới; chữ lỗi chỉ theo bảng US-AS-09 AC4/AC10 và US-AS-15 AC14.")
w("- Không có ca nào kiểm Play Integrity, App Attest, thông báo đẩy, đa cửa sổ, máy tính bảng, màn gập, chế độ tối thật (ngoài phạm vi US-AS-15).")
w("- Ca chỉ máy thật / điều kiện: Keystore phần cứng (SEC-13, DEV-01/02/07/08), `GEO_MOCK` thật (GEO-10), PWA 8 ngày (PWA-13), VoiceOver thật (A11-10/11), TalkBack (A11-02…08 nếu AVD không có), hai thiết bị (FLW-15), danh sách thu hồi Google bị chặn (DEV-04), AVD API cũ (INS-09), mạng trường (GEO-13).")
open('/Users/kuro/Documents/TA_Agent_qc3/docs/sprints/10.8/qc/tc-mobile-app.md','w',encoding='utf8').write('\n'.join(out)+'\n')
print('rows',nA,'MT',nMT,'groups',len(GROUPS))
