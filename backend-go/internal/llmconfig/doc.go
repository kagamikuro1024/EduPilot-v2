// Package llmconfig: dịch vụ cấu hình cổng LLM (nhà cung cấp, mô hình, tuyến theo tác vụ, ngân sách) — FEAT-llm-gateway
// US-P1-01. Khoá API chỉ ghi: DB giữ bản mã AES-256-GCM (AAD = "llm_providers:"+id), mọi đầu ra chỉ có HasKey / KeyStatus,
// hàm duy nhất trả khoá rõ là Resolver.DecryptKey (chỉ internal/llm gọi). Không có SQL ở đây (sqlc, internal/store/queries/llm.sql);
// không float: tiền là decimal, tham số tuyến là json.Number.
package llmconfig
