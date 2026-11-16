package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	official "github.com/modelcontextprotocol/go-sdk/mcp"

	"codepath/internal/reports"
	"codepath/internal/testutil"
)

func TestReportFlow(t *testing.T) {
	repo, rel := testutil.RepoWithFile(t)
	svc, token := testutil.NewService(t, repo, "http://127.0.0.1")
	ts := httptest.NewServer(Handler(svc, token, nil))
	defer ts.Close()
	svc.BaseURL = ts.URL

	res, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("health %s", res.Status)
	}
	_ = res.Body.Close()

	body := testutil.MustJSON(t, testutil.Sample(rel))
	post := func(payload []byte, mutate func(*http.Request)) (*http.Response, []byte) {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/reports", bytes.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		if mutate != nil {
			mutate(req)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		return res, data
	}

	res, _ = post(body, func(r *http.Request) {
		r.Host = "evil.example"
		r.Header.Set("Authorization", "Bearer "+token)
	})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("evil host status %s", res.Status)
	}

	res, _ = post(body, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Origin", "http://evil.example")
	})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("evil origin status %s", res.Status)
	}

	res, _ = post(body, nil)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("missing token status %s", res.Status)
	}

	res, payload := post(body, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+token)
	})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create %s: %s", res.Status, payload)
	}
	var created struct {
		ReportID string   `json:"reportId"`
		URL      string   `json:"url"`
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(payload, &created); err != nil {
		t.Fatal(err)
	}
	if !reports.ValidID(created.ReportID) || !strings.HasPrefix(created.URL, ts.URL+"/reports/") {
		t.Fatalf("created: %+v", created)
	}

	res, err = http.Get(ts.URL + "/api/reports/" + created.ReportID)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get %s", res.Status)
	}
	_ = res.Body.Close()

	res, err = http.Get(ts.URL + "/api/reports")
	if err != nil {
		t.Fatal(err)
	}
	listBody, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if !strings.Contains(string(listBody), created.ReportID) {
		t.Fatalf("list: %s", listBody)
	}

	sourceURL := ts.URL + "/api/reports/" + created.ReportID + "/source?stepId=start&refIndex=0"
	res, err = http.Get(sourceURL)
	if err != nil {
		t.Fatal(err)
	}
	if res.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("cache control %q", res.Header.Get("Cache-Control"))
	}
	sourceBody, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if !strings.Contains(string(sourceBody), "func main") {
		t.Fatalf("source: %s", sourceBody)
	}

	if err := os.Remove(filepath.Join(repo, rel)); err != nil {
		t.Fatal(err)
	}
	res, err = http.Get(sourceURL)
	if err != nil {
		t.Fatal(err)
	}
	missingBody, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK || !strings.Contains(string(missingBody), `"missing":true`) {
		t.Fatalf("missing source %s: %s", res.Status, missingBody)
	}

	bad := testutil.Sample(rel)
	bad.Language = "python"
	res, errBody := post(testutil.MustJSON(t, bad), func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+token)
	})
	if res.StatusCode != http.StatusBadRequest || !strings.Contains(string(errBody), "language") {
		t.Fatalf("validation %s: %s", res.Status, errBody)
	}

	traversal := testutil.Sample("../secret.go")
	res, errBody = post(testutil.MustJSON(t, traversal), func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+token)
	})
	if res.StatusCode != http.StatusBadRequest || !strings.Contains(string(errBody), "parent") {
		t.Fatalf("traversal %s: %s", res.Status, errBody)
	}
}

func TestMCPEndpoint(t *testing.T) {
	repo, rel := testutil.RepoWithFile(t)
	svc, token := testutil.NewService(t, repo, "http://127.0.0.1")
	ts := httptest.NewServer(Handler(svc, token, nil))
	defer ts.Close()
	svc.BaseURL = ts.URL

	ctx := context.Background()
	client := official.NewClient(&official.Implementation{Name: "test", Version: "0"}, nil)
	session, err := client.Connect(ctx, &official.StreamableClientTransport{Endpoint: ts.URL + "/mcp"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &official.CallToolParams{
		Name:      "create_walkthrough",
		Arguments: testutil.Sample(rel),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("tool error: %+v", result.Content)
	}
	encoded, _ := json.Marshal(result.StructuredContent)
	if !strings.Contains(string(encoded), ts.URL+"/reports/") {
		t.Fatalf("result: %s", encoded)
	}
}
