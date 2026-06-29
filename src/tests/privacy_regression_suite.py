import pytest
from src.guardrails import is_pii_detected_flexible

@pytest.mark.parametrize("text, expected", [
    # MSSV chuẩn (8 số)
    ("Sinh viên có MSSV: 20210001", True),
    ("Mã số sinh viên: 12345678", True),
    ("12345678", True),
    
    # SĐT (0 đầu hoặc +84)
    ("Số điện thoại: 0912345678", True),
    ("Liên hệ: +84912345678", True),
    ("SĐT 0.9.1.2.3.4.5.6.7.8", True),
    ("Call 0-9-8-7-6-5-4-3-2-1", True),
    
    # Email sinh viên nhạy cảm
    ("Email: student@hcmut.edu.vn", True),
    ("Gửi tới 20210001@student.university.edu", True),
    ("test.user@gmail.com", True),
    
    # Edge cases: M S S V, MSSV: ..., khoảng trắng dư thừa
    ("M S S V 2 0 2 1 0 0 0 1", True),
    ("MSSV:    20210001   ", True),
    ("My id is 1 2 3 4 5 6 7 8", True),
    
    # Bypass test: arr[ MSSV: 123 ] (Masking test)
    ("arr[ MSSV: 20210001 ]", True),
    ("matrix[ 0912345678 ]", True),
    ("list[20210001]", True),
    ("vector[ +84912345678 ]", True),
    
    # Non-PII cases (Negative tests)
    ("Đây là một câu hỏi học thuật bình thường về OOP.", False),
    ("arr[i] = 10", False),
    ("matrix[row][col] = 5.0", False),
    ("Số lượng sinh viên là 100", False),
    ("Năm 2024 có nhiều biến động", False),
])
def test_pii_robustness(text, expected):
    """
    TIP-005: Privacy Regression Suite
    Ensures the firewall captures various PII formats while avoiding false positives on code.
    """
    result = is_pii_detected_flexible(text)
    assert result == expected, f"Failed for text: {text}. Expected {expected}, got {result}"

if __name__ == "__main__":
    pytest.main([__file__])
