package practice

import (
	"context"
	"encoding/json"
)

// Schema của hai việc LLM, để test bản ghi demo kiểm đầu ra như `llm.Gateway.Structured` (sai schema = lỗi nhà cung cấp).
const (
	ExtractSchema  = extractSchema
	GenerateSchema = generateSchema
)

var StemKey = stemKey

var DefaultRetryWait = defaultRetryWait

func CallWorker(w *Worker, ctx context.Context, system, user, schema string) (json.RawMessage, error) {
	return w.call(ctx, system, user, schema)
}

var ShuffleOptions = shuffleOptions
