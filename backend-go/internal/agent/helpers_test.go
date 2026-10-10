package agent

import (
	"io"
	"log/slog"
	"reflect"
	"strings"
)

func newLog(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func reflectMethod(v any, name string) (reflect.Method, bool) {
	return reflect.TypeOf(v).MethodByName(name)
}

var _ = strings.Contains
