// Command velocity-mcp-mobile is an Android **testing** MCP server.
//
// It exposes Espresso- and Compose-test-style verbs to an LLM agent, plus
// the minimum supporting infrastructure tests need (animation control,
// permissions, app launch/clear, screenshot/diff, layout, logcat).
//
// Runtime requirements (must be on PATH):
//   - adb (always)
//   - android (Google's agent CLI; recommended — improves screen_resolve
//     and screen_layout when present, gracefully falls back to adb otherwise)
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
	"github.com/randheer094/velocity-mcp-mobile/internal/input"
	"github.com/randheer094/velocity-mcp-mobile/internal/runner"
	"github.com/randheer094/velocity-mcp-mobile/internal/system"
	apptest "github.com/randheer094/velocity-mcp-mobile/internal/testing"
	"github.com/randheer094/velocity-mcp-mobile/internal/tools"
	"github.com/randheer094/velocity-mcp-mobile/internal/ui"
)

// version is overridable at build time via -ldflags.
var version = "0.3.0"

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
	cli := androidcli.New(r) // optional
	if !cli.Available() {
		log.Printf("`android` agent CLI not found — `screen_resolve` will be unavailable and `screen_layout`/`find_*` will use UIAutomator instead. Install: https://developer.android.com/tools/agents/android-cli")
	}

	deps := &tools.Deps{
		Adb:        adbClient,
		AndroidCLI: cli,
		Resolver:   device.NewResolver(adbClient, cli, 5*time.Second),
		Apps:       apps.New(adbClient, cli),
		Layout:     ui.NewLayoutClient(adbClient, cli),
		Screenshot: ui.NewScreenshotClient(adbClient, cli),
		Input:      input.New(adbClient),
		Logs:       diagnostics.NewLogClient(adbClient),
		Screen:     system.NewScreenClient(adbClient),
		Animations: system.NewAnimationsClient(adbClient),
	}
	// Testing surface (Espresso/Compose-style verbs) layers on top of the
	// existing layout + input clients.
	deps.Tester = apptest.New(deps.Layout, deps.Input)
	deps.Intents = apptest.NewIntentRecorder(deps.Logs)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "android-test-mcp",
		Title:   "Android Testing MCP Server",
		Version: version,
	}, nil)
	tools.RegisterAll(server, deps)

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigC := make(chan os.Signal, 2)
	signal.Notify(sigC, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigC
		log.Printf("shutting down")
		cancel()
	}()

	if err := server.Run(rootCtx, &mcp.StdioTransport{}); err != nil {
		log.Printf("server exited: %v", err)
		os.Exit(1)
	}
}
