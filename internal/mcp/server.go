// Package mcp adapts CodePath to MCP. The preferred path is Streamable HTTP
// mounted on codepath serve at /mcp. ServeStdio remains a thin forwarder for
// clients that still spawn a local stdio process.
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"codepath/internal/reports"
	"codepath/internal/service"
)

const toolDescription = "Save an interactive CodePath walkthrough for the current repository. Before calling this tool, inspect the relevant code and construct a factual sequence of steps with concrete repository-relative file paths and line spans. Include realistic but synthetic data transformations for each scenario. Do not claim runtime observation. Return the local report URL for the user to open."

// Options configures the stdio forwarder. Repo is the canonical repository
// root and is not taken from tool arguments.
type Options struct {
	Repo      string
	ServerURL string
	Token     string
	TokenErr  error
}

type toolResult struct {
	ReportID string   `json:"reportId"`
	URL      string   `json:"url"`
	Warnings []string `json:"warnings"`
}

// LocalServer builds an MCP server whose create_walkthrough tool calls svc
// in-process. Used by the HTTP /mcp endpoint on codepath serve.
func LocalServer(svc *service.Service) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "codepath", Version: "0.1.0"}, nil)
	local := &localAdapter{svc: svc}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_walkthrough",
		Description: toolDescription,
	}, local.create)
	return server
}

// HTTPHandler serves Streamable HTTP MCP for svc.
func HTTPHandler(svc *service.Service) http.Handler {
	server := LocalServer(svc)
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{Stateless: true})
}

type localAdapter struct {
	svc *service.Service
}

func (a *localAdapter) create(ctx context.Context, _ *mcp.CallToolRequest, in reports.CreateRequest) (*mcp.CallToolResult, toolResult, error) {
	body, err := json.Marshal(in)
	if err != nil {
		return nil, toolResult{}, err
	}
	created, err := a.svc.Create(ctx, body)
	if err != nil {
		return nil, toolResult{}, err
	}
	if created.Warnings == nil {
		created.Warnings = []string{}
	}
	return nil, toolResult{
		ReportID: created.ReportID,
		URL:      created.URL,
		Warnings: created.Warnings,
	}, nil
}

type forwardAdapter struct {
	repo     string
	base     string
	addrHint string
	token    string
	tokenErr error
	client   *http.Client
}

// NewServer builds the stdio forwarder MCP server (HTTP client to serve).
func NewServer(opts Options) (*mcp.Server, error) {
	base, addr, err := normalizeServerURL(opts.ServerURL)
	if err != nil {
		return nil, err
	}
	a := &forwardAdapter{
		repo:     opts.Repo,
		base:     base,
		addrHint: addr,
		token:    opts.Token,
		tokenErr: opts.TokenErr,
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "codepath", Version: "0.1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_walkthrough",
		Description: toolDescription,
	}, a.create)
	return server, nil
}

// ServeStdio runs the forwarder on stdin and stdout.
func ServeStdio(ctx context.Context, opts Options) error {
	log.SetOutput(os.Stderr)
	server, err := NewServer(opts)
	if err != nil {
		return err
	}
	return server.Run(ctx, &mcp.StdioTransport{})
}

func (a *forwardAdapter) create(ctx context.Context, _ *mcp.CallToolRequest, in reports.CreateRequest) (*mcp.CallToolResult, toolResult, error) {
	if a.tokenErr != nil {
		return nil, toolResult{}, fmt.Errorf("%s. Start it with: %s", a.tokenErr.Error(), a.startCommand())
	}
	if err := a.ping(ctx); err != nil {
		return nil, toolResult{}, err
	}
	body, err := json.Marshal(in)
	if err != nil {
		return nil, toolResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.base+"/api/reports", bytes.NewReader(body))
	if err != nil {
		return nil, toolResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Content-Type", "application/json")
	res, err := a.client.Do(req)
	if err != nil {
		return nil, toolResult{}, fmt.Errorf("CodePath server is not running. Start it with: %s", a.startCommand())
	}
	defer res.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(res.Body, reports.MaxBodyBytes))
	if err != nil {
		return nil, toolResult{}, err
	}
	if res.StatusCode >= 300 {
		return nil, toolResult{}, fmt.Errorf("create_walkthrough was rejected (%s): %s", res.Status, strings.TrimSpace(string(payload)))
	}
	var out toolResult
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, toolResult{}, fmt.Errorf("codepath server returned an unreadable response")
	}
	if out.Warnings == nil {
		out.Warnings = []string{}
	}
	return nil, out, nil
}

func (a *forwardAdapter) ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.base+"/health", nil)
	if err != nil {
		return err
	}
	res, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("CodePath server is not running. Start it with: %s", a.startCommand())
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("CodePath server at %s returned %s. Start it with: %s", a.base, res.Status, a.startCommand())
	}
	return nil
}

func (a *forwardAdapter) startCommand() string {
	return fmt.Sprintf("codepath serve --repo %q --addr %s", a.repo, a.addrHint)
}

func normalizeServerURL(raw string) (string, string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.Scheme != "http" || u.User != nil {
		return "", "", fmt.Errorf("--server must be an http loopback URL such as http://127.0.0.1:4317")
	}
	host := u.Hostname()
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return "", "", fmt.Errorf("--server must point at loopback, not %s", host)
	}
	if u.Path != "" && u.Path != "/" {
		return "", "", fmt.Errorf("--server must not include a path")
	}
	port := u.Port()
	if port == "" {
		port = "80"
	}
	addr := net.JoinHostPort(host, port)
	return strings.TrimRight(u.Scheme+"://"+u.Host, "/"), addr, nil
}
