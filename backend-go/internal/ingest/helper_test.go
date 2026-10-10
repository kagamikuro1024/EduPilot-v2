package ingest_test

import (
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/platform/outbox"
)

func outboxMsg(payload []byte) outbox.Message {
	return outbox.Message{ID: uuid.New(), Topic: "job.enqueue", Payload: payload}
}
