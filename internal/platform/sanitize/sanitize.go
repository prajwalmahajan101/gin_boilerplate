package sanitize

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

const redacted = "***REDACTED***"

const maxStringLen = 512

var sensitiveKeys = []string{
	"secret", "token", "credential", "api_key", "apikey", "bearer",
	"jwt", "password", "passwd", "x-super-admin-secret", "authorization",
}

var sensitiveShortKeys = []string{"key", "auth", "pan", "dob"}

var shortKeyRE = regexp.MustCompile(`(?i)\b(` + strings.Join(sensitiveShortKeys, "|") + `)\b`)

var valuePatterns = []*regexp.Regexp{
	regexp.MustCompile(`\b\d{12}\b`),
	regexp.MustCompile(`\b[A-Z]{5}\d{4}[A-Z]\b`),
	regexp.MustCompile(`\b[A-Z]{4}0[A-Z0-9]{6}\b`),
	regexp.MustCompile(`\b[6-9]\d{9}\b`),
	regexp.MustCompile(`\b\d{9,18}\b`),
	regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`),
	regexp.MustCompile(`\b[\w.+-]+@[\w-]+\.[\w.-]+\b`),
	regexp.MustCompile(`\b\d{2}[/-]\d{2}[/-]\d{4}\b`),
}

func isSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	for _, s := range sensitiveKeys {
		if strings.Contains(k, s) {
			return true
		}
	}
	return shortKeyRE.MatchString(k)
}

func String(s string) string {
	for _, re := range valuePatterns {
		s = re.ReplaceAllString(s, redacted)
	}
	s = escapeControl(s)
	if len(s) > maxStringLen {
		s = s[:maxStringLen] + "[truncated]"
	}
	return s
}

func escapeControl(s string) string {
	r := strings.NewReplacer("\n", `\n`, "\r", `\r`, "\t", `\t`)
	return r.Replace(s)
}

func Value(v any) any {
	if s, ok := v.(string); ok {
		return String(s)
	}
	return v
}

func Attr(_ []string, a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindGroup {
		return a
	}
	if isSensitiveKey(a.Key) {
		a.Value = slog.StringValue(redacted)
		return a
	}
	if a.Value.Kind() == slog.KindString {
		a.Value = slog.StringValue(String(a.Value.String()))
		return a
	}
	if a.Value.Kind() == slog.KindAny {
		switch v := a.Value.Any().(type) {
		case error:
			a.Value = slog.StringValue(String(v.Error()))
		case fmt.Stringer:
			a.Value = slog.StringValue(String(v.String()))
		}
	}
	return a
}
