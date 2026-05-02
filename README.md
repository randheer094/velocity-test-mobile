# velocity-mcp-mobile

An **Android testing** [Model Context Protocol](https://modelcontextprotocol.io) server, written in Go.

The server exposes **Espresso** ViewMatchers/ViewActions/ViewAssertions and **Jetpack Compose** test verbs (`onNodeWithText`, `assertIsDisplayed`, `performClick`, `waitUntilExists`, …) as MCP tools an LLM agent can call directly. Each tool takes a shared **matcher** object — the same vocabulary across find / assert / act / wait — so an agent's prompt of *"verify Login is visible; click it; wait for Welcome"* is three self-contained tool calls.

Internally the server walks the device's accessibility tree (UIAutomator XML or Google's [`android` agent CLI](https://developer.android.com/tools/agents/android-cli) JSON), applies the matcher, and dispatches `adb shell input` actions. **No in-process instrumentation** — no companion APK, no Espresso runtime on the device.

- **Single static Go binary**, ~7 MB, sub-10ms cold start.
- **No telemetry**, no phone-home — fully local.
- **Testing-only**: lifecycle/deployment/diagnostics tooling intentionally absent. Just the Espresso/Compose surface plus the minimum supporting infrastructure tests need (animations off, app launch/clear, permissions, intents, screenshot/diff/layout, logcat).

## Runtime requirements

| Tool | Required? | Used for |
| --- | --- | --- |
| `adb` | **always** | every interaction with the device |
| `android` ([agent CLI](https://developer.android.com/tools/agents/android-cli)) | recommended | `screen_resolve` (LLM-friendly visual lookup); preferred path for `screen_layout` and `screen_capture` |

If `android` is missing the server logs one warning and the affected tools fall back to UIAutomator/`adb screencap`.

## Install

```bash
git clone https://github.com/randheer094/velocity-mcp-mobile.git
cd velocity-mcp-mobile
make build
./velocity-mcp-mobile --version
./velocity-mcp-mobile --list-tools | wc -l
```

## Hooking up to a client

```jsonc
// ~/Library/Application Support/Claude/claude_desktop_config.json
{
  "mcpServers": {
    "android-test": {
      "command": "/absolute/path/to/velocity-mcp-mobile"
    }
  }
}
```

Or interactively:

```bash
npx @modelcontextprotocol/inspector ./velocity-mcp-mobile
```

## The matcher

Every assert / act / wait tool takes a `match` selector with the same JSON shape:

```jsonc
{
  // Identity (any combination)
  "text": "Login",                       // exact
  "textContains": "ogi",                 // substring
  "textRegex": "^Log",                   // Go regex
  "contentDescription": "Login button",
  "contentDescriptionContains": "...",
  "resourceId": "loginBtn",              // accepts suffix or fully-qualified id
  "testTag": "loginBtn",                 // Compose testTag (works with testTagsAsResourceId)
  "className": "Button",                 // substring
  "hint": "Username",
  "package": "com.example.app",
  "errorText": "Invalid email",

  // State filters (any subset)
  "clickable": true, "longClickable": false,
  "enabled": true, "checkable": false, "checked": false,
  "focused": true, "focusable": true,
  "selected": false, "scrollable": false,
  "displayed": true,                     // bounds + visibleToUser
  "completelyDisplayed": true,           // Espresso isCompletelyDisplayed
  "displayingAtLeastPercent": 75,        // Espresso isDisplayingAtLeast
  "on": true, "off": false,              // Compose isOn / isOff aliases
  "toggleable": true,                    // Compose isToggleable

  // Tree position
  "isRoot": false,
  "childCount": 3, "minChildCount": 1,
  "parentIndex": 0,                      // Espresso withParentIndex

  // Input semantics
  "hasImeAction": true,
  "inputType": "Password",

  // Hierarchy combinators (each is a nested matcher)
  "hasAncestor":   { "scrollable": true },
  "hasDescendant": { "text": "Item 1" },
  "hasParent":     { "className": "Container" },
  "hasSibling":    { "checked": true },

  // Logical combinators
  "allOf": [{ "className": "Button" }, { "enabled": true }],
  "anyOf": [{ "text": "OK" }, { "text": "Continue" }],
  "not":   { "className": "Disabled" },

  // Disambiguate when multiple match
  "instance": 0
}
```

Recursion is exposed via JSON Schema `$ref`, so MCP clients with schema-aware tooling get full discovery.

## Tools (92 total)

### Espresso ViewMatchers / Compose finders
`find_node`, `find_all_nodes`, `count_nodes`, `print_tree`

### Espresso ViewAssertions / Compose assertions
- **Existence/visibility**: `assert_visible`, `assert_not_visible`, `assert_completely_displayed`, `assert_displaying_at_least`, `assert_exists`, `assert_does_not_exist`
- **State**: `assert_clickable`, `assert_enabled`, `assert_disabled`, `assert_focused`, `assert_selected`, `assert_checked`, `assert_unchecked`, `assert_on`, `assert_off`, `assert_toggleable`
- **Text/CD**: `assert_text_equals`, `assert_text_contains`, `assert_content_description_equals`
- **Geometry**: `assert_width_dp`, `assert_height_dp`, `assert_width_at_least_dp`, `assert_height_at_least_dp`, `assert_position_in_root`
- **Tree shape**: `assert_is_root`, `assert_has_child_count`, `assert_has_minimum_child_count`, `assert_has_descendant`
- **Collections**: `assert_count_equals`, `assert_any`, `assert_all`

### Espresso ViewActions / Compose actions
`click`, `double_click`, `long_click`, `type_text`, `replace_text`, `clear_text`, `submit_text`, `swipe_node`, `slow_swipe_node`, `scroll_to`, `scroll_to_index`, `perform_ime_action`, `perform_key_press`, `assert_clickable_and_click`

### Synchronization (poll-based — no IdlingResource hook from outside)
`wait_until_visible`, `wait_until_not_visible`, `wait_until_text`, `wait_until_count`, `wait_until_at_least_one_exists`, `wait_for_idle`

### Espresso top-level
`espresso_press_back`, `press_back_unconditionally`, `close_soft_keyboard`, `open_overflow_menu`, `open_contextual_action_mode_menu`

### Espresso-Intents (recording-only — no stubbing)
`intent_monitor_start`, `intent_monitor_stop`, `intent_list_captured`, `assert_intent_sent`, `assert_intent_count`

### Test setup / teardown / verification (around tests)
- **Device & fixtures**: `device_list`, `device_get_screen_size`, `device_get_props`, `device_get_orientation`, `device_set_orientation`, `animations_set`, `animations_get`
- **App lifecycle/state**: `app_list`, `app_launch`, `app_terminate`, `app_clear_data`, `app_get_info`, `permission_grant`, `permission_revoke`, `intent_send`, `app_data_list`, `app_data_read`
- **Screen / visual**: `screen_capture`, `screen_layout`, `screen_resolve`, `screen_diff`
- **Input utilities**: `clipboard_get`, `clipboard_set`, `press_key`, `type_into_focused`
- **Logs**: `logcat_tail`, `logcat_clear`

### Caveats vs. the real frameworks (in-process bits we can't externally implement)

| Capability | Behaviour here |
| --- | --- |
| Compose `testTag` | Matches when the app sets `Modifier.semantics { testTagsAsResourceId = true }`; otherwise fall back to `contentDescription` / `text`. |
| `IdlingResource` / `onIdle()` | Approximated by `wait_for_idle` — polls the tree and returns when two snapshots hash identically over an idle window. |
| `mainClock.advanceTimeBy(...)` | Not supported (in-process only). |
| Espresso `intending().respondWith(...)` | Not supported. `assert_intent_sent` / `assert_intent_count` *read* dispatched intents from logcat. |
| `assertWidthIsEqualTo(dp)` | Supported via `assert_width_dp` with an explicit `density` argument; pixel-to-dp conversion happens server-side. |
| `performKeyPress(key, meta)` | Modifier flags (Ctrl/Shift/Alt) are accepted but only honoured on devices whose `input keycombination` supports them. |
| `performScrollToIndex(idx)` | LazyColumn/Row indexing is opaque externally; we dispatch `idx` page-sized swipes inside the matched scrollable container. |

## Example: the user's pattern

```jsonc
{ "tool": "assert_visible",             "args": { "match": { "text": "Welcome" } } }
{ "tool": "assert_clickable_and_click", "args": { "match": { "text": "Continue" } } }
{ "tool": "wait_until_visible",         "args": { "match": { "testTag": "homeScreen" }, "timeoutMs": 5000 } }
```

## Development

```bash
make vet       # go vet ./...
make test      # go test ./...
make build     # static binary
make lint      # vet + gofmt
make list-tools
```

The codebase is `internal/` packages plus a thin `main.go`:

- `internal/runner` — single subprocess gateway (timeout, byte cap, structured errors)
- `internal/adb` / `internal/androidcli` — thin wrappers
- `internal/ui` — layout (UIAutomator + android-CLI JSON), screenshot, pixel diff
- `internal/matcher` — JSON-friendly recursive selector + `Find` / `FindAll`
- `internal/testing` — assertions, actions, sync, intent recorder
- `internal/tools` — MCP tool registrations (matcher-bearing tools use `Server.AddTool` with hand-built JSON Schema for recursion via `$ref`)
