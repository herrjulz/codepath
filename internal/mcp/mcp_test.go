package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	official "github.com/modelcontextprotocol/go-sdk/mcp"

	"codepath/internal/service"
	"codepath/internal/testutil"
)

func TestCreateWalkthrough(t *testing.T) {
	repo, rel := testutil.RepoWithFile(t)
	svc, token := testutil.NewService(t, repo, "http://127.0.0.1")
	ts := httptest.NewServer(apiOnly(svc, token))
	defer ts.Close()
	svc.BaseURL = ts.URL

	ctx := context.Background()
	session := connect(t, ctx, Options{Repo: repo, ServerURL: ts.URL, Token: token})
	defer session.Close()

	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(listed)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "create_walkthrough") || strings.Contains(string(raw), "repositoryRoot") {
		t.Fatalf("tool schema: %s", raw)
	}

	result, err := session.CallTool(ctx, &official.CallToolParams{
		Name:      "create_walkthrough",
		Arguments: testutil.Sample(rel),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("tool error: %s", texts(result))
	}
	encoded, _ := json.Marshal(result.StructuredContent)
	if !strings.Contains(string(encoded), ts.URL+"/reports/") {
		t.Fatalf("result: %s", encoded)
	}

	bad := testutil.Sample(rel)
	bad.Scenarios[0].Examples[0].Kind = "observed"
	result, err = session.CallTool(ctx, &official.CallToolParams{
		Name:      "create_walkthrough",
		Arguments: bad,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError || !strings.Contains(texts(result), "synthetic") {
		t.Fatalf("expected validation tool error, got %s", texts(result))
	}
}

func TestHTTPCreateWalkthrough(t *testing.T) {
	repo, rel := testutil.RepoWithFile(t)
	svc, _ := testutil.NewService(t, repo, "http://127.0.0.1")
	ts := httptest.NewServer(HTTPHandler(svc))
	defer ts.Close()
	svc.BaseURL = ts.URL

	ctx := context.Background()
	client := official.NewClient(&official.Implementation{Name: "test", Version: "0"}, nil)
	session, err := client.Connect(ctx, &official.StreamableClientTransport{Endpoint: ts.URL}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(listed)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "create_walkthrough") {
		t.Fatalf("tool schema: %s", raw)
	}

	result, err := session.CallTool(ctx, &official.CallToolParams{
		Name:      "create_walkthrough",
		Arguments: testutil.Sample(rel),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("tool error: %s", texts(result))
	}
	encoded, _ := json.Marshal(result.StructuredContent)
	if !strings.Contains(string(encoded), ts.URL+"/reports/") {
		t.Fatalf("result: %s", encoded)
	}
}

func TestOfflineServer(t *testing.T) {
	repo, _ := testutil.RepoWithFile(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	ctx := context.Background()
	session := connect(t, ctx, Options{
		Repo:      repo,
		ServerURL: "http://" + addr,
		Token:     strings.Repeat("ab", 32),
	})
	defer session.Close()
	result, err := session.CallTool(ctx, &official.CallToolParams{
		Name:      "create_walkthrough",
		Arguments: testutil.Sample("main.go"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError || !strings.Contains(texts(result), "codepath serve") {
		t.Fatalf("offline error: %s", texts(result))
	}
}

func TestStdioIsProtocolOnly(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "codepath")
	build := exec.Command("go", "build", "-o", bin, "codepath/cmd/codepath")
	build.Dir = testutil.ModuleRoot(t)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	cmd := exec.Command(bin, "mcp", "--repo", repo, "--server", "http://127.0.0.1:1", "--data-dir", t.TempDir())
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()
	if _, err := io.WriteString(stdin, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"codepath-test","version":"0"}}}`+"\n"); err != nil {
		t.Fatal(err)
	}
	line := make(chan string, 1)
	go func() {
		reader := bufio.NewReader(stdout)
		text, err := reader.ReadString('\n')
		if err != nil {
			line <- "read error: " + err.Error()
			return
		}
		line <- text
	}()
	select {
	case text := <-line:
		var msg map[string]any
		if err := json.Unmarshal([]byte(text), &msg); err != nil {
			t.Fatalf("stdout is not JSON-RPC: %s", text)
		}
		if msg["jsonrpc"] != "2.0" {
			t.Fatalf("stdout: %s", text)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for initialize response")
	}
}

func apiOnly(svc *service.Service, token string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("POST /api/reports", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		result, err := svc.Create(r.Context(), body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(result)
	})
	return mux
}

func connect(t *testing.T, ctx context.Context, opts Options) *official.ClientSession {
	t.Helper()
	srv, err := NewServer(opts)
	if err != nil {
		t.Fatal(err)
	}
	client := official.NewClient(&official.Implementation{Name: "test", Version: "0"}, nil)
	left, right := official.NewInMemoryTransports()
	if _, err := srv.Connect(ctx, left, nil); err != nil {
		t.Fatal(err)
	}
	session, err := client.Connect(ctx, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func texts(result *official.CallToolResult) string {
	var b strings.Builder
	for _, content := range result.Content {
		if text, ok := content.(*official.TextContent); ok {
			b.WriteString(text.Text)
		}
	}
	return b.String()
}
