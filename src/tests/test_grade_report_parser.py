from data_pipeline.pipeline.grade_report import parse_grade_report


SAMPLE = """
<!-- PAGE:1 -->
MOCK DATA
BẢNG ĐIỂM MOCK - LAB 2: SEARCH ALGORITHMS
STT Mã SV Họ và tên Nhóm Linear
(2đ)
Binary
(2đ)
BFS/DFS
(3đ)
Quiz
(3đ)
Tổng
(10đ) Ghi chú
14 SV260114 Đặng Quang Minh Nhóm 5 1.6 1.4 2.3 2.0 7.3 Cần trình bày rõ hơn
15 SV260115 Lê Văn Quang Trung Nhóm 5 2.0 1.9 2.8 2.7 9.4 Code sạch, giải thích tốt
"""


def test_parse_grade_report_extracts_structured_private_rows():
    records = parse_grade_report(SAMPLE, "bang_diem_lab2.pdf")

    assert len(records) == 2
    trung = records[1]
    assert trung["student_code"] == "SV260115"
    assert trung["student_name"] == "Lê Văn Quang Trung"
    assert trung["assignment_title"] == "LAB 2: SEARCH ALGORITHMS"
    assert trung["total_score"] == 9.4
    assert trung["max_score"] == 10.0
    assert trung["feedback"] == "Code sạch, giải thích tốt"
    assert trung["component_scores"] == {
        "Linear": 2.0,
        "Binary": 1.9,
        "BFS/DFS": 2.8,
        "Quiz": 2.7,
    }
    assert trung["source_page"] == 1
    assert trung["source_notice"] == "MOCK DATA"


def test_parse_grade_report_rejects_non_grade_text():
    assert parse_grade_report("Lecture notes about graph search", "lecture.pdf") == []
