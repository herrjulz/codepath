// Package redact masks high-confidence secrets in model-supplied text and
// in source lines served to the local viewer.
package redact

import (
	"regexp"
	"strings"
)

var (
	pemPattern    = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`)
	tokenPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
		regexp.MustCompile(`\bghp_[A-Za-z0-9]{20,}\b`),
		regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{20,}\b`),
		regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{10,}\b`),
		regexp.MustCompile(`\bsk-[A-Za-z0-9]{20,}\b`),
	}
)

// Text replaces PEM blocks and known token prefixes. Short demo values such
// as "demo-only" are left unchanged.
func Text(s string) (string, bool) {
	out := pemPattern.ReplaceAllString(s, "[redacted]")
	for _, re := range tokenPatterns {
		out = re.ReplaceAllString(out, "[redacted]")
	}
	return out, out != s
}

// Line redacts a single source line without joining or splitting lines, so
// editor line numbers stay aligned with the file.
func Line(s string) (string, bool) {
	if strings.Contains(s, "PRIVATE KEY") || strings.Contains(s, "BEGIN OPENSSH") {
		return "[redacted]", true
	}
	return Text(s)
}
