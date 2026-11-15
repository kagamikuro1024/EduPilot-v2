package contract

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
)

// practiceScenarios: thao tác 5–10, 13, 14 của SRS FEAT-practice 6 (luyện đề, US-P9-05…09) với mọi status đã khai báo.
func (r *runner) practiceScenarios(x examRig) {
	ctx := context.Background()
	db := r.rig.deps.DB
	var gvID uuid.UUID
	if err := db.QueryRow(ctx, `select user_id from enrollments where course_id = $1 and role_in_course = 'TEACHER'`, x.cid).Scan(&gvID); err != nil {
		r.t.Fatal(err)
	}
	mkQ := func(topic string, n int) {
		for i := range n {
			var qid uuid.UUID
			ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
			key, _ := json.Marshal(map[string]any{"option_ids": []string{ids[1].String()}})
			if err := db.QueryRow(ctx, `insert into question_bank (course_id, type, title, topic, difficulty, stem, answer_key, explanation, review_status, created_by, reviewed_by, reviewed_at)
				values ($1, 'MCQ_SINGLE', $2, $3, 'EASY', $4, $5::jsonb, 'Vì B đúng.', 'APPROVED', $6, $6, now()) returning id`, x.cid, topic+" "+uuid.NewString()[:4], topic, topic+" câu "+uuid.NewString()[:6]+"?", string(key), gvID).Scan(&qid); err != nil {
				r.t.Fatal(err)
			}
			for k, id := range ids {
				if _, err := db.Exec(ctx, `insert into question_options (id, course_id, question_id, position, body) values ($1, $2, $3, $4, $5)`, id, x.cid, qid, k+1, string(rune('A'+k))); err != nil {
					r.t.Fatal(err)
				}
			}
			_ = i
		}
	}
	const topic, few = "Luyện đề hợp đồng", "Chủ đề ít câu"
	mkQ(topic, 6)
	mkQ(few, 3)
	// sinh viên thứ hai (cùng lớp) để thử 404 "lượt của người khác"
	var sv2ID uuid.UUID
	if err := db.QueryRow(ctx, `insert into users (email, full_name, role, status, password_hash, email_verified_at) values ($1, 'Người Thử Hai', 'STUDENT', 'ACTIVE', 'x', now()) returning id`, "ct-pr-"+uuid.NewString()[:8]+"@example.test").Scan(&sv2ID); err != nil {
		r.t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, x.cid, sv2ID); err != nil {
		r.t.Fatal(err)
	}
	sv2 := r.rig.token(r.t, sv2ID.String(), auth.RoleStudent)
	stranger := r.rig.token(r.t, uuid.NewString(), auth.RoleStudent) // không phải thành viên lớp

	p := "/api/v1/courses/" + x.cid + "/practice"
	bad := "/api/v1/courses/khong-phai-uuid/practice"
	start := `{"mode":"TOPIC","topic":"` + topic + `","count":5}`

	// 5: tổng quan.
	r.must(call{method: "GET", path: p + "/overview", token: x.sv}, 200)
	r.must(call{method: "GET", path: p + "/overview"}, 401)
	r.must(call{method: "GET", path: p + "/overview", token: x.gv}, 403)
	r.must(call{method: "GET", path: p + "/overview", token: x.ta}, 403)
	r.must(call{method: "GET", path: p + "/overview", token: stranger}, 403)
	r.must(call{method: "GET", path: bad + "/overview", token: x.sv}, 404)

	// 6: bắt đầu.
	_, b := r.must(call{method: "POST", path: p + "/start", token: x.sv, headers: x.idem(), body: start}, 201)
	var att struct {
		ID    string `json:"id"`
		Items []struct {
			ID       string `json:"id"`
			Question struct {
				Options []struct {
					ID string `json:"id"`
				} `json:"options"`
			} `json:"question"`
		} `json:"items"`
	}
	if err := json.Unmarshal(b, &att); err != nil || len(att.Items) != 5 {
		r.t.Fatalf("start: %v %s", err, b)
	}
	r.must(call{method: "POST", path: p + "/start", token: x.sv, headers: x.idem(), body: start}, 200) // làm tiếp lượt dở
	r.must(call{method: "POST", path: p + "/start", headers: x.idem(), body: start}, 401)
	r.must(call{method: "POST", path: p + "/start", token: x.gv, headers: x.idem(), body: start}, 403)
	r.must(call{method: "POST", path: bad + "/start", token: x.sv, headers: x.idem(), body: start}, 404)
	r.must(call{method: "POST", path: p + "/start", token: x.sv, body: start}, 422)                                                                   // thiếu Idempotency-Key
	r.must(call{method: "POST", path: p + "/start", token: x.sv, headers: x.idem(), body: `{"mode":"TOPIC","topic":"` + topic + `","count":3}`}, 422) // count ngoài [5, 50]
	r.must(call{method: "POST", path: p + "/start", token: x.sv, headers: x.idem(), body: `{"mode":"TOPIC","topic":"` + few + `","count":5}`}, 409)   // NOT_ENOUGH_QUESTIONS
	r.must(call{method: "POST", path: "/api/v1/courses/" + x.arch + "/practice/start", token: x.sv, headers: x.idem(), body: start}, 409)             // COURSE_ARCHIVED

	// 7: đọc lượt.
	A := p + "/attempts/" + att.ID
	r.must(call{method: "GET", path: A, token: x.sv}, 200)
	r.must(call{method: "GET", path: A}, 401)
	r.must(call{method: "GET", path: A, token: x.gv}, 403)
	r.must(call{method: "GET", path: A, token: sv2}, 404) // lượt của người khác
	r.must(call{method: "GET", path: p + "/attempts/" + uuid.NewString(), token: x.sv}, 404)

	// 8: ghi câu (TOPIC: phản hồi ngay, khoá đáp án).
	it := att.Items[0]
	ans := func(opt string) string { return `{"answer":{"option_ids":["` + opt + `"]}}` }
	PUT := A + "/answers/" + it.ID
	r.must(call{method: "PUT", path: PUT, token: x.sv, body: ans(it.Question.Options[0].ID)}, 200)
	r.must(call{method: "PUT", path: PUT, token: x.sv, body: ans(it.Question.Options[1].ID)}, 409) // ANSWER_LOCKED
	r.must(call{method: "PUT", path: PUT, body: ans(it.Question.Options[0].ID)}, 401)
	r.must(call{method: "PUT", path: PUT, token: x.gv, body: ans(it.Question.Options[0].ID)}, 403)
	r.must(call{method: "PUT", path: PUT, token: sv2, body: ans(it.Question.Options[0].ID)}, 404)
	r.must(call{method: "PUT", path: A + "/answers/" + uuid.NewString(), token: x.sv, body: ans(it.Question.Options[0].ID)}, 404)
	r.must(call{method: "PUT", path: A + "/answers/" + att.Items[1].ID, token: x.sv, body: ans(uuid.NewString())}, 422) // INVALID_OPTION_ID

	// 9: vị trí câu đang làm.
	r.must(call{method: "PATCH", path: A, token: x.sv, body: `{"current_index":2}`}, 200)
	r.must(call{method: "PATCH", path: A, token: x.sv, body: `{"current_index":99}`}, 422)
	r.must(call{method: "PATCH", path: A, body: `{"current_index":1}`}, 401)
	r.must(call{method: "PATCH", path: A, token: x.gv, body: `{"current_index":1}`}, 403)
	r.must(call{method: "PATCH", path: A, token: sv2, body: `{"current_index":1}`}, 404)

	// 10: nộp.
	r.must(call{method: "POST", path: A + "/submit", token: x.sv}, 422) // thiếu Idempotency-Key
	r.must(call{method: "POST", path: A + "/submit", headers: x.idem()}, 401)
	r.must(call{method: "POST", path: A + "/submit", token: x.gv, headers: x.idem()}, 403)
	r.must(call{method: "POST", path: A + "/submit", token: sv2, headers: x.idem()}, 404)
	_, sub := r.must(call{method: "POST", path: A + "/submit", token: x.sv, headers: x.idem()}, 200)
	_, sub2 := r.must(call{method: "POST", path: A + "/submit", token: x.sv, headers: x.idem()}, 200) // gọi lại → cùng kết quả
	var r1, r2 struct {
		Result json.RawMessage `json:"result"`
	}
	_ = json.Unmarshal(sub, &r1)
	_ = json.Unmarshal(sub2, &r2)
	if string(r1.Result) != string(r2.Result) || len(r1.Result) < 5 {
		r.t.Fatalf("submit lần hai khác lần một: %s vs %s", r1.Result, r2.Result)
	}
	r.must(call{method: "PATCH", path: A, token: x.sv, body: `{"current_index":1}`}, 409)                                                  // ATTEMPT_CLOSED
	r.must(call{method: "PUT", path: A + "/answers/" + att.Items[2].ID, token: x.sv, body: ans(att.Items[2].Question.Options[0].ID)}, 409) // ATTEMPT_CLOSED

	// 11: giải thích (SSE). A đã nộp.
	X := A + "/items/" + it.ID + "/explain"
	r.must(call{method: "POST", path: X, token: x.sv}, 200)
	r.must(call{method: "POST", path: X}, 401)
	r.must(call{method: "POST", path: X, token: x.gv}, 403)
	r.must(call{method: "POST", path: X, token: sv2}, 404)
	r.must(call{method: "POST", path: A + "/items/" + uuid.NewString() + "/explain", token: x.sv}, 404)

	// 12: nhận xét tự luận. Lượt riêng có một câu ESSAY.
	const essayTopic = "Tự luận hợp đồng"
	mkQ(essayTopic, 4)
	var eq uuid.UUID
	ekey, _ := json.Marshal(map[string]string{"model_answer": "Nêu A và B.", "rubric": "A 5đ; B 5đ"})
	if err := db.QueryRow(ctx, `insert into question_bank (course_id, type, title, topic, difficulty, stem, answer_key, review_status, created_by, reviewed_by, reviewed_at)
		values ($1, 'ESSAY', 'luận', $2, 'MEDIUM', 'Phân tích A và B.', $3::jsonb, 'APPROVED', $4, $4, now()) returning id`, x.cid, essayTopic, string(ekey), gvID).Scan(&eq); err != nil {
		r.t.Fatal(err)
	}
	_, eb := r.must(call{method: "POST", path: p + "/start", token: x.sv, headers: x.idem(), body: `{"mode":"TOPIC","topic":"` + essayTopic + `","count":5}`}, 201)
	var ea struct {
		ID    string `json:"id"`
		Items []struct {
			ID       string `json:"id"`
			Question struct {
				Type string `json:"type"`
			} `json:"question"`
		} `json:"items"`
	}
	if err := json.Unmarshal(eb, &ea); err != nil {
		r.t.Fatal(err)
	}
	var essayID, mcqID string
	for _, i := range ea.Items {
		if i.Question.Type == "ESSAY" {
			essayID = i.ID
		} else if mcqID == "" {
			mcqID = i.ID
		}
	}
	E := p + "/attempts/" + ea.ID + "/items/"
	r.must(call{method: "POST", path: E + mcqID + "/explain", token: x.sv}, 409)                       // NOT_ANSWERED: lượt còn mở, câu chưa trả lời
	r.must(call{method: "POST", path: E + essayID + "/explain", token: x.sv}, 422)                     // INVALID_QUESTION: tự luận không có đáp án đúng để giải thích
	r.must(call{method: "POST", path: E + essayID + "/feedback", token: x.sv, headers: x.idem()}, 409) // NOT_ANSWERED
	r.must(call{method: "POST", path: E + mcqID + "/feedback", token: x.sv, headers: x.idem()}, 422)   // INVALID_QUESTION: chỉ tự luận
	r.must(call{method: "POST", path: E + essayID + "/feedback", token: x.sv}, 422)                    // thiếu Idempotency-Key
	r.must(call{method: "POST", path: E + essayID + "/feedback", headers: x.idem()}, 401)
	r.must(call{method: "POST", path: E + essayID + "/feedback", token: x.gv, headers: x.idem()}, 403)
	r.must(call{method: "POST", path: E + essayID + "/feedback", token: sv2, headers: x.idem()}, 404)
	r.must(call{method: "PUT", path: p + "/attempts/" + ea.ID + "/answers/" + essayID, token: x.sv, body: `{"answer":{"text":"Ý A là quan trọng."}}`}, 200)
	if st, _, b := r.do(call{method: "POST", path: E + essayID + "/feedback", token: x.sv, headers: x.idem()}); st != 200 && st != 503 {
		r.t.Fatalf("feedback → %d: %s", st, b)
	}

	// 1–4: trích, sinh, duyệt hàng loạt, số liệu lô (giảng viên / TA).
	mkDoc := func(course, typ, status string) string {
		var id string
		if err := db.QueryRow(ctx, `insert into documents (course_id, title, type, status, visible_to_students) values ($1, 'Tài liệu thử', $2::document_type, $3::document_status, $2 <> 'ANSWER_KEY') returning id`, course, typ, status).Scan(&id); err != nil {
			r.t.Fatal(err)
		}
		return id
	}
	Q := "/api/v1/courses/" + x.cid + "/questions"
	paperID, pendingPaper, lecID, keyID := mkDoc(x.cid, "EXAM_PAPER", "READY"), mkDoc(x.cid, "EXAM_PAPER", "PROCESSING"), mkDoc(x.cid, "LECTURE", "READY"), mkDoc(x.cid, "ANSWER_KEY", "READY")
	ext := func(doc string) string { return `{"document_id":"` + doc + `"}` }
	r.must(call{method: "POST", path: Q + "/extract", token: x.gv, headers: x.idem(), body: ext(paperID)}, 202)
	r.must(call{method: "POST", path: Q + "/extract", token: x.ta, headers: x.idem(), body: ext(paperID)}, 202)
	r.must(call{method: "POST", path: Q + "/extract", headers: x.idem(), body: ext(paperID)}, 401)
	r.must(call{method: "POST", path: Q + "/extract", token: x.sv, headers: x.idem(), body: ext(paperID)}, 403)
	r.must(call{method: "POST", path: Q + "/extract", token: x.gv, headers: x.idem(), body: ext(uuid.NewString())}, 404)
	r.must(call{method: "POST", path: Q + "/extract", token: x.gv, headers: x.idem(), body: ext(pendingPaper)}, 409) // DOCUMENT_NOT_READY
	r.must(call{method: "POST", path: Q + "/extract", token: x.gv, headers: x.idem(), body: ext(keyID)}, 422)        // WRONG_DOCUMENT_TYPE
	r.must(call{method: "POST", path: Q + "/extract", token: x.gv, body: ext(paperID)}, 422)                         // thiếu Idempotency-Key

	gen := func(body string) call {
		return call{method: "POST", path: Q + "/generate", token: x.gv, headers: x.idem(), body: body}
	}
	r.must(gen(`{"document_ids":["`+lecID+`"],"topic":"Mạng","difficulty":"EASY","count":3}`), 202)
	r.must(call{method: "POST", path: Q + "/generate", headers: x.idem(), body: `{"document_ids":["` + lecID + `"],"topic":"Mạng","difficulty":"EASY","count":3}`}, 401)
	r.must(call{method: "POST", path: Q + "/generate", token: x.sv, headers: x.idem(), body: `{"document_ids":["` + lecID + `"],"topic":"Mạng","difficulty":"EASY","count":3}`}, 403)
	r.must(gen(`{"document_ids":["`+uuid.NewString()+`"],"topic":"Mạng","difficulty":"EASY","count":3}`), 404)
	r.must(gen(`{"document_ids":["`+mkDoc(x.cid, "LECTURE", "PROCESSING")+`"],"topic":"Mạng","difficulty":"EASY","count":3}`), 409)
	r.must(gen(`{"document_ids":["`+keyID+`"],"topic":"Mạng","difficulty":"EASY","count":3}`), 422) // ANSWER_KEY không làm nguồn
	r.must(gen(`{"document_ids":["`+lecID+`"],"topic":"Mạng","difficulty":"EASY","count":11}`), 422)

	// bulk + gen-stats trên hai câu PENDING của một lô.
	jobID := uuid.NewString()
	var bq [2]string
	for i := range bq {
		if err := db.QueryRow(ctx, `insert into question_bank (course_id, type, title, topic, difficulty, stem, answer_key, review_status, origin, created_by, ai_job_id)
			values ($1, 'TRUE_FALSE', 'bulk', 'Lô', 'EASY', $2, '{"value":true}', 'PENDING', 'EXTRACTED', $3, $4) returning id`, x.cid, "Nhận định lô "+uuid.NewString()[:6], gvID, jobID).Scan(&bq[i]); err != nil {
			r.t.Fatal(err)
		}
	}
	bulk := func(tok, body string) call { return call{method: "POST", path: Q + "/bulk", token: tok, body: body} }
	r.must(bulk(x.gv, `{"action":"SET_TAGS","ids":["`+bq[0]+`","`+uuid.NewString()+`"],"topic":"Nhãn mới"}`), 200)
	r.must(bulk(x.ta, `{"action":"APPROVE","ids":["`+bq[0]+`"]}`), 200)
	r.must(bulk(x.gv, `{"action":"REJECT","ids":["`+bq[1]+`"]}`), 200)
	r.must(bulk("", `{"action":"APPROVE","ids":["`+bq[0]+`"]}`), 401)
	r.must(bulk(x.sv, `{"action":"APPROVE","ids":["`+bq[0]+`"]}`), 403)
	r.must(call{method: "POST", path: "/api/v1/courses/khong-phai-uuid/questions/bulk", token: x.gv, body: `{"action":"APPROVE","ids":["` + bq[0] + `"]}`}, 404)
	r.must(call{method: "POST", path: "/api/v1/courses/" + x.arch + "/questions/bulk", token: x.gv, body: `{"action":"APPROVE","ids":["` + bq[0] + `"]}`}, 409) // COURSE_ARCHIVED
	r.must(bulk(x.gv, `{"action":"DELETE","ids":["`+bq[0]+`"]}`), 422)
	r.must(call{method: "GET", path: Q + "/gen-stats?job_id=" + jobID, token: x.gv}, 200)
	r.must(call{method: "GET", path: Q + "/gen-stats?job_id=" + jobID}, 401)
	r.must(call{method: "GET", path: Q + "/gen-stats?job_id=" + jobID, token: x.sv}, 403)
	r.must(call{method: "GET", path: Q + "/gen-stats?job_id=" + uuid.NewString(), token: x.gv}, 404)
	r.must(call{method: "GET", path: Q + "/gen-stats?job_id=khong", token: x.gv}, 422)

	// 13, 14: lịch sử, điểm yếu.
	r.must(call{method: "GET", path: p + "/history?limit=5", token: x.sv}, 200)
	r.must(call{method: "GET", path: p + "/history"}, 401)
	r.must(call{method: "GET", path: p + "/history", token: x.gv}, 403)
	r.must(call{method: "GET", path: bad + "/history", token: x.sv}, 404)
	r.must(call{method: "GET", path: p + "/history?cursor=@@@", token: x.sv}, 422)
	r.must(call{method: "GET", path: p + "/weakness", token: x.sv}, 200)
	r.must(call{method: "GET", path: p + "/weakness"}, 401)
	r.must(call{method: "GET", path: p + "/weakness", token: x.gv}, 403)
	r.must(call{method: "GET", path: bad + "/weakness", token: x.sv}, 404)
}
