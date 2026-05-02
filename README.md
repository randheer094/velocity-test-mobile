# velocity-mcp-mobile

An **Android-only** [Model Context Protocol](https://modelcontextprotocol.io) server, written in Go, that gives an LLM agent everything it needs to drive, observe, and **verify** an Android app on a device or emulator.

The agent calls **Espresso- and Compose-test-style verbs** like `assert_visible({text:"Login"})` and `click({testTag:"submitBtn"})`. Internally the server walks the device's accessibility tree and dispatches `adb shell input` actions — no in-process instrumentation required.

Inspired by [`mobile-next/mobile-mcp`](https://github.com/mobile-next/mobile-mcp). Differences:

- **Android-only** — no iOS / WebDriverAgent.
- **No telemetry**, no phone-home — fully local.
- **Single static Go binary**, sub-10ms cold start, no Node/Python runtime.
- Prefers Google's new agent CLI ([`android`](https://developer.android.com/tools/agents/android-cli)) where it offers something cleaner; falls back to `adb` everywhere else.
- **First-class testing surface** — Espresso ViewMatchers + ViewActions + ViewAssertions and Compose finders/actions/assertions exposed as MCP tools, including `wait_until_visible`, `wait_for_idle`, `scroll_to`, and recording-only `assert_intent_sent`.
- Wider verification surface — pixel diffs, perfetto/atrace, animations control, doze simulation, network/airplane/Wi-Fi/data toggles, time/timezone, mock GPS, app-data inspection (run-as), wireless ADB, and more.

## Runtime requirements

| Tool | Required? | Used for |
| --- | --- | --- |
| `adb` | **always** | every interaction with the device |
| `android` ([agent CLI](https://developer.android.com/tools/agents/android-cli)) | recommended | `screen_resolve`, `screen_layout` (preferred path), `screen_capture` (preferred path), `app_install` (preferred path), `emulator_*`, `docs_*` |

Both must be on `PATH`. If `android` is missing, the server logs a single warning and the affected tools return an actionable error pointing at the install page; everything else still works through plain `adb`.

## Install

```bash
git clone https://github.com/randheer094/velocity-mcp-mobile.git
cd velocity-mcp-mobile
go install .            # or: make build
```

Or build from a checkout:

```bash
make build
./velocity-mcp-mobile --version
./velocity-mcp-mobile --list-tools | wc -l
```

## Hooking up to a client

### Claude Desktop / Claude Code

```jsonc
// ~/Library/Application Support/Claude/claude_desktop_config.json (macOS)
{
  "mcpServers": {
    "android": {
      "command": "/absolute/path/to/velocity-mcp-mobile"
    }
  }
}
```

### MCP Inspector (interactive)

```bash
npx @modelcontextprotocol/inspector ./velocity-mcp-mobile
```

## Tool surface

The server exposes **~107 tools** in two layers — a low-level device-driving layer and a high-level testing layer. Use `--list-tools` to print the canonical names.

### Testing layer (Espresso / Compose-style)

Every testing tool takes a shared **matcher** object — the same vocabulary across find / assert / act / wait. Recursive combinators (`hasAncestor`, `hasDescendant`, `not`, `allOf`, `anyOf`) are exposed via JSON Schema `$ref`.

```jsonc
// Matcher fields (any combination)
{
  "text": "Login",                 // exact
  "textContains": "ogi",           // substring
  "textRegex": "^Log",             // Go regex
  "contentDescription": "...",
  "resourceId": "loginBtn",        // accepts suffix or fully-qualified id
  "testTag": "loginBtn",           // Compose testTag (works with testTagsAsResourceId)
  "className": "Button",           // substring
  "clickable": true, "enabled": true, "checked": false,  // state filters (any subset)
  "displayed": true,
  "hasAncestor": { "scrollable": true },
  "hasDescendant": { "text": "Item 1" },
  "not": { "className": "Disabled" },
  "instance": 0                    // pick the Nth match
}
```

| Group | Tools |
| --- | --- |
| **Find** | `find_node`, `find_all_nodes`, `count_nodes`, `print_tree` |
| **Assert** (read-only) | `assert_visible`, `assert_not_visible`, `assert_exists`, `assert_does_not_exist`, `assert_clickable`, `assert_enabled`, `assert_disabled`, `assert_focused`, `assert_selected`, `assert_checked`, `assert_unchecked`, `assert_text_equals`, `assert_text_contains`, `assert_content_description_equals`, `assert_count_equals`, `assert_has_descendant` |
| **Act** | `click`, `double_click`, `long_click`, `type_text`, `replace_text`, `clear_text`, `submit_text`, `swipe_node`, `scroll_to`, `perform_ime_action`, `assert_clickable_and_click` |
| **Sync** | `wait_until_visible`, `wait_until_not_visible`, `wait_until_text`, `wait_until_count`, `wait_for_idle` (poll-and-hash heuristic) |
| **Espresso conveniences** | `espresso_press_back`, `close_soft_keyboard`, `open_overflow_menu` |
| **Intents (recording-only)** | `intent_monitor_start`, `intent_list_captured`, `assert_intent_sent` — scrapes ActivityManager logcat. Stubbing (`intending().respondWith()`) is **not** possible without instrumentation. |

The user's pattern *"verify X is visible; verify Y is clickable and click Y"* maps directly:

```jsonc
{ "tool": "assert_visible",            "args": { "match": { "text": "Welcome" } } }
{ "tool": "assert_clickable_and_click","args": { "match": { "text": "Continue" } } }
```

#### Caveats vs. the real frameworks

| Capability | Behaviour here |
| --- | --- |
| Compose `testTag` | Works when the app sets `Modifier.semantics { testTagsAsResourceId = true }` (then matched via `resourceId`/`testTag`); otherwise fall back to `contentDescription` / `text`. |
| `IdlingResource` / `onIdle()` | Approximated by `wait_for_idle` — polls the tree and returns when two snapshots hash identically over an idle window. No in-process hook. |
| `mainClock.advanceTimeBy(...)` | Not supported (in-process only). |
| Espresso `intending()` stubbing | Not supported. `assert_intent_sent` reads dispatched intents from logcat. |
| `assertWidthIsEqualTo(dp)` | Not exposed — bounds are returned in pixels; convert via `device_get_screen_size` density if needed. |

### Lower-level device-driving layer

Highlights of the underlying surface (the testing layer is layered on top of these):

| Category | Tools |
| --- | --- |
| Device | `device_list`, `device_get_screen_size`, `device_get_props`, `device_get_orientation`, `device_set_orientation` |
| Emulator | `emulator_list`, `emulator_start`, `emulator_stop` |
| Apps | `app_list`, `app_install`, `app_uninstall`, `app_launch`, `app_terminate`, `app_clear_data`, `app_get_info`, `permission_grant`, `permission_revoke`, `intent_send`, `app_data_list`, `app_data_read` |
| UI capture | `screen_capture`, `screen_layout`, `screen_resolve` |
| UI input | `tap`, `double_tap`, `long_press`, `swipe`, `drag`, `fling`, `type_keys`, `press_button` |
| Clipboard | `clipboard_get`, `clipboard_set` |
| Test asserts | `wait_for_element`, `assert_text_visible`, `screen_diff` |
| Diagnostics | `logcat_tail`, `logcat_clear`, `dumpsys_meminfo`, `dumpsys_gfxinfo`, `dumpsys_battery`, `dumpsys_activity` |
| Tracing | `atrace_capture`, `perfetto_capture` |
| Recording | `screen_record_start`, `screen_record_stop` |
| Files | `file_push`, `file_pull` |
| System state | `screen_wake`, `screen_lock`, `animations_set`, `animations_get`, `doze_simulate`, `time_set_timezone`, `network_set_airplane`, `network_set_wifi`, `network_set_mobile_data`, `location_set` |
| Maintenance | `device_reboot`, `wireless_enable`, `wireless_connect`, `wireless_pair`, `wireless_disconnect` |
| Knowledge | `docs_search`, `docs_fetch` |

### Designed for stable UI tests

`animations_set 0` plus `wait_for_element` is the recommended way to drive flake-free UI flows. Pair `screen_capture` with `screen_diff` for visual regression. `screen_layout` returns a flat list of interactive nodes with bounds so an agent can click by intent rather than blind coordinates; `screen_resolve` (Android CLI) is even more direct for label-based clicks.

## Notes & caveats

- `app_data_list` / `app_data_read` use `run-as` and therefore only work on **debuggable** builds. Release builds will return an actionable error.
- `network_set_wifi` / `network_set_mobile_data` rely on the `svc` helper; on some OEM builds `svc` requires root.
- `location_set` uses `adb emu geo fix` on emulators; on physical devices it cannot truly mock a location without a developer-options mock-location-provider app, so it returns a `device` mode result describing what's required.
- `device_reboot` is destructive — it requires `confirm: true`.
- `screenrecord` runs on-device and the file is pulled to host on stop. Long sessions are bounded by Android's own `screenrecord` ceiling (typically 180s) per chunk.
- `perfetto` requires API 28+ and a working `tracebox` on most devices.

## Development

```bash
make vet      # go vet ./...
make test     # go test ./...
make build    # static binary
make lint     # vet + gofmt
```

The entire codebase is `internal/` packages plus a thin `main.go` and a `tools/` directory of MCP handlers, one file per surface. Every subprocess goes through `internal/runner` so timeouts, output caps, and error wrapping are uniform.

## License

Source provided as-is for the user's project; choose a license at the repository level.
