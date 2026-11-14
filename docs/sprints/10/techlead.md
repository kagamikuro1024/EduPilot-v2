# Sprint 10: kênh Tech Lead

## TL-1 (US-P9-02, L1 Cao): trích đề trên docling thật ra 0/24 câu

PM hỏi thay dev (hạn 14:00): chẩn đoán vì sao trích ra 0 câu, đề xuất sửa (tệp, hàm). Không sửa mã.

**TL trả lời:**

### Kết luận
Đầu ra của docling đúng: Markdown có đủ tiêu đề đề thi và cả 24 câu. Lỗi nằm ở bộ chia đoạn của `internal/ingest`, không nằm ở prompt. Chỉ 3/24 câu mất vì `windows()` như QC ghi. 13/24 câu và tên đề bị `mergeShort` xoá hẳn trước khi lưu vào `content_chunks`. Vì vậy chỉ sửa `practice/extract.go` thì vẫn thiếu đề bài của 13 câu, kể cả khi chạy LLM thật. Phải sửa 2 tệp, cách sửa ở mục "Đề xuất sửa".

### Bằng chứng (lần chạy 2026-11-14 trên stack `ep-qc`, chỉ đọc)
1. **Đầu ra docling.** Gọi `POST /v1/convert/file` trong container `ep-qc-docling-1` với `seed/practice/de-mau-trac-nghiem.pdf`, cùng tham số `ingest/docling.go` gửi (`md`, `pypdfium2`, `do_ocr=false`, mốc trang `<!-- page -->`). Kết quả có 159 dòng:
   - `## ĐỀ MẪU LUYỆN TẬP — MẠNG MÁY TÍNH` ở dòng 1, ngay sau đó là `## Câu 1. …`, không có thân;
   - 16 câu có dạng tiêu đề `## Câu N. …` (1, 2, 7, 8, 13–24);
   - 8 câu có dạng mục danh sách `5. Câu 3. …`, `10. Câu 4. …` (3–6, 9–12);
   - có 3 mốc trang, tức 4 trang.
2. **DB (`content_chunks` của "QC Đề mẫu").** Có 3 đoạn, mỗi đoạn 600/760/680 ký tự, ở trang 1/2/3. Cột `heading` lần lượt là `Câu 1.`, `Câu 8.`, `Câu 17.`. Cột `text` chỉ chứa 8 câu dạng danh sách cùng phương án của mọi câu. Mất đề bài các câu 2, 7, 13–16, 18–24 và mất tên đề. Không đoạn nào ghi trang 4.
3. **Tái hiện ngoài stack.** Chạy `ingest.ChunkMarkdown(Clean(md), ChunkChars=800, MinChunkChars=200)` trên Markdown ở mục 1 bằng `go test -overlay`, không đụng cây mã. Kết quả ra đúng 3 đoạn 600/760/680, trùng từng ký tự với DB.

### Cơ chế mất chữ (`backend-go/internal/ingest/chunk.go`)
- `ChunkMarkdown`, dòng 80–84: dòng `## …` không vào thân. Nó chỉ thành `heading` của các khối phía sau.
- Mỗi `## Câu N.` mở một khối khoảng 60–100 ký tự. Khối nhỏ hơn `MinChunkChars=200` nên `mergeShort` gộp nó vào đoạn trước, dòng 142–146: chỉ nối `Text`, còn `Heading` của khối bị gộp bị **bỏ đi**. Đề bài của câu đó vì thế không còn trong `text`, `heading` hay embedding. Lỗi này áp cho mọi tài liệu nhiều tiêu đề ngắn: quy chế, slide. Nghĩa là RAG cũng bị.
- Tiêu đề không có thân, như tên đề nằm ngay trước `## Câu 1`, bị ghi đè ở dòng 82 mà chưa thành khối nào. Provider giả (`seed/demo/fake_llm.json`, bản ghi `practice-extract-de-mau`) khớp theo chuỗi tên đề này. Không khớp thì nó sinh JSON ngẫu nhiên theo schema, ra 1 câu `INVALID`. Đó chính là `skipped:[{page:1,number:1,reason:"INVALID"}]` QC thấy.
- `mergeShort` gộp cả qua ranh giới trang: các câu 19–24 ở trang 4 bị nhập vào đoạn trang 3. Khi sửa xong mà vẫn để vậy, trích dẫn của 6 câu này sẽ ghi sai trang.
- `practice/extract.go` `windows()`, dòng 185, chỉ đưa `c.Text` vào lời nhắc. Truy vấn `PracticeDocChunks` (`store/queries/practice.sql:186`) cũng không lấy cột `heading`. Do đó câu nằm ở `heading` của mỗi đoạn vẫn mất.

### Đề xuất sửa
1. **`internal/ingest/chunk.go`, `mergeShort`**:
   - Khi gộp hai đoạn có tiêu đề khác nhau, chèn tiêu đề của đoạn bị gộp vào đầu phần văn bản nối thêm, qua helper kiểu `headed(c, into)`. Áp cho cả gộp lùi lẫn gộp tới. Khi so với `maxRunes`, tính cả độ dài tiêu đề.
   - Chỉ gộp khi hai đoạn **cùng `PageNo`**.
2. **`internal/ingest/chunk.go`, `ChunkMarkdown`**: dùng cờ `bare`. Nếu tiêu đề cũ chưa có thân mà gặp tiêu đề mới, đẩy một khối có `text = tiêu đề cũ`.
3. **`internal/store/queries/practice.sql`, `PracticeDocChunks`**: thêm `heading` vào select rồi chạy `sqlc generate`. Trong `practice/extract.go`, `windows()` đưa `*c.Heading` lên trước `c.Text`. Bỏ qua bước này khi `heading` rỗng hoặc `text` đã bắt đầu bằng chính tiêu đề đó.
4. **Test hồi quy (`internal/ingest`)**: lưu Markdown docling thật của đề mẫu vào `testdata`, lấy lại bằng lệnh ở mục Bằng chứng 1. Kiểm rằng sau `ChunkMarkdown`, chuỗi `heading` + `text` chứa đủ `Câu 1.` … `Câu 24.` và tên đề, và đoạn chứa `Câu 19.` có `PageNo` 4. Lỗi lọt được vì `TestExtractQuestions` dựng chunk bằng tay.

Minh hoạ cho bước 1 (không phải bản để chép):
```go
if i > 0 && cs[i-1].PageNo == cs[i].PageNo {
	if t := cs[i-1].Text + "\n" + headed(cs[i], cs[i-1].Heading); utf8.RuneCountInString(t) <= maxRunes {
		cs[i-1].Text = t; cs = append(cs[:i], cs[i+1:]...); i--; continue
	}
}
// gộp tới: tương tự với cs[i+1]; cs[i+1].Heading = cs[i].Heading
func headed(c Chunk, into string) string {
	if c.Heading == "" || c.Heading == into { return c.Text }
	return c.Heading + "\n" + c.Text
}
```

**PoC đã chạy** (bản sao ở `/tmp`, nạp bằng `go test -overlay`, không commit):
- Với bước 1 và 2, ra 4 đoạn ở trang 1/2/3/4. Đoạn đầu chứa tên đề. Ghép `heading` + `text` thì đếm được **24/24** dòng `Câu N.`.
- `go test ./internal/ingest/` trên chính bản vá vẫn xanh: test cũ giữ nguyên, không phải sửa.

### Rủi ro
- **Tài liệu đã nạp trước bản vá vẫn giữ đoạn cũ, thiếu chữ.** Dev ghi vào handoff: QC phải tải lại đề mẫu, hoặc dựng stack sạch rồi seed. Cùng bản vá, RAG cũng lấy lại được các tiêu đề trước đây bị rơi. Câu trả lời chat có thể đổi nhẹ.
- **Không gộp qua trang làm tăng số đoạn ngắn hơn 200 ký tự**, nhưng chỉ ở trang có rất ít chữ. Mức này chấp nhận được, đổi lại trích dẫn trang đúng.
- **Docling không ổn định giữa `## Câu N.` và `5. Câu N.`.** Prompt hiện tại đọc được cả hai dạng (`"Câu 7" → 7`), không cần sửa `prompts/extract.txt`.
