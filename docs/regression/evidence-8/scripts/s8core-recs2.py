from s8core_lib import *
rec("TC-P6-02-17", "P", "go vet ./internal/grade/... ./internal/grading/... rc=0; golangci-lint run ./internal/grade/...: 0 issues; TestNoFloatInGrade PASS (logs/s8core-go2.log)")
rec("TC-P6-02-36", "P", "TestWhatIfNoWrite/MinFinalFor/PreviewUsesCompute PASS; BenchmarkCompute1000 = 27,9 ms/op (≤200 ms) trên máy QC (tải cao)")
rec("TC-P6-01-20", "F", "weekly A 10, B 8, C 0 và 24 dòng MCQ khớp CSV (đọc qua explain), nhưng giải trình C không có dòng 'không dự → 0 theo quy chế' (DEF-R8-s8core-01)")
rec("TC-P6-02-25", "F", "C: KT=0, status COMPLETE, nhưng dòng KT chỉ expr '0' (filled_by_user, không quote điều 1.4), không có chữ 'không dự → 0 theo quy chế' (DEF-R8-s8core-01)")
