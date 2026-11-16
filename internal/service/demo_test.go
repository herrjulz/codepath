package service_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codepath/internal/testutil"
)

func TestDemoReportResolves(t *testing.T) {
	root := testutil.ModuleRoot(t)
	body, err := os.ReadFile(filepath.Join(root, "examples", "authentication-flow.json"))
	if err != nil {
		t.Fatal(err)
	}
	svc, _ := testutil.NewService(t, root, "http://127.0.0.1:4317")
	created, err := svc.Create(context.Background(), body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(created.URL, "/reports/") {
		t.Fatalf("url %s", created.URL)
	}
	report, err := svc.Get(created.ReportID)
	if err != nil {
		t.Fatal(err)
	}
	if report.Repository.DisplayName == "" || len(report.Scenarios) < 3 {
		t.Fatalf("report was not enriched: %+v", report.Repository)
	}
	want := map[string]string{
		"http_login":     "func Login",
		"user_lookup":    "func lookupUser",
		"password_check": "func checkPassword",
		"build_claims":   "func buildClaims",
		"sign_jwt":       "func signJWT",
		"http_ok":        "func writeJSON",
		"missing_user":   "invalid credentials",
		"wrong_password": "checkPassword",
	}
	for id, snippet := range want {
		view, err := svc.Source(created.ReportID, id, 0)
		if err != nil {
			t.Fatal(id, err)
		}
		if view.Missing || view.Stale || !strings.Contains(view.Content, snippet) {
			t.Fatalf("%s view: %+v", id, view)
		}
		if report.Steps[0].CodeRefs[0].FileSha256 == "" {
			t.Fatal("missing file hash")
		}
	}
	reopened, err := svc.Get(created.ReportID)
	if err != nil || reopened.ID != created.ReportID {
		t.Fatalf("reopen: %+v %v", reopened, err)
	}
}
