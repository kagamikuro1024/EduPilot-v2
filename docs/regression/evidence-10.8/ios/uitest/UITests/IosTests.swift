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
        let raw = (try? String(contentsOfFile: "/Users/kuro/.config/edupilot-team/qc2-work/code.txt", encoding: .utf8)) ?? ""
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
}
