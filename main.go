// Command velocity-mcp-mobile is an Android-only Model Context Protocol
// server. It runs over stdio and exposes ~50 tools for end-to-end Android
// app verification & testing on devices and emulators.
//
// Runtime requirements (must be on PATH):
//   - adb (always)
//   - android (Google's agent CLI; recommended — many tools fall back to adb when it is missing)
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/randheer094/velocity-mcp-mobile/internal/adb"
	"github.com/randheer094/velocity-mcp-mobile/internal/androidcli"
	"github.com/randheer094/velocity-mcp-mobile/internal/apps"
	"github.com/randheer094/velocity-mcp-mobile/internal/device"
	"github.com/randheer094/velocity-mcp-mobile/internal/diagnostics"
	"github.com/randheer094/velocity-mcp-mobile/internal/files"
	"github.com/randheer094/velocity-mcp-mobile/internal/input"
	"github.com/randheer094/velocity-mcp-mobile/internal/maintenance"
	"github.com/randheer094/velocity-mcp-mobile/internal/runner"
	"github.com/randheer094/velocity-mcp-mobile/internal/system"
	"github.com/randheer094/velocity-mcp-mobile/internal/tools"
	"github.com/randheer094/velocity-mcp-mobile/internal/ui"
)

// version is overridable at build time via -ldflags.
var version = "0.1.0"

func main() {
	listTools := flag.Bool("list-tools", false, "print the registered tool names and exit")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("velocity-mcp-mobile", version)
		return
	}
	if *listTools {
		for _, n := range tools.Catalog() {
			fmt.Println(n)
		}
		return
	}

	// Logging goes to stderr so it doesn't pollute the stdio MCP transport.
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ltime | log.Lmicroseconds)

	r := runner.New(30*time.Second, 0)
	adbClient, err := adb.New(r)
	if err != nil {
		log.Fatalf("startup: %v", err)
	}
	cli := androidcli.New(r) // OK if absent
	if !cli.Available() {
		log.Printf("`android` agent CLI not found — some tools (emulator_*, screen_resolve, docs_*) will be unavailable. Install: https://developer.android.com/tools/agents/android-cli")
	}

	deps := &tools.Deps{
		Adb:         adbClient,
		AndroidCLI:  cli,
		Resolver:    device.NewResolver(adbClient, cli, 5*time.Second),
		Apps:        apps.New(adbClient, cli),
		Layout:      ui.NewLayoutClient(adbClient, cli),
		Screenshot:  ui.NewScreenshotClient(adbClient, cli),
		Recorder:    ui.NewRecorder(adbClient),
		Input:       input.New(adbClient),
		Logs:        diagnostics.NewLogClient(adbClient),
		Dumpsys:     diagnostics.NewDumpsysClient(adbClient),
		Trace:       diagnostics.NewTraceClient(adbClient),
		Files:       files.New(adbClient),
		Screen:      system.NewScreenClient(adbClient),
		Animations:  system.NewAnimationsClient(adbClient),
		Doze:        system.NewDozeClient(adbClient),
		Time:        system.NewTimeClient(adbClient),
		Network:     system.NewNetworkClient(adbClient),
		Location:    system.NewLocationClient(adbClient),
		Maintenance: maintenance.New(adbClient),
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "android-mcp",
		Title:   "Android MCP Server",
		Version: version,
	}, nil)
	tools.RegisterAll(server, deps)

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown: SIGINT / SIGTERM cancels root context and stops any
	// active screen recordings cleanly.
	sigC := make(chan os.Signal, 2)
	signal.Notify(sigC, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigC
		log.Printf("shutting down")
		shutdownCtx, scancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer scancel()
		deps.Recorder.StopAll(shutdownCtx)
		cancel()
	}()

	if err := server.Run(rootCtx, &mcp.StdioTransport{}); err != nil {
		log.Printf("server exited: %v", err)
		os.Exit(1)
	}
}
