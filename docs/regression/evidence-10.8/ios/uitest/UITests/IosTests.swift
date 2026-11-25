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
        let sheet = safari.otherElements["ActivityListView"]
        if sheet.exists { sheet.swipeUp() }
        sleep(1); dump(safari, "share2")
        XCTAssertTrue(tapAny(safari, ["Add to Home Screen"], timeout: 8), "không thấy Add to Home Screen"); sleep(3); dump(safari, "addhome")
        XCTAssertTrue(tapAny(safari, ["Add"], timeout: 8), "không thấy nút Add"); sleep(3); dump(safari, "added")
    }
}
