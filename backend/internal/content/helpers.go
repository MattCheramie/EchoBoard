package content

import (
	"strings"
	"time"
)

// fmtTime formats a time as RFC3339 UTC, or "" for the zero time.
func fmtTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// formatTime formats an optional time, returning "" for nil/zero.
func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return fmtTime(*t)
}

// parseTime parses an RFC3339 string, returning the zero time on empty/invalid.
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func join(parts []string, sep string) string { return strings.Join(parts, sep) }

// dedupe returns the non-empty, trimmed, de-duplicated values in input order.
func dedupe(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
