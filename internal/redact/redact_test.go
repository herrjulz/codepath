package redact

import (
	"strings"
	"testing"
)

func TestTextRedactsTokensAndKeepsDemoValues(t *testing.T) {
	in := "token AKIAIOSFODNN7EXAMPLE password demo-only"
	out, changed := Text(in)
	if !changed || strings.Contains(out, "AKIA") || !strings.Contains(out, "demo-only") {
		t.Fatalf("changed=%v out=%s", changed, out)
	}
	pem := "-----BEGIN RSA PRIVATE KEY-----\nabc\n-----END RSA PRIVATE KEY-----"
	out, changed = Text(pem)
	if !changed || strings.Contains(out, "BEGIN") {
		t.Fatalf("pem out=%s", out)
	}
	line, changed := Line("const key = \"sk-" + strings.Repeat("a", 24) + "\"")
	if !changed || strings.Contains(line, "sk-") || strings.Contains(line, "\n") {
		t.Fatalf("line=%q changed=%v", line, changed)
	}
	plain, changed := Line("return password == demoPassword")
	if changed || plain != "return password == demoPassword" {
		t.Fatalf("plain=%q changed=%v", plain, changed)
	}
}
