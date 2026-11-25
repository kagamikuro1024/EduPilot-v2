import XCTest

// Điều khiển Safari / Màn hình chính của iOS Simulator cho TC-AS09-34. Tham số qua biến môi trường TEST_RUNNER_*.
final class IosTests: XCTestCase {
    let safari = XCUIApplication(bundleIdentifier: "com.apple.mobilesafari")
    let springboard = XCUIApplication(bundleIdentifier: "com.apple.springboard")
    func dump(_ app: XCUIApplication, _ name: String) {
        try? app.debugDescription.write(toFile: "/Users/kuro/.config/edupilot-team/qc2-work/ios-tree-\(name).txt", atomically: true, encoding: .utf8)
    }
    func testDumpSafari() {
        safari.launch(); sleep(3); dump(safari, "safari")
    }

    /// Safari đang ở trang /login: nhập tài khoản và bấm Đăng nhập. Env: TEST_RUNNER_EMAIL, TEST_RUNNER_PW.
    /// Ô mật khẩu: bấm nút «hiện mật khẩu» (ô thành text thường) rồi gõ; nếu không có thì bấm ô và gõ.
    func testLogin() {
        let env = ProcessInfo.processInfo.environment
        safari.activate()
        let web = safari.webViews.firstMatch
        XCTAssertTrue(web.waitForExistence(timeout: 20))
        let email = web.textFields.firstMatch
        XCTAssertTrue(email.waitForExistence(timeout: 20), "không thấy ô email")
        email.tap(); sleep(2)
        email.typeText(env["EMAIL"] ?? "")
        XCTAssertEqual(email.value as? String, env["EMAIL"] ?? "", "email gõ thiếu ký tự")
        sleep(1); dump(safari, "login-email")
        let next = safari.buttons["Next"]
        XCTAssertTrue(next.waitForExistence(timeout: 10), "không thấy nút Next của thanh bàn phím")
        next.tap(); sleep(2)
        safari.typeText((env["PW"] ?? "") + "\n")
        sleep(6); dump(safari, "after-login")
    }
    func testDumpWeb() { safari.activate(); sleep(2); dump(safari, "web") }

    /// Đóng hộp «Save Password?» (và hộp tương tự) nếu có.
    func testDismiss() {
        safari.activate()
        let b = safari.buttons["Not Now"]
        if b.waitForExistence(timeout: 8) { b.tap() }
        sleep(1)
    }

    func tapAny(_ app: XCUIApplication, _ labels: [String], timeout: Double = 6) -> Bool {
        for l in labels {
            let pred = NSPredicate(format: "label CONTAINS[c] %@", l)
            for q in [app.buttons, app.cells, app.staticTexts, app.menuItems, app.otherElements] {
                let e = q.matching(pred).firstMatch
                if e.waitForExistence(timeout: timeout / Double(labels.count)) && e.isHittable { e.tap(); return true }
            }
        }
        return false
    }
    /// Safari → Thêm vào Màn hình chính (Share → Add to Home Screen → Add). Ghi cây sau mỗi bước để dò.
    func testAddToHome() {
        safari.activate(); sleep(2)
        safari.buttons["MoreMenuButton"].tap(); sleep(2); dump(safari, "menu1")
        XCTAssertTrue(tapAny(safari, ["Share"]), "không thấy Share"); sleep(3); dump(safari, "share")
        // bảng chia sẻ nằm ở tiến trình khác; kéo lên để thấy hết
        sleep(4); dump(safari, "share2")
        let box = safari.otherElements["ShareSheet.RemoteContainerView"]
        if !safari.staticTexts["Add to Home Screen"].exists && box.exists { box.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: 0.8)).press(forDuration: 0.1, thenDragTo: box.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: 0.2))) }
        sleep(2); dump(safari, "share3")
        XCTAssertTrue(tapAny(safari, ["Add to Home Screen"], timeout: 8), "không thấy Add to Home Screen"); sleep(3); dump(safari, "addhome")
        XCTAssertTrue(tapAny(safari, ["Add"], timeout: 8), "không thấy nút Add"); sleep(3); dump(safari, "added")
    }

    // ---- PWA (Màn hình chính, bundle com.apple.webapp) ----
    let pwa = XCUIApplication(bundleIdentifier: "com.apple.webapp")
    func openPwa() {
        if pwa.state == .runningForeground { return }
        XCUIDevice.shared.press(.home); sleep(1)
        // biểu tượng EduPilot nằm ở hàng 3, cột 1 của Màn hình chính (07-after-add-to-home.png); springboard không cho frame ⇒ bấm theo toạ độ chuẩn hoá
        let dd = springboard.coordinate(withNormalizedOffset: CGVector(dx: 0.9, dy: 0.5))
        dd.press(forDuration: 0.1, thenDragTo: springboard.coordinate(withNormalizedOffset: CGVector(dx: 0.1, dy: 0.5))); sleep(2)
        dump(springboard, "home-page2")
        springboard.coordinate(withNormalizedOffset: CGVector(dx: 0.154, dy: 0.37)).tap(); sleep(6)
        XCTAssertTrue(pwa.wait(for: .runningForeground, timeout: 20), "PWA không chạy")
    }
    func typeInto(_ app: XCUIApplication, field: XCUIElement, _ text: String) {
        field.tap(); sleep(2); field.typeText(text)
    }
    /// Đăng nhập trong PWA (kho lưu trữ riêng, chưa có phiên).
    func testPwaLogin() {
        let env = ProcessInfo.processInfo.environment
        openPwa(); dump(pwa, "pwa-start")
        let web = pwa.webViews.firstMatch
        XCTAssertTrue(web.waitForExistence(timeout: 20))
        let email = web.textFields.firstMatch
        if !email.waitForExistence(timeout: 4) {
            let lb = web.buttons.matching(NSPredicate(format: "label CONTAINS 'Đăng nhập'")).firstMatch
            let ll = web.links.matching(NSPredicate(format: "label CONTAINS 'Đăng nhập'")).firstMatch
            if lb.exists { lb.tap() } else if ll.exists { ll.tap() }
            sleep(4)
        }
        if email.waitForExistence(timeout: 15) {
            typeInto(pwa, field: email, env["EMAIL"] ?? "")
            XCTAssertEqual(email.value as? String, env["EMAIL"] ?? "", "email gõ thiếu ký tự")
            let next = pwa.buttons["Next"]
            XCTAssertTrue(next.waitForExistence(timeout: 10)); next.tap(); sleep(2)
            pwa.typeText((env["PW"] ?? "") + "\n"); sleep(6)
        }
        let nn = pwa.buttons["Not Now"]; if nn.waitForExistence(timeout: 6) { nn.tap() }
        sleep(2); dump(pwa, "pwa-after-login")
    }
    /// Trong PWA: vào màn Điểm danh (Thêm → Điểm danh), gõ mã đọc từ code.txt, bấm Điểm danh.
    func testPwaCheckin() {
        openPwa(); sleep(2)
        let web = pwa.webViews.firstMatch
        XCTAssertTrue(web.waitForExistence(timeout: 20))
        if !web.textFields.firstMatch.exists {
            // lối vào: ô «Tìm nhanh hoặc đi đến…» trên thanh trên (màn Hôm nay không có mục Điểm danh)
            XCTAssertTrue(tapAny(pwa, ["Tìm nhanh"], timeout: 8), "không thấy ô tìm nhanh"); sleep(2)
            pwa.typeText("Điểm danh"); sleep(2); dump(pwa, "pwa-search")
            XCTAssertTrue(tapAny(pwa, ["Điểm danh"], timeout: 8), "không thấy kết quả Điểm danh"); sleep(3)
        }
        dump(pwa, "pwa-checkin-form")
        let code = web.textFields.firstMatch
        XCTAssertTrue(code.waitForExistence(timeout: 10), "không thấy ô mã")
        let raw = ProcessInfo.processInfo.environment["PWA_CODE"].flatMap { $0.isEmpty ? nil : $0 } ?? ((try? String(contentsOfFile: "/Users/kuro/.config/edupilot-team/qc2-work/code.txt", encoding: .utf8)) ?? "")
        typeInto(pwa, field: code, raw.trimmingCharacters(in: .whitespacesAndNewlines))
        dump(pwa, "pwa-code-typed")
        let b = web.buttons.matching(NSPredicate(format: "label == 'Điểm danh'")).firstMatch
        if b.exists { b.tap() } else { pwa.typeText("\n") }
        sleep(5); dump(pwa, "pwa-checkin-done")
    }

    /// Tắt hẳn PWA rồi mở lại, ghi cây màn Hôm nay (tìm lối vào Điểm danh).
    func testPwaToday() {
        if pwa.state != .notRunning { pwa.terminate() }
        sleep(2); openPwa(); sleep(5); dump(pwa, "pwa-today")
        XCTAssertTrue(pwa.staticTexts.matching(NSPredicate(format: "label CONTAINS 'Điểm danh'")).firstMatch.waitForExistence(timeout: 5), "Hôm nay không có lối Điểm danh")
    }

    /// Trong PWA: mở thẻ buổi học đang diễn ra ở Hôm nay, ghi cây + ảnh để tìm lối Điểm danh.
    func testPwaSessionCard() {
        if pwa.state != .notRunning { pwa.terminate() }
        sleep(2); openPwa(); sleep(4)
        let esc = pwa.buttons["Đóng"]; if esc.exists { esc.tap() }
        let card = pwa.links.matching(NSPredicate(format: "label CONTAINS 'Buổi 2'")).firstMatch
        XCTAssertTrue(card.waitForExistence(timeout: 10), "không thấy thẻ Buổi 2")
        card.tap(); sleep(4); dump(pwa, "pwa-session")
    }

    // ---- qc3: lô A (chụp ảnh + đo khung phần tử) ----
    func evidDir() -> String { ProcessInfo.processInfo.environment["EVID"] ?? "/tmp" }
    func savePng(_ name: String) {
        let data = XCUIScreen.main.screenshot().pngRepresentation
        try? data.write(to: URL(fileURLWithPath: evidDir() + "/" + name + ".png"))
    }
    func shotName() -> String { ProcessInfo.processInfo.environment["SHOT"] ?? "shot" }
    /// Chụp Safari hiện tại (dùng sau khi runner gọi `simctl openurl`).
    func testShotSafari() { safari.activate(); sleep(4); savePng(shotName()); dump(safari, shotName()) }
    /// Chụp PWA đang chạy (dùng sau khi PWA đã mở bằng testPwaLogin/testPwaToday).
    func testShotPwa() { pwa.activate(); sleep(3); savePng(shotName()); dump(pwa, shotName()) }
    /// Đo khung: thanh trên, thanh dưới/nav, ô nhập, nút chính — đầu ra JSON trong shots/<SHOT>-frames.txt
    func testMeasureCheckin() {
        safari.activate(); sleep(3)
        let web = safari.webViews.firstMatch
        _ = web.waitForExistence(timeout: 20)
        let win = safari.windows.firstMatch.frame
        var lines: [String] = ["window=\(win)"]
        let labels = ["Mã điểm danh", "Điểm danh", "Hôm nay", "Chat riêng", "Luyện đề", "Thêm", "Thoát"]
        for l in labels {
            let pred = NSPredicate(format: "label == %@", l)
            for (i, q) in [web.buttons, web.links, web.staticTexts, web.textFields, web.otherElements].enumerated() {
                let e = q.matching(pred).firstMatch
                if e.exists { lines.append("\(l) [\(i)] frame=\(e.frame)"); break }
            }
        }
        for (i, f) in web.textFields.allElementsBoundByIndex.enumerated() where i < 3 { lines.append("field\(i) frame=\(f.frame)") }
        let txt = lines.joined(separator: "\n")
        try? txt.write(toFile: evidDir() + "/" + shotName() + "-frames.txt", atomically: true, encoding: .utf8)
        savePng(shotName())
        print(txt)
    }
    /// Danh sách đăng ký service worker (PWA-01) — chỉ tin được trong trang của app.
    func testSwRegistrations() {
        safari.activate(); sleep(2)
        let web = safari.webViews.firstMatch
        _ = web.waitForExistence(timeout: 20)
        dump(safari, "sw-check")
    }

    /// Màn hình chính (SpringBoard) — chụp biểu tượng PWA (PWA-03).
    func testShotHome() { safarileave(); savePng(shotName()); dump(springboard, shotName()) }
    func safarileave() {
        XCUIDevice.shared.press(.home); sleep(3)
        let dd = springboard.coordinate(withNormalizedOffset: CGVector(dx: 0.9, dy: 0.5))
        dd.press(forDuration: 0.1, thenDragTo: springboard.coordinate(withNormalizedOffset: CGVector(dx: 0.1, dy: 0.5))); sleep(2)
    }
    /// Mở PWA bằng biểu tượng ở trang 2 của Màn hình chính rồi chụp.
    func testOpenPwaShot() {
        if pwa.state != .runningForeground { openPwa() }
        sleep(5); savePng(shotName()); dump(pwa, shotName())
    }

    /// PWA: vào màn Điểm danh qua ô «Tìm nhanh» rồi chụp (SAF-06, THM-04 phần PWA).
    func testPwaGotoCheckin() {
        if pwa.state != .runningForeground { openPwa() }
        sleep(3)
        let web = pwa.webViews.firstMatch
        _ = web.waitForExistence(timeout: 15)
        if !web.textFields.firstMatch.exists {
            if tapAny(pwa, ["Tìm nhanh"], timeout: 8) {
                sleep(2); pwa.typeText("Điểm danh"); sleep(2)
                _ = tapAny(pwa, ["Điểm danh"], timeout: 8); sleep(3)
            } else if tapAny(pwa, ["Thêm"], timeout: 6) {
                sleep(2); _ = tapAny(pwa, ["Điểm danh"], timeout: 8); sleep(3)
            }
        }
        dump(pwa, shotName()); savePng(shotName())
        print("fields=", web.textFields.count)
    }

    /// Tắt hẳn Safari (xoá sessionStorage của tab) — dùng để kiểm lại từ trạng thái sạch.
    func testKillSafari() { safari.terminate(); sleep(3); safari.launch(); sleep(4); savePng(shotName()) }

    /// Lô B: ô mã (ERR-11/12/13) — nút khoá khi rỗng, câu lỗi chung, maxlength 12.
    func testCodeEntry() {
        safari.activate(); sleep(3)
        let web = safari.webViews.firstMatch
        _ = web.waitForExistence(timeout: 20)
        var log: [String] = []
        let field = web.textFields.firstMatch
        let btnName = "Điểm danh"
        let btn = web.buttons.matching(NSPredicate(format: "label == %@", btnName)).firstMatch
        log.append("field_exists=\(field.waitForExistence(timeout: 10)) btn_exists=\(btn.exists) btn_enabled=\(btn.isEnabled)")
        let long = String(repeating: "A", count: 40)
        field.tap(); sleep(2); field.typeText(long); sleep(1)
        log.append("value_len=\(((field.value as? String) ?? "").count)")
        let bad = "ZZZ999"
        field.tap(); sleep(1)
        let cur = (field.value as? String) ?? ""
        if !cur.isEmpty { field.typeText(String(repeating: XCUIKeyboardKey.delete.rawValue, count: cur.count)) }
        field.typeText(bad); sleep(1)
        if btn.isEnabled { btn.tap() } else { safari.typeText("\n") }
        sleep(4)
        let body = safari.webViews.firstMatch.staticTexts.allElementsBoundByIndex.map(\.label).joined(separator: " | ")
        log.append("body=\(body.prefix(400))")
        try? log.joined(separator: "\n").write(toFile: evidDir() + "/" + shotName() + "-log.txt", atomically: true, encoding: .utf8)
        savePng(shotName())
        print(log.joined(separator: "\n"))
    }

    /// ERR-11: gõ mã sai rồi Enter → câu lỗi chung dưới ô, aria-invalid.
    func testWrongCode() {
        safari.activate(); sleep(3)
        let web = safari.webViews.firstMatch; _ = web.waitForExistence(timeout: 20)
        let field = web.textFields.firstMatch; _ = field.waitForExistence(timeout: 10)
        field.tap(); sleep(2)
        let cur = (field.value as? String) ?? ""
        if !cur.isEmpty { field.typeText(String(repeating: XCUIKeyboardKey.delete.rawValue, count: cur.count)) }
        field.typeText("ZZZ999"); sleep(1); safari.typeText("\n"); sleep(5)
        var log = ["web_true"]
        log.append("texts=\(web.staticTexts.allElementsBoundByIndex.map(\.label).joined(separator: " | "))")
        try? log.joined(separator: "\n").write(toFile: evidDir() + "/" + shotName() + "-log.txt", atomically: true, encoding: .utf8)
        savePng(shotName()); print(log.joined(separator: "\n").prefix(600))
    }

    /// Đổi tài khoản trong app: Thêm → Đăng xuất → đăng nhập sv.gioi (dùng cho DEF-MOB-05 vì sv.qc3 đã điểm danh các cửa sổ đang mở).
    func testSwitchToGioi() {
        safari.activate(); sleep(3)
        let web = safari.webViews.firstMatch; _ = web.waitForExistence(timeout: 20)
        let more = web.buttons.matching(NSPredicate(format: "label == %@", "Thêm")).firstMatch
        if more.exists { more.tap() } else { web.staticTexts.matching(NSPredicate(format: "label == %@", "Thêm")).firstMatch.tap() }
        sleep(3)
        let out = web.buttons.matching(NSPredicate(format: "label CONTAINS[c] %@", "Đăng xuất")).firstMatch
        if out.exists { out.tap() } else { web.staticTexts.matching(NSPredicate(format: "label CONTAINS[c] %@", "Đăng xuất")).firstMatch.tap() }
        sleep(4)
        let confirm = safari.alerts.buttons.matching(NSPredicate(format: "label CONTAINS[c] %@", "Đăng xuất")).firstMatch
        if confirm.exists { confirm.tap(); sleep(4) }
        let email = web.textFields.firstMatch
        _ = email.waitForExistence(timeout: 15)
        email.tap(); sleep(1); email.typeText("sv.gioi@edupilot.local")
        let pw = web.secureTextFields.firstMatch
        pw.tap(); sleep(1); pw.typeText("Edupilot#Seed-2026")
        let login = web.buttons.matching(NSPredicate(format: "label == %@", "Đăng nhập")).firstMatch
        if login.exists { login.tap() } else { safari.typeText("\n") }
        sleep(9)
        savePng(shotName())
        print("after login: \((web.staticTexts.allElementsBoundByIndex.map(\.label).prefix(12)).joined(separator: " | "))")
    }

    /// FLW-04: ô mã + nút Điểm danh gõ mã đúng → «Đã điểm danh»; không có nút quét (iOS).
    func testCodeCheckin() {
        safari.activate(); sleep(2)
        let web = safari.webViews.firstMatch; _ = web.waitForExistence(timeout: 20)
        let scan = web.buttons.matching(NSPredicate(format: "label CONTAINS[c] %@", "Quét")).count
        let field = web.textFields.firstMatch; _ = field.waitForExistence(timeout: 12)
        let code = "NKB 6Q1"
        field.tap(); sleep(2); field.typeText(code); sleep(1)
        safari.typeText("\n"); sleep(7)
        let texts = web.staticTexts.allElementsBoundByIndex.map(\.label).joined(separator: " | ")
        try? ("scan_buttons=\(scan) code=\(code) texts=\(texts)").write(toFile: evidDir() + "/" + shotName() + "-log.txt", atomically: true, encoding: .utf8)
        savePng(shotName()); print("scan=\(scan) texts=\(texts.prefix(300))")
    }

    /// Đăng xuất: bấm avatar góc phải trên → «Đăng xuất» → xác nhận.
    func testLogout() {
        safari.activate(); sleep(2)
        let web = safari.webViews.firstMatch; _ = web.waitForExistence(timeout: 20)
        let f = safari.frame
        safari.coordinate(withNormalizedOffset: CGVector(dx: 0.88, dy: 0.115)).tap()
        sleep(3)
        var labels = web.staticTexts.allElementsBoundByIndex.map(\.label)
        var target = web.buttons.matching(NSPredicate(format: "label CONTAINS[c] %@", "Đăng xuất")).firstMatch
        if !target.exists { target = web.staticTexts.matching(NSPredicate(format: "label CONTAINS[c] %@", "Đăng xuất")).firstMatch }
        if target.exists { target.tap(); sleep(2) }
        let confirm = safari.alerts.buttons.matching(NSPredicate(format: "label CONTAINS[c] %@", "Đăng xuất")).firstMatch
        if confirm.exists { confirm.tap(); sleep(5) }
        labels = web.staticTexts.allElementsBoundByIndex.map(\.label)
        try? ("frame=\(f) labels=\(labels.joined(separator: " | "))").write(toFile: evidDir() + "/" + shotName() + "-log.txt", atomically: true, encoding: .utf8)
        savePng(shotName())
        print("labels=\(labels.prefix(20))")
    }

    /// Xoay ngang, chụp, trả lại dọc.
    func testLandscape() {
        XCUIDevice.shared.orientation = .landscapeLeft; sleep(4)
        safari.activate(); sleep(2)
        savePng(shotName())
        XCUIDevice.shared.orientation = .portrait; sleep(2)
    }
    /// Bàn phím mở: chụp ô nhập; gõ mã sai + phím Đi; chụp câu lỗi so với bàn phím.
    func testKeyFlow() {
        safari.activate(); sleep(2)
        let web = safari.webViews.firstMatch; _ = web.waitForExistence(timeout: 20)
        let field = web.textFields.firstMatch; _ = field.waitForExistence(timeout: 12)
        field.tap(); sleep(3)
        savePng(shotName() + "-kbd")
        field.typeText("ZZZ999"); sleep(1)
        safari.typeText("\n"); sleep(6)
        savePng(shotName() + "-err")
        print(web.staticTexts.allElementsBoundByIndex.map(\.label).joined(separator: " | ").prefix(300))
    }

    /// Chọn lớp QC3M1 trong ô chọn lớp ở thanh trên.
    func testSelectClass() {
        safari.activate(); sleep(2)
        let web = safari.webViews.firstMatch; _ = web.waitForExistence(timeout: 20)
        let sel = web.buttons.matching(NSPredicate(format: "label CONTAINS[c] %@", "QC3")).firstMatch
        if sel.exists { sel.tap() } else { web.staticTexts.matching(NSPredicate(format: "label CONTAINS[c] %@", "QC3")).firstMatch.tap() }
        sleep(3)
        let opt = web.staticTexts.matching(NSPredicate(format: "label CONTAINS[c] %@", "QC3M1")).firstMatch
        if opt.exists { opt.tap() } else { web.buttons.matching(NSPredicate(format: "label CONTAINS[c] %@", "QC3M1")).firstMatch.tap() }
        sleep(5)
        savePng(shotName())
        print(web.staticTexts.allElementsBoundByIndex.map(\.label).prefix(12).joined(separator: " | "))
    }

    /// KBD-04: gõ MỘT dạng (env KBD_FORM) vào ô trống, đọc giá trị.
    func testKbd04() {
        safari.activate(); sleep(2)
        let web = safari.webViews.firstMatch; _ = web.waitForExistence(timeout: 20)
        let f = web.textFields.firstMatch; _ = f.waitForExistence(timeout: 12)
        let form = ProcessInfo.processInfo.environment["KBD_FORM"] ?? "abc 123"
        f.tap(); sleep(2)
        f.typeText(form); sleep(2)
        let v = (f.value as? String) ?? ""
        try? ("\(form) -> \(v)").write(toFile: evidDir() + "/" + shotName() + "-log.txt", atomically: true, encoding: .utf8)
        savePng(shotName()); print("\(form) -> \(v)")
    }

    /// KBD-02: ngang + bàn phím mở — isHittable của nút «Điểm danh», tối đa 3 lần vuốt.
    func testKbd02Strict() {
        XCUIDevice.shared.orientation = .landscapeLeft; sleep(4)
        safari.activate(); sleep(3)
        let web = safari.webViews.firstMatch; _ = web.waitForExistence(timeout: 20)
        let f = web.textFields.firstMatch; _ = f.waitForExistence(timeout: 12)
        f.tap(); sleep(3)
        let btn = web.buttons.matching(NSPredicate(format: "label == %@", "Điểm danh")).firstMatch
        var log: [String] = ["btn_exists=\(btn.exists) hit0=\(btn.exists ? String(btn.isHittable) : "n/a")"]
        savePng(shotName() + "-0")
        var tries = 0
        if btn.exists {
            while !btn.isHittable && tries < 3 {
                web.swipeUp(); sleep(2); tries += 1
                log.append("after_swipe\(tries) hit=\(btn.isHittable)")
                savePng(shotName() + "-\(tries)")
            }
        }
        log.append("final_hit=\(btn.exists ? String(btn.isHittable) : "notfound")")
        try? log.joined(separator: "\n").write(toFile: evidDir() + "/" + shotName() + "-log.txt", atomically: true, encoding: .utf8)
        XCUIDevice.shared.orientation = .portrait; sleep(2)
        print(log.joined(separator: " | "))
    }

    /// KBD-09 (strict): chạm giữ 1.0 s ⇒ menu Paste/«Dán» ⇒ xử lý «Allow Paste» của SpringBoard.
    func testKbd09Strict() {
        safari.activate(); sleep(2)
        let web = safari.webViews.firstMatch; _ = web.waitForExistence(timeout: 20)
        let f = web.textFields.firstMatch; _ = f.waitForExistence(timeout: 12)
        f.tap(); sleep(2)
        f.press(forDuration: 1.0); sleep(2)
        savePng(shotName() + "-menu")
        var log: [String] = ["menus=\(safari.menuItems.allElementsBoundByIndex.map(\.label).prefix(7))"]
        let paste = safari.menuItems.matching(NSPredicate(format: "label IN %@", ["Paste", "Dán"])).firstMatch
        if paste.exists { paste.tap() }
        sleep(2)
        let spring = XCUIApplication(bundleIdentifier: "com.apple.springboard")
        let allow = spring.buttons.matching(NSPredicate(format: "label IN %@", ["Allow Paste", "Cho phép dán"])).firstMatch
        if allow.waitForExistence(timeout: 4) { allow.tap(); sleep(2); log.append("allow_tapped") }
        sleep(2)
        log.append("value=\((f.value as? String) ?? "")")
        savePng(shotName())
        try? log.joined(separator: "\n").write(toFile: evidDir() + "/" + shotName() + "-log.txt", atomically: true, encoding: .utf8)
        print(log.joined(separator: " | "))
    }

    /// TXT-04b: aA → Page Zoom → 200 %.
    func testTxt04b() {
        safari.activate(); sleep(2)
        _ = safari.webViews.firstMatch.waitForExistence(timeout: 20)
        let aa = safari.buttons.matching(NSPredicate(format: "label IN %@", ["aA", "AA", "Page Menu"])).firstMatch
        if aa.exists { aa.tap() } else { safari.buttons["PageMenuButton"].tap() }
        sleep(3)
        let zoom = safari.buttons.matching(NSPredicate(format: "label CONTAINS[c] %@", "Page Zoom")).firstMatch
        if zoom.exists { zoom.tap(); sleep(2) }
        let two = safari.buttons.matching(NSPredicate(format: "label IN %@", ["200%", "200 %"])).firstMatch
        if two.exists { two.tap(); sleep(4) }
        savePng(shotName())
        print("zoom_tapped=\(two.exists)")
    }

    /// PRM-07/08 + GEO: xử lý hộp xin quyền vị trí (allow/deny) rồi chụp kết quả.
    func testGeoPrompt() {
        safari.activate(); sleep(2)
        let web = safari.webViews.firstMatch; _ = web.waitForExistence(timeout: 20)
        let allow = ProcessInfo.processInfo.environment["GEO_ALLOW"] ?? "1"
        var log: [String] = []
        let spring = XCUIApplication(bundleIdentifier: "com.apple.springboard")
        let allowOnce = spring.buttons.matching(NSPredicate(format: "label IN %@", ["Allow Once", "Cho phép một lần"])).firstMatch
        let allowWhile = spring.buttons.matching(NSPredicate(format: "label IN %@", ["Allow While Using Websites", "Cho phép khi dùng trang web"])).firstMatch
        let deny = spring.buttons.matching(NSPredicate(format: "label CONTAINS[c] %@", "phép")).firstMatch
        if allowOnce.waitForExistence(timeout: 8) || allowWhile.waitForExistence(timeout: 2) || deny.exists {
            log.append("prompt shown")
            if allow == "1" {
                if allowWhile.exists { allowWhile.tap() } else if allowOnce.exists { allowOnce.tap() }
            } else if deny.exists { deny.tap() }
            sleep(2)
        } else { log.append("no prompt") }
        sleep(8)
        log.append("texts=\((web.staticTexts.allElementsBoundByIndex.map(\.label).prefix(10)).joined(separator: " | "))")
        savePng(shotName())
        try? log.joined(separator: "\n").write(toFile: evidDir() + "/" + shotName() + "-log.txt", atomically: true, encoding: .utf8)
        print(log.joined(separator: " | "))
    }

    /// PWA: bấm aA ở thanh dưới (toạ độ) → Share → Add to Home Screen → Add.
    func testPwaInstall() {
        safari.activate(); sleep(2)
        _ = safari.webViews.firstMatch.waitForExistence(timeout: 20)
        let aa = safari.buttons.matching(NSPredicate(format: "label IN %@", ["aA", "AA", "Page Menu"])).firstMatch
        if aa.exists { aa.tap() } else { safari.buttons["PageMenuButton"].tap() }
        sleep(3); savePng(shotName() + "-menu")
        let share = safari.staticTexts.matching(NSPredicate(format: "label CONTAINS[c] %@", "Share")).firstMatch
        if share.exists { share.tap() } else { safari.buttons.matching(NSPredicate(format: "label CONTAINS[c] %@", "Share")).firstMatch.tap() }
        sleep(6); savePng(shotName() + "-share")
        var add = safari.staticTexts.matching(NSPredicate(format: "label CONTAINS[c] %@", "Add to Home Screen")).firstMatch
        if !add.exists { safari.swipeUp(); sleep(2); add = safari.staticTexts.matching(NSPredicate(format: "label CONTAINS[c] %@", "Add to Home Screen")).firstMatch }
        if add.exists { add.tap(); sleep(4); savePng(shotName() + "-sheet") }
        let addBtn = safari.buttons["Add"]
        if addBtn.waitForExistence(timeout: 5) { addBtn.tap(); sleep(4) } else {
            safari.coordinate(withNormalizedOffset: CGVector(dx: 0.91, dy: 0.085)).tap(); sleep(4)
        }
        savePng(shotName())
    }

    /// Mở PWA từ Màn hình chính (web clip), chụp trạng thái standalone.
    func testPwaOpen() {
        let spring = XCUIApplication(bundleIdentifier: "com.apple.springboard")
        spring.activate(); sleep(2)
        let icon = spring.icons.matching(NSPredicate(format: "label == %@", "EduPilot")).firstMatch
        if icon.exists { icon.tap() } else { spring.icons.matching(NSPredicate(format: "label CONTAINS[c] %@", "EduPilot")).firstMatch.tap() }
        sleep(12)
        savePng(shotName())
        print("safari_chrome=\(safari.buttons["PageMenuButton"].exists)")
    }

    /// NAV: vuốt cạnh trái để quay lại trong PWA, chụp trước/sau.
    func testNavBack() {
        let spring = XCUIApplication(bundleIdentifier: "com.apple.springboard")
        spring.activate(); sleep(2)
        let icon = spring.icons.matching(NSPredicate(format: "label == %@", "EduPilot")).firstMatch
        if icon.exists { icon.tap() } else { spring.icons.matching(NSPredicate(format: "label CONTAINS[c] %@", "EduPilot")).firstMatch.tap() }
        sleep(8)
        savePng(shotName() + "-home")
        let app = XCUIApplication(bundleIdentifier: "com.apple.webapp")
        let web = app.webViews.firstMatch
        if web.waitForExistence(timeout: 10) {
            web.swipeRight(); sleep(3); savePng(shotName() + "-back")
            web.buttons.matching(NSPredicate(format: "label CONTAINS[c] %@", "Điểm danh")).firstMatch.tap()
            sleep(4); savePng(shotName() + "-checkin")
            web.swipeRight(); sleep(3); savePng(shotName() + "-back2")
        }
    }

    /// PWA: về «Hôm nay» (toạ độ) và chụp — kiểm phiên còn sống sau khi Safari đăng xuất (P-11-148).
    func testPwaHomeState() {
        let spring = XCUIApplication(bundleIdentifier: "com.apple.springboard")
        spring.activate(); sleep(2)
        spring.icons.matching(NSPredicate(format: "label == %@", "EduPilot")).firstMatch.tap()
        sleep(9)
        let app = XCUIApplication(bundleIdentifier: "com.apple.webapp")
        app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: 0.08)).tap(); sleep(2)
        app.coordinate(withNormalizedOffset: CGVector(dx: 0.10, dy: 0.915)).tap(); sleep(4)
        savePng(shotName())
    }

    /// Cuộn trang Safari 2 lần rồi chụp (xem khối dưới màn).
    func testScrollDown() {
        safari.activate(); sleep(3)
        let web = safari.webViews.firstMatch; _ = web.waitForExistence(timeout: 15)
        web.swipeUp(); sleep(2); savePng(shotName() + "-1")
        web.swipeUp(); sleep(2); savePng(shotName() + "-2")
        web.swipeUp(); sleep(3); savePng(shotName() + "-3")
    }

    /// Mở danh sách chọn buổi trên /attendance (giảng viên) và chụp.
    func testSessionPicker() {
        safari.activate(); sleep(3)
        let web = safari.webViews.firstMatch; _ = web.waitForExistence(timeout: 15)
        web.coordinate(withNormalizedOffset: CGVector(dx: 0.25, dy: 0.418)).tap(); sleep(3)
        savePng(shotName() + "-list")
        let si = web.staticTexts.allElementsBoundByIndex.map(\.label)
        try? si.joined(separator: " | ").write(toFile: evidDir() + "/" + shotName() + "-list.txt", atomically: true, encoding: .utf8)
        print("opts=\(si.prefix(30))")
    }

    /// /attendance: mở danh sách buổi bằng nhãn (không theo toạ độ) và chụp.
    func testPickSession() {
        safari.activate(); sleep(3)
        let web = safari.webViews.firstMatch; _ = web.waitForExistence(timeout: 15)
        let trig = web.staticTexts.matching(NSPredicate(format: "label BEGINSWITH %@", "Buổi ")).firstMatch
        XCTAssertTrue(trig.waitForExistence(timeout: 10), "không thấy nút chọn buổi")
        trig.tap(); sleep(3)
        savePng(shotName() + "-list")
        let opts = web.staticTexts.allElementsBoundByIndex.map(\.label).filter { $0.hasPrefix("Buổi ") }
        try? opts.joined(separator: " | ").write(toFile: evidDir() + "/" + shotName() + "-opts.txt", atomically: true, encoding: .utf8)
        print("opts=\(opts)")
    }

    /// /attendance (GV): bấm «Mở màn chiếu» rồi chụp màn chiếu.
    func testOpenDisplay() {
        safari.activate(); sleep(3)
        let web = safari.webViews.firstMatch; _ = web.waitForExistence(timeout: 15)
        let b = web.buttons.matching(NSPredicate(format: "label CONTAINS %@", "Mở màn chiếu")).firstMatch
        if !b.exists {
            let t = web.staticTexts.matching(NSPredicate(format: "label CONTAINS %@", "Mở màn chiếu")).firstMatch
            XCTAssertTrue(t.waitForExistence(timeout: 10), "không thấy nút Mở màn chiếu"); t.tap()
        } else { b.tap() }
        sleep(10); savePng(shotName())
        let labels = web.staticTexts.allElementsBoundByIndex.map(\.label)
        try? labels.joined(separator: " | ").write(toFile: evidDir() + "/" + shotName() + "-labels.txt", atomically: true, encoding: .utf8)
    }
}
