package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"codepath/internal/config"
	"codepath/internal/mcp"
	"codepath/internal/server"
	"codepath/internal/service"
	"codepath/internal/source"
	"codepath/internal/store"
	"codepath/web"
)

func main() {
	log.SetOutput(os.Stderr)
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "serve":
		err = serve(os.Args[2:])
	case "mcp":
		err = runMCP(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Fprint(os.Stderr, usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "codepath: %v\n", err)
		os.Exit(1)
	}
}

const usage = `CodePath is a local viewer for code walkthroughs submitted by OpenCode.

Usage:
  codepath serve --repo <path> [--addr 127.0.0.1:4317] [--data-dir <path>] [--seed <report.json>]
  codepath mcp --repo <path> [--server http://127.0.0.1:4317] [--data-dir <path>]

serve starts the local web UI and report API. mcp speaks MCP over stdio and
forwards create_walkthrough to a running serve process. Start serve first.

Reports and the API token are stored per repository under the user config
directory unless --data-dir is set. serve creates the token file. mcp only
reads it. Both processes must use the same repository and data directory.

stdout is reserved for MCP protocol frames in mcp mode. Logs go to stderr.
`

func serve(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	repoFlag := fs.String("repo", "", "repository to read source from")
	addr := fs.String("addr", "127.0.0.1:4317", "loopback listen address")
	dataDirFlag := fs.String("data-dir", "", "directory for reports and the API token")
	seed := fs.String("seed", "", "create a report from this JSON file before listening")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := validateListenAddr(*addr); err != nil {
		return err
	}
	repo, err := config.CanonicalRepo(*repoFlag)
	if err != nil {
		return err
	}
	dataDir, err := config.DataDir(repo, *dataDirFlag)
	if err != nil {
		return err
	}
	token, err := config.EnsureToken(dataDir)
	if err != nil {
		return err
	}
	svc, err := newService(repo, dataDir, "http://"+displayHost(*addr))
	if err != nil {
		return err
	}
	if *seed != "" {
		body, err := os.ReadFile(*seed)
		if err != nil {
			return fmt.Errorf("seed: %w", err)
		}
		result, err := svc.Create(context.Background(), body)
		if err != nil {
			return fmt.Errorf("seed: %w", err)
		}
		fmt.Printf("Report: %s\n", result.URL)
		for _, warning := range result.Warnings {
			fmt.Fprintf(os.Stderr, "warning: %s\n", warning)
		}
	}
	assets, err := server.DistFS(web.Dist)
	if err != nil {
		return err
	}
	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           server.Handler(svc, token, assets),
		ReadHeaderTimeout: 5 * time.Second,
	}
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w (is another codepath serve already running?)", *addr, err)
	}
	errCh := make(chan error, 1)
	go func() {
		if err := httpServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	fmt.Printf("CodePath listening on http://%s\n", displayHost(*addr))
	fmt.Printf("Repository: %s\n", repo)
	fmt.Printf("Data: %s\n", dataDir)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func runMCP(args []string) error {
	log.SetOutput(os.Stderr)
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	repoFlag := fs.String("repo", "", "repository the walkthrough refers to")
	serverURL := fs.String("server", "http://127.0.0.1:4317", "codepath serve base URL")
	dataDirFlag := fs.String("data-dir", "", "directory that contains the API token")
	if err := fs.Parse(args); err != nil {
		return err
	}
	repo, err := config.CanonicalRepo(*repoFlag)
	if err != nil {
		return err
	}
	dataDir, err := config.DataDir(repo, *dataDirFlag)
	if err != nil {
		return err
	}
	token, tokenErr := config.ReadToken(dataDir)
	return mcp.ServeStdio(context.Background(), mcp.Options{
		Repo:      repo,
		ServerURL: *serverURL,
		Token:     token,
		TokenErr:  tokenErr,
	})
}

func newService(repo, dataDir, baseURL string) (*service.Service, error) {
	files, err := source.Open(repo)
	if err != nil {
		return nil, err
	}
	st, err := store.Open(dataDir)
	if err != nil {
		return nil, err
	}
	return &service.Service{
		RepoRoot: files.Root,
		Files:    files,
		Store:    st,
		BaseURL:  baseURL,
	}, nil
}

func validateListenAddr(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		return fmt.Errorf("--addr must be host:port, for example 127.0.0.1:4317")
	}
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("refusing to listen on %s: CodePath binds to loopback only", addr)
	}
	return nil
}

func displayHost(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if stringsHasColon(host) {
		return "[" + host + "]:" + port
	}
	return host + ":" + port
}

func stringsHasColon(s string) bool {
	for _, r := range s {
		if r == ':' {
			return true
		}
	}
	return false
}
