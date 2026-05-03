# velocity-test-mobile

An **Android testing** [Model Context Protocol](https://modelcontextprotocol.io) server, written in Go.

The server exposes **Espresso** ViewMatchers/ViewActions/ViewAssertions and **Jetpack Compose** test verbs (`onNodeWithText`, `assertIsDisplayed`, `performClick`, `waitUntilExists`, …) as **104 MCP tools** an LLM agent can call directly. Each tool takes a shared **matcher** object — the same vocabulary across find / assert / act / wait — so an agent's prompt of *"verify Login is visible; click Continue; wait for Welcome"* maps to three self-contained tool calls with no element handles to thread.

Internally the server walks the device's accessibility tree (UIAutomator XML or Google's [`android` agent CLI](https://developer.android.com/tools/agents/android-cli) JSON), applies the matcher, and dispatches `adb shell input` actions. **No in-process instrumentation** — no companion APK, no Espresso runtime on the device.

- **Single static Go binary**, ~7 MB, sub-10 ms cold start.
- **No telemetry**, no phone-home — fully local.
- **Testing-only**: lifecycle/deployment/diagnostics surface intentionally absent. Just the Espresso/Compose API plus the minimum supporting infrastructure tests need (animations off, app launch/clear, permissions, intents, screenshot/diff/layout, logcat, clipboard).

## Documentation

| | |
| --- | --- |
| **[`docs/MATCHERS.md`](docs/MATCHERS.md)** | The full matcher vocabulary — text, content-desc, testTag, state filters, hierarchy combinators, logical combinators. |
| **[`docs/TOOLS.md`](docs/TOOLS.md)** | Every tool, its arguments, and what it maps to in Espresso/Compose. |
| **[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)** | How the server is structured: layered packages, request flow, sync model, failure modes. |
| **[`docs/EXAMPLES.md`](docs/EXAMPLES.md)** | Realistic test flows expressed as MCP tool calls — login, list scrolling, deep-link assertions, visual regression, sibling correctness. |
| **[`examples/sample-skill/`](examples/sample-skill/)** | A complete worked Markdown-runbook skill — preflight, fixtures for preparation and teardown, three sample tests — written in the human-friendly prose style backed by a verb→tool vocabulary. |

## Runtime requirements

| Tool | Required? | Used for |
| --- | --- | --- |
| `adb` | **always** | every interaction with the device |
| `android` ([agent CLI](https://developer.android.com/tools/agents/android-cli)) | recommended | `screen_resolve` (LLM-friendly visual lookup); preferred path for `screen_layout` and `screen_capture` |

If `android` is missing the server logs one warning and the affected tools fall back to UIAutomator + `adb screencap`.

## Install

Both paths land the `velocity-test-mobile` binary in `~/.local/bin`. Make sure that directory is on `PATH`.

### From a release (recommended)

Pick the asset for your Mac and curl it straight into `~/.local/bin`:

```bash
mkdir -p ~/.local/bin

# Apple Silicon (arm64)
curl -L -o ~/.local/bin/velocity-test-mobile \
  https://github.com/randheer094/velocity-test-mobile/releases/latest/download/velocity-test-mobile-macos-arm64

# Intel Mac (x86_64)
curl -L -o ~/.local/bin/velocity-test-mobile \
  https://github.com/randheer094/velocity-test-mobile/releases/latest/download/velocity-test-mobile-macos-x86_64

chmod +x ~/.local/bin/velocity-test-mobile
```

### From source

```bash
git clone https://github.com/randheer094/velocity-test-mobile.git
cd velocity-test-mobile
make install                     # builds, then moves binary to ~/.local/bin
velocity-test-mobile --version
velocity-test-mobile --list-tools | wc -l   # 104
```

## Hooking up to a client

### Claude Desktop

```jsonc
// ~/Library/Application Support/Claude/claude_desktop_config.json
{
  "mcpServers": {
    "velocity-test-mobile": {
      "command": "/absolute/path/to/velocity-test-mobile"
    }
  }
}
```

### Claude Code

Install the Claude Code CLI:

```bash
npm install -g @anthropic-ai/claude-code
```

Then register this server (per-project or globally):

```bash
# project scope (writes .mcp.json in the current directory)
claude mcp add velocity-test-mobile /absolute/path/to/velocity-test-mobile --scope project

# user scope (available across all projects)
claude mcp add velocity-test-mobile /absolute/path/to/velocity-test-mobile --scope user
```

Verify the server is connected:

```bash
claude mcp list
```

### Gemini CLI

Install the Gemini CLI:

```bash
npm install -g @google/gemini-cli
```

Then register this server (project or user scope):

```bash
# project scope (writes .gemini/settings.json in the current directory)
gemini mcp add velocity-test-mobile /absolute/path/to/velocity-test-mobile --scope project

# user scope (writes ~/.gemini/settings.json, available across all projects)
gemini mcp add velocity-test-mobile /absolute/path/to/velocity-test-mobile --scope user
```

Verify the server is connected:

```bash
gemini mcp list
```

### MCP Inspector

```bash
npx @modelcontextprotocol/inspector ./velocity-test-mobile
```

## The matcher in 30 seconds

Every assert / act / wait tool takes a `match` selector with the same JSON shape:

```jsonc
{
  // Identity
  "text": "Login",                  // exact
  "textContains": "ogi",            // substring
  "textRegex": "^Log",              // Go regex
  "contentDescription": "Login button",
  "resourceId": "loginBtn",         // suffix or fully-qualified
  "testTag": "loginBtn",            // Compose; works with testTagsAsResourceId
  "className": "Button",            // substring
  "hint": "Username",
  "errorText": "Required",

  // State filters (any subset; all are tri-state)
  "clickable": true, "enabled": true, "checked": false,
  "displayed": true, "completelyDisplayed": true,
  "displayingAtLeastPercent": 75,
  "on": true, "off": false, "toggleable": true,

  // Tree position
  "isRoot": false, "childCount": 3, "minChildCount": 1, "parentIndex": 0,

  // Hierarchy combinators (each is a nested matcher)
  "hasAncestor":   { "scrollable": true },
  "hasDescendant": { "text": "Item 1" },
  "hasParent":     { "className": "Container" },
  "hasSibling":    { "checked": true },

  // Logical combinators
  "allOf": [{ "className": "Button" }, { "enabled": true }],
  "anyOf": [{ "text": "OK" }, { "text": "Continue" }],
  "not":   { "className": "Disabled" },

  // Disambiguation
  "instance": 0
}
```

See [`docs/MATCHERS.md`](docs/MATCHERS.md) for the field-by-field reference and [`docs/EXAMPLES.md`](docs/EXAMPLES.md) for realistic flows.

## Tools at a glance (104)

| Group | Examples |
| --- | --- |
| **Find** | `find_node`, `find_all_nodes`, `count_nodes`, `print_tree` |
| **Assert — visibility** | `assert_visible`, `assert_not_visible`, `assert_completely_displayed`, `assert_displaying_at_least`, `assert_exists`, `assert_does_not_exist` |
| **Assert — state** | `assert_clickable`, `assert_enabled`/`assert_disabled`, `assert_focused`, `assert_selected`, `assert_checked`/`assert_unchecked`, `assert_on`/`assert_off`, `assert_toggleable` |
| **Assert — text/CD** | `assert_text_equals`, `assert_text_contains`, `assert_content_description_equals` |
| **Assert — geometry** | `assert_width_dp`, `assert_height_dp`, `assert_width_at_least_dp`, `assert_height_at_least_dp`, `assert_position_in_root` |
| **Assert — tree shape** | `assert_is_root`, `assert_has_child_count`, `assert_has_minimum_child_count`, `assert_has_descendant` |
| **Assert — collections** | `assert_count_equals`, `assert_any`, `assert_all` |
| **Act** | `click`, `double_click`, `long_click`, `type_text`, `replace_text`, `clear_text`, `submit_text`, `swipe_node`, `slow_swipe_node`, `scroll_to`, `scroll_to_index`, `perform_ime_action`, `perform_key_press`, `assert_clickable_and_click` |
| **Sync** | `wait_until_visible`, `wait_until_not_visible`, `wait_until_text`, `wait_until_count`, `wait_until_at_least_one_exists`, `wait_for_idle` |
| **Espresso top-level** | `espresso_press_back`, `press_back_unconditionally`, `close_soft_keyboard`, `open_overflow_menu`, `open_contextual_action_mode_menu` |
| **Intents (recording-only)** | `intent_monitor_start`, `intent_monitor_stop`, `intent_list_captured`, `assert_intent_sent`, `assert_intent_count` |
| **Test fixtures** | `device_list`, `device_get_screen_size`, `device_get_props`, `device_get_orientation`, `device_set_orientation`, `animations_set`, `animations_get`, `app_list`, `app_launch`, `app_terminate`, `app_clear_data`, `app_get_info`, `permission_grant`, `permission_revoke`, `appops_set`, `appops_get`, `intent_send`, `app_data_list`, `app_data_read`, `screen_capture`, `screen_layout`, `screen_resolve`, `screen_diff`, `clipboard_get`, `clipboard_set`, `press_key`, `type_into_focused`, `logcat_tail`, `logcat_clear` |
| **Activity / service / location / shell** | `activity_get_top`, `activity_wait_for_top`, `activity_start`, `service_get_state`, `service_wait_for_state`, `location_get_last_known`, `notification_list`, `notification_shade_set`, `notification_tap`, `shell_exec` |

Full reference in [`docs/TOOLS.md`](docs/TOOLS.md).

## Security

The `shell_exec` tool forwards arbitrary commands verbatim to `adb shell` on the connected device. Anyone who can reach this MCP server (locally, over a forwarded socket, or via a misconfigured client) can execute shell commands on every device that's currently `adb`-attached. Treat the server like an open `adb shell` and only run it on hosts and connections you trust.

## Caveats vs. the real frameworks (in-process bits we can't externally implement)

| Capability | Behaviour here |
| --- | --- |
| Compose `testTag` | Matches when the app sets `Modifier.semantics { testTagsAsResourceId = true }`; otherwise fall back to `contentDescription` / `text`. |
| `IdlingResource` / `onIdle()` | Approximated by `wait_for_idle` — polls the tree and returns when two snapshots hash identically over an idle window. |
| `mainClock.advanceTimeBy(...)` | Not supported (in-process only). |
| Espresso `intending().respondWith(...)` | Not supported. `assert_intent_sent` / `assert_intent_count` *read* dispatched intents from logcat. |
| `assertWidthIsEqualTo(dp)` | Supported via `assert_width_dp` with an explicit `density` argument; pixel-to-dp conversion happens server-side. |
| `performKeyPress(key, meta)` | Modifier flags (Ctrl/Shift/Alt) dispatch via `input keycombination` on Android 12+; on older devices the key alone is sent and the result reports the missing coverage. |
| `performScrollToIndex(idx)` | LazyColumn/Row indexing is opaque externally; we dispatch `idx` page-sized swipes inside the matched scrollable container (direction configurable). |

## A worked example

```jsonc
{ "tool": "animations_set",            "args": { "scale": 0 } }
{ "tool": "app_clear_data",            "args": { "package": "com.example.app" } }
{ "tool": "app_launch",                "args": { "package": "com.example.app" } }
{ "tool": "wait_for_idle",             "args": { "idleWindowMs": 800 } }

{ "tool": "type_text",                 "args": { "match": { "testTag": "username" }, "text": "alice" } }
{ "tool": "type_text",                 "args": { "match": { "testTag": "password" }, "text": "s3cret" } }
{ "tool": "assert_clickable_and_click","args": { "match": { "text": "Sign in" } } }

{ "tool": "wait_until_visible",        "args": { "match": { "text": "Welcome, alice" }, "timeoutMs": 10000 } }
{ "tool": "assert_count_equals",       "args": { "match": { "resourceId": "order_row" }, "expected": 5 } }
{ "tool": "assert_all",                "args": { "match": { "resourceId": "order_row" }, "sub": { "clickable": true } } }
```

More in [`docs/EXAMPLES.md`](docs/EXAMPLES.md).

## Development

```bash
make vet         # go vet ./...
make test        # go test ./...
make build       # static binary
make lint        # vet + gofmt
make list-tools  # canonical tool catalogue
```

The codebase is `internal/` packages plus a thin `main.go`:

- `internal/runner` — single subprocess gateway (timeout, byte cap, structured errors)
- `internal/adb` / `internal/androidcli` — typed CLI wrappers
- `internal/ui` — layout (UIAutomator + android-CLI JSON), screenshot, pixel diff
- `internal/matcher` — JSON-friendly recursive selector + `Find` / `FindAll`
- `internal/testing` — assertions, actions, sync, intent recorder
- `internal/tools` — MCP tool registrations (matcher-bearing tools use `Server.AddTool` with hand-built JSON Schema for recursion via `$ref`)

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the full design.
