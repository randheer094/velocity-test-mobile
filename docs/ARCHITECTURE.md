# Architecture

`velocity-mcp-mobile` is an Android **testing** MCP server. An LLM agent connects via stdio and calls Espresso-/Compose-test-style verbs against a connected device or emulator.

```
        ┌──────────────────────────────────────────────────────────────────┐
        │ MCP client (Claude Desktop, Inspector, custom agent, …)         │
        └────────────────────────────────────────────────────────────────────┘
                                    │ JSON-RPC over stdio
                                    ▼
        ┌──────────────────────────────────────────────────────────────────┐
        │ velocity-mcp-mobile (Go binary, ~7 MB)                          │
        │                                                                  │
        │   internal/tools           ← MCP tool definitions / schemas      │
        │      ├── testing.go        ← matcher-bearing verbs (Server.AddTool, hand-built JSON Schema with $ref recursion)
        │      └── *.go              ← support tools (find / capture / log / app / animations)
        │                                                                  │
        │   internal/testing         ← Orchestrator                        │
        │      ├── assertions.go     ← AssertVisible / AssertWidthDp / …   │
        │      ├── actions.go        ← Click / TypeText / ScrollTo / …     │
        │      ├── sync.go           ← WaitUntilVisible / WaitForIdle      │
        │      └── intents.go        ← logcat-scrape Intent recorder       │
        │                                                                  │
        │   internal/matcher         ← shared selector vocabulary          │
        │      ├── matcher.go        ← Matcher struct + Match              │
        │      └── select.go         ← Find / FindAll over flattened tree  │
        │                                                                  │
        │   internal/ui              ← layout + screenshot + diff          │
        │   internal/input           ← tap/swipe/text/clipboard primitives │
        │   internal/apps            ← app launch / clear / permissions    │
        │   internal/diagnostics     ← logcat                              │
        │   internal/system          ← screen size / orientation / animations
        │   internal/device          ← discovery + getprop                 │
        │   internal/adb             ← `adb` subprocess wrapper            │
        │   internal/androidcli      ← `android` agent CLI wrapper         │
        │   internal/runner          ← single point for os/exec            │
        └──────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
        ┌──────────────────────────────────────────────────────────────────┐
        │ adb (always)         android (recommended)                       │
        │  ↓                    ↓                                          │
        │ Android device / emulator (UIAutomator + input + logcat)         │
        └──────────────────────────────────────────────────────────────────┘
```

## How a verb runs

A typical call — `assert_clickable_and_click({match:{text:"Submit"}})` — flows like this:

1. **MCP transport** — JSON-RPC arrives on stdin. The SDK dispatches to the registered handler.
2. **Schema validation** — the JSON Schema for the tool (built in `tools/testing_schema.go`) validates `device`, `match`, and any extras. Recursion in `match` is expressed via `$defs/matcher` + `$ref`.
3. **Device resolution** — `Resolver.Resolve(ctx, args.Device)` returns the target serial (auto-picked when only one device is connected).
4. **Tree snapshot** — `LayoutClient.Tree(ctx, dev)` runs `android layout --pretty` if the agent CLI is available, else `adb exec-out uiautomator dump /dev/tty`, parses, and returns the recursive `ui.Element`.
5. **Matcher evaluation** — `matcher.FindAll(root, m)` flattens the tree once, then evaluates every node by index against the matcher's local predicates plus tree-aware combinators (`HasAncestor`, `HasDescendant`, `HasParent`, `HasSibling`, `IsRoot`, `ParentIndex`, `CompletelyDisplayed`, `DisplayingAtLeastPercent`).
6. **Action dispatch** — once the target node is identified, the orchestrator computes the centre point and calls into `internal/input` which spawns `adb shell input tap …`.
7. **Result serialization** — the orchestrator returns a structured result (`AssertResult`, `ActionResult`, `WaitResult`). The handler renders it as `mcp.TextContent` with pretty-printed JSON.

## Layered design

The codebase is strictly layered so dependencies always point downward:

| Layer | Responsibility | Examples |
| --- | --- | --- |
| `runner` | The **only** place subprocesses are spawned. Centralises timeouts, byte caps, structured `*ExecError`. | `Run`, `Stream` |
| `adb` / `androidcli` | Typed wrappers over the runtime CLIs. | `c.Shell`, `c.ExecOut`, `c.KeyCombination` |
| `device` / `apps` / `ui` / `input` / `diagnostics` / `system` | Domain primitives — independent of MCP. | `LayoutClient.Tree`, `Apps.Launch`, `Input.PressKeyCombination` |
| `matcher` | JSON-friendly selector vocabulary + tree walking. Pure logic, no side effects. | `Match`, `FindAll`, `IsDisplayed` |
| `testing` | Orchestrator: composes layout + matcher + input into Espresso/Compose-style verbs. | `Click`, `AssertWidthDp`, `WaitUntilVisible`, `AssertAny` |
| `tools` | MCP tool registrations: schemas + handler glue. | `RegisterTesting`, `RegisterApp` |
| `main.go` | Wire-up: builds the dependency graph, runs the stdio server. |

Tests at every layer:

- `runner` — timeout/exit-code/byte-cap behaviour.
- `adb` — argv construction, keycode lookup including A-Z/0-9.
- `apps` — `dumpsys package` parsing, `launcher activities` parsing, run-as path safety.
- `device` — `adb devices -l` parsing, getprop parsing.
- `diagnostics` — meminfo / gfxinfo / battery parsers.
- `matcher` — every matcher field, tree-position predicates, duplicate-sibling correctness, instance disambiguation.
- `testing` — intent log scraping, idle-tree hashing.
- `ui` — screenshot diff, UIAutomator XML parsing, `android layout` JSON parsing, bounds parsing.
- `system` — wm size / density parsing.

## The matcher as a contract

Every testing tool takes the same `match` argument shape (see [`MATCHERS.md`](MATCHERS.md) for the field reference). This means an agent's prompt of *"verify Login is visible; verify Continue is clickable then click it; wait for Welcome"* maps to three self-contained tool calls with no shared element handles. The recursive `hasAncestor`/`hasDescendant`/`allOf` combinators let an agent express precise predicates declaratively.

The MCP SDK's Go-type schema generator can't follow recursive struct types, so matcher-bearing tools register via `Server.AddTool` with hand-built JSON Schema using `$defs/matcher` + `$ref`. The schema is what the LLM client sees during tool discovery.

## Sync model: poll-and-hash

Espresso's `IdlingResource` and Compose's `mainClock.advanceTimeBy(...)` operate inside the app process and have no external equivalent. We approximate:

- `wait_until_*` polls the layout tree at `intervalMs` until the predicate matches, or `timeoutMs` elapses.
- `wait_for_idle` snapshots the tree, hashes (class + text + label + resourceId + bounds + state flags), and waits until two consecutive snapshots match for an `idleWindowMs` window.

This is unavoidably heuristic; agents that need exact frame-by-frame timing must instrument the app under test.

## Intent recording (no stubbing)

`Espresso.intending(...).respondWith(...)` is also in-process and unavailable externally. We support the *recording* half:

- `intent_monitor_start` clears the device logcat buffer.
- `intent_list_captured` reads `ActivityManager` `START` lines from logcat, parses `act=`/`dat=`/`pkg=`/`cmp=` extras.
- `assert_intent_sent` / `assert_intent_count` test for matches.

This is approximate — implicit intents that don't surface in `ActivityManager` aren't captured, and `intending()` stubbing is impossible without instrumentation.

## Concurrency

- Subprocesses are bounded by `context.WithTimeout` (default 30s; tunable per call).
- The `IntentRecorder` uses a `sync.Mutex` over its session map — multiple devices can have independent recording windows.
- The MCP SDK serialises tool calls per session by default, so handlers don't generally race.

## Build & distribution

- Single `go build` produces a static binary.
- `go vet` and `gofmt` pass.
- All unit tests run in milliseconds (no device required).
- Cross-compile to macOS/Linux/Windows by setting `GOOS`/`GOARCH`.

## Failure modes the server surfaces clearly

| Condition | Behaviour |
| --- | --- |
| `adb` not on `PATH` | server logs and exits non-zero at startup. |
| `android` not on `PATH` | server logs once; tools that depend on it return an actionable error pointing at the install page; matcher tools fall back to UIAutomator. |
| No devices connected | every device-targeted tool returns `no devices connected; run \`adb devices\` to verify`. |
| Multiple devices, no `device` arg | tool returns the candidate list and asks for an explicit serial. |
| Empty matcher | tool returns `matcher is empty: supply at least one of …`. |
| `run-as` on a release build | tool returns `package is not debuggable; run-as is unavailable on release builds`. |
| `input keycombination` on Android < 12 | `perform_key_press` falls back to a plain keyevent and reports the missing modifier coverage. |
