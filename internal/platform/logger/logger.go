package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strconv"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/reqcontext"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/sanitize"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

type ctxHandler struct {
	slog.Handler
}

func (h ctxHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := reqcontext.RequestIDFromContext(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

func Setup() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:       parseLevel(os.Getenv("LOG_LEVEL")),
		ReplaceAttr: replaceAttr,
	}

	var w io.Writer = os.Stdout
	if path := os.Getenv("LOG_FILE"); path != "" {
		w = io.MultiWriter(os.Stdout, &lumberjack.Logger{
			Filename:   path,
			MaxSize:    maxSizeMB(),
			MaxBackups: backupCount(),
			Compress:   true,
		})
	}

	var base slog.Handler
	if envBool("LOG_JSON", true) {
		base = slog.NewJSONHandler(w, opts)
	} else {
		base = slog.NewTextHandler(w, opts)
	}

	logger := slog.New(ctxHandler{base})
	slog.SetDefault(logger)
	return logger
}

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey && a.Value.Kind() == slog.KindTime {
		a.Value = slog.TimeValue(a.Value.Time().UTC())
		return a
	}
	return sanitize.Attr(groups, a)
}

func parseLevel(s string) slog.Level {
	switch s {
	case "DEBUG":
		return slog.LevelDebug
	case "WARNING", "WARN":
		return slog.LevelWarn
	case "ERROR", "CRITICAL":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func maxSizeMB() int {
	if v, err := strconv.Atoi(os.Getenv("LOG_MAX_MB")); err == nil && v > 0 {
		return v
	}
	return 10
}

func backupCount() int {
	if v, err := strconv.Atoi(os.Getenv("LOG_BACKUP_COUNT")); err == nil && v > 0 {
		return v
	}
	return 5
}

func Error(msg string, args ...any) { slog.Default().Error(msg, args...) }
func Warn(msg string, args ...any)  { slog.Default().Warn(msg, args...) }
func Info(msg string, args ...any)  { slog.Default().Info(msg, args...) }
func Debug(msg string, args ...any) { slog.Default().Debug(msg, args...) }

func Fatal(msg string, args ...any) {
	slog.Default().Error(msg, args...)
	os.Exit(1)
}
