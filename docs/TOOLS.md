# Tool reference

129 tools, grouped by purpose. Every device-targeted tool accepts an optional `device` argument (omit when only one device is connected). Every matcher-bearing tool takes a `match` object — see [MATCHERS.md](MATCHERS.md) for the field reference.

## Conventions

- **`AssertResult`** — `{ ok: bool, reason?: string, element?: <node>, matched: int }`
- **`ActionResult`** — `{ ok: bool, element?: <node>, x: int, y: int, reason?: string }`
- **`WaitResult`**   — `{ ok: bool, attempts: int, waitedMs: int64, element?: <node>, matchedNow: int, reason?: string }`
- Where a value is missing (no element matched, etc.), `reason` carries an actionable string.

---

## Device & fixtures

| Tool | Read-only | Args | Behaviour |
| --- | --- | --- | --- |
| `device_list` | yes | — | Returns connected devices via `adb devices -l`. |
| `device_get_screen_size` | yes | — | Parsed `wm size` + `wm density`: `{ width, height, density }`. Density is critical for `assert_width_dp`/`assert_height_dp`. |
| `device_get_orientation` | yes | — | Returns `"portrait"` or `"landscape"` (from `settings get system user_rotation`). |
| `device_set_orientation` | no | `orientation: "portrait"|"landscape"` | Locks orientation via `settings put`. |
| `device_get_props` | yes | — | Curated subset of `getprop`: serial, model, brand, manufacturer, SDK level, release, fingerprint, ABI list. |
| `animations_set` | no | `scale: number` | Writes `window_animation_scale`, `transition_animation_scale`, `animator_duration_scale`. **Pass 0 before any UI test** to eliminate animation-driven flakiness. |
| `animations_get` | yes | — | Returns the three animation scales. |

## App lifecycle / state / verification

| Tool | Read-only | Args | Behaviour |
| --- | --- | --- | --- |
| `app_list` | yes | — | Apps with launcher activities, parsed from `cmd package query-activities`. |
| `app_launch` | no | `package`, `locale?` | `monkey` to launch; optional `cmd locale set-app-locales`. |
| `app_terminate` | no | `package`, `kind?` (`force_stop` default / `kill`) | Two modes. `force_stop` → `am force-stop`: hard stop, marks package STOPPED, cancels alarms/jobs, drops queued broadcasts, blocks service auto-restart until next launch — use for clean test teardown. `kill` → `am kill`: soft kill of cached/background process only (no-op in foreground), leaves package state intact so `START_STICKY` services restart, alarms still fire, broadcasts still deliver — use to simulate OS memory reclamation (e.g. `START_STICKY` recovery, JobService re-run). |
| `app_clear_data` | no (destructive) | `package` | `pm clear`. |
| `app_get_info` | yes | `package` | Parsed `dumpsys package`: `versionName`, `versionCode`, `targetSdk`, `minSdk`, `firstInstallTime`, `lastUpdateTime`, requested vs granted permissions, signers. |
| `permission_grant` | no | `package`, `permission` | `pm grant`. |
| `permission_revoke` | no | `package`, `permission` | `pm revoke`. |
| `appops_set` | no | `package`, `op`, `mode` (`allow`/`deny`/`ignore`/`default`) | `appops set`. Distinct from runtime permissions — covers system-level grants like `android:mock_location` that `permission_grant` can't reach. |
| `appops_get` | yes | `package`, `op` | `appops get`. Returns `default` when the op is not explicitly set. |
| `intent_send` | no | `mode?` (`start`/`broadcast`/`service`/`foreground_service`), `action?`, `category?`, `data?` (URI), `mime?`, `package?`, `class?` (`pkg/.Class`), `flags?: []`, `stringExtras?`, `intExtras?`, `boolExtras?`, `floatExtras?` | `am start`/`broadcast`/`start-service`/`start-foreground-service`. Inputs are validated against safe regexes. |
| `app_data_list` | yes | `package`, `relativePath?` | `run-as <pkg> ls -la`. Requires a debuggable build. |
| `app_data_read` | yes | `package`, `relativePath` | `run-as <pkg> cat`. Path validated against `..` traversal. |

## Activity, service, location, notification & shell introspection

These wrap the `dumpsys` services and `cmd statusbar` so runbooks can stay
inside the MCP contract instead of falling back to `Bash` for `adb shell`.

| Tool | Read-only | Args | Behaviour |
| --- | --- | --- | --- |
| `activity_get_top` | yes | — | Returns `{ bundle_id, activity, task_id? }` from `dumpsys activity activities` (`topResumedActivity`). Returns null when no activity is resumed. |
| `activity_wait_for_top` | yes | `bundle_id`, `activity?`, `timeout_ms?` | Poll until the resumed activity matches; `activity` accepts FQN or relative form (`.MainActivity`). |
| `activity_start` | no | `bundle_id`, `activity`, `action?`, `data?`, `flags?`, `stringExtras?`, `intExtras?`, `boolExtras?`, `floatExtras?` | `am start -n <pkg>/<activity>`. Bypasses launcher resolution — use when `monkey -c LAUNCHER` picks the wrong activity (e.g. LeakCanary). |
| `service_get_state` | yes | `bundle_id`, `component?` | Parsed `dumpsys activity services <pkg>`: `{ running, foreground, component, notification_id?, start_id?, last_intent_action? }`. |
| `service_wait_for_state` | yes | `bundle_id`, `component?`, `running?`, `foreground?`, `timeout_ms?` | Poll variant of `service_get_state` — wait until every set field matches. |
| `location_get_last_known` | yes | `provider?` (`gps`/`network`/`passive`/`fused`) | Parses `last location=Location[<provider> <lat>,<lng> ...]` from `dumpsys location`. Returns null when no fix is reported. |
| `notification_list` | yes | `bundle_id?`, `channel_id?` | Parsed `dumpsys notification --noredact`: `[{ bundle_id, channel_id, title, text, ongoing, posted_time_ms, tag, id }]`. |
| `notification_shade_set` | no | `state` (`expanded`/`collapsed`/`quick_settings`) | `cmd statusbar expand-notifications` / `collapse` / `expand-settings`. |
| `notification_tap` | no | `bundle_id`, `channel_id?`, `title_match?` | Higher-level convenience: opens the shade, finds a matching notification by title/text via the testing matcher, clicks it, collapses the shade. |
| `shell_exec` | no | `command`, `timeout_ms?` (default 10000) | Documented backstop for arbitrary `adb shell <command>`. Returns `{ stdout, stderr, exit_code }` regardless of exit status. Prefer the typed wrappers above; reach for this only for one-off introspection (new dumpsys services, debug-only setprop). |

## Screen capture & visual regression

| Tool | Read-only | Args | Behaviour |
| --- | --- | --- | --- |
| `screen_capture` | yes | `displayId?`, `saveTo?` | Returns PNG inline (`mcp.ImageContent`). If `saveTo` is set, also writes the PNG/JPG to disk. |
| `screen_layout` | yes | — | Flat list of interactive elements with bounds. (Use `find_*` for targeted lookups.) |
| `screen_resolve` | yes | `label` | LLM-friendly visual label-to-coordinates lookup via `android screen resolve`. Requires the agent CLI. |
| `screen_diff` | yes | `pathA`, `pathB`, `diffOutput?`, `tolerance?`, `thresholdPct?` | Per-pixel comparison; optionally writes a red-overlay diff image. Reports `mismatchedPixels`, `mismatchPct`, `exceedsTolerance`. |

## Device-state simulation

Test-time environmental knobs. None of these survive a device reboot — call them at the start of each session and (for `battery_set_state`) reset before exit.

| Tool | Read-only | Args | Behaviour |
| --- | --- | --- | --- |
| `device_set_font_scale` | no | `scale: number` (0.5..2.5) | `settings put system font_scale <v>`. Drives accessibility / large-text regression tests. |
| `device_set_dark_mode` | no | `mode: "yes"|"no"|"auto"` | `cmd uimode night yes|no|auto`. |
| `airplane_mode_set` | no | `on: bool` | `cmd connectivity airplane-mode enable|disable`. |
| `battery_set_state` | no | `reset?: bool`, `level?: 0..100`, `status?: 1..5`, `ac?: 1|2`, `usb?: 1|2`, `wireless?: 1|2` | `dumpsys battery set ...`. Pair test-time overrides with `{reset: true}` in cleanup; otherwise the device reports fake state until reboot. Status codes: 1 unknown · 2 charging · 3 discharging · 4 not_charging · 5 full. AC/USB/wireless: 1 unplugged, 2 plugged. |
| `network_set` | no | `wifi?: bool`, `mobile?: bool` | `svc wifi enable|disable`, `svc data enable|disable`. Set only the radios you want to change. |
| `app_set_locale` | no | `package`, `tag?` | `cmd locale set-app-locales`. Empty `tag` clears the override. Distinct from `app_launch.locale` (per-launch only). |

## Input utilities

(High-level Espresso/Compose verbs are in the testing section below.)

| Tool | Read-only | Args | Behaviour |
| --- | --- | --- | --- |
| `clipboard_get` | yes | — | Read primary clipboard (Android 10+). |
| `clipboard_set` | no | `text` | Write primary clipboard. Unicode-safe via base64. |
| `press_key` | no | `key` | `input keyevent <code>` for any key in [the keycode map](#keycode-map). |
| `type_into_focused` | no | `text`, `submit?` | Type into whichever view currently has focus (Espresso `typeTextIntoFocusedView`). |
| `tap_at_coordinates` | no | `x`, `y` | Raw `input tap`. Fallback for Canvas / WebView surfaces a matcher cannot resolve. |
| `long_press_at_coordinates` | no | `x`, `y`, `durationMs?` (800) | Raw long-press. |
| `swipe_screen` | no | `fromX`, `fromY`, `toX`, `toY`, `durationMs?` (200) | Free-form swipe between two coordinates — edge swipes, pull-to-refresh. |
| `drag_screen` | no | `fromX`, `fromY`, `toX`, `toY`, `durationMs?` (800) | Same dispatch as `swipe_screen` but with a longer default so apps that distinguish drag from swipe register the gesture. |

## Logs

| Tool | Read-only | Args | Behaviour |
| --- | --- | --- | --- |
| `logcat_tail` | yes | `package?`, `tag?`, `priority?`, `maxLines?`, `since?`, `regex?` | `logcat -d` with the given filterspec; `package` is resolved to a PID via `pidof`. `regex` is post-filtered in Go. |
| `logcat_clear` | no | — | `logcat -c`. |
| `screen_record_start` | no | `local_file`, `max_duration_s?`, `size_width?`, `size_height?`, `bit_rate?` | Begins `adb shell screenrecord` in the background, writing to `/sdcard/`. One active recording per device. Returns `{ remote_file }` — save this path; if the server crashes before `screen_record_stop`, call `pull_file` + `shell_exec rm -f <remote_file>` to recover. The device-side process is bounded by `--time-limit` (max 180 s on older Android, 1800 s on Android 11+). |
| `screen_record_stop` | no | — | Cancels the active recording, pulls the MP4 to the path passed to `screen_record_start`, and removes the device-side temp. Returns `{ local_file, remote_file, duration_ms }`. |
| `pull_file` | no | `remote`, `local` | `adb pull` for arbitrary paths (`/sdcard/`, `/data/local/tmp/`). For debuggable-app private storage, prefer `app_data_read`. |

---

## Espresso ViewMatchers / Compose finders (debug)

| Tool | Args | Returns |
| --- | --- | --- |
| `find_node` | `match` | `{ found, element? }` (the `instance`-th match, default 0) |
| `find_all_nodes` | `match` | `{ count, elements: [<node>, ...] }` |
| `count_nodes` | `match` | `{ count }` |
| `print_tree` | `match?`, `maxDepth?` | Indented hierarchy. With `match`, prints the matched subtree; otherwise the whole tree. |

## Espresso ViewAssertions / Compose assertions

All assertion tools are read-only and return an `AssertResult`. They re-snapshot the tree on every call.

### Existence & visibility

| Tool | Maps to |
| --- | --- |
| `assert_visible` | `isDisplayed()` / Compose `assertIsDisplayed()` |
| `assert_not_visible` | Compose `assertIsNotDisplayed()` |
| `assert_completely_displayed` | `isCompletelyDisplayed()` |
| `assert_displaying_at_least` | `isDisplayingAtLeast(percent)` — extra arg `percent: 1..100` |
| `assert_exists` | Compose `assertExists()` |
| `assert_does_not_exist` | Compose `assertDoesNotExist()` / Espresso `doesNotExist()` |

### State

| Tool | Maps to |
| --- | --- |
| `assert_clickable` | `isClickable()` / Compose `assertHasClickAction()` |
| `assert_long_clickable` | `isLongClickable()` |
| `assert_enabled` / `assert_disabled` | `isEnabled()` / `isNotEnabled()` |
| `assert_focused` | `hasFocus()` |
| `assert_selected` | `isSelected()` |
| `assert_checked` / `assert_unchecked` | `isChecked()` / `isNotChecked()` |
| `assert_on` / `assert_off` | Compose `assertIsOn()` / `assertIsOff()` |
| `assert_toggleable` | Compose `assertIsToggleable()` |
| `assert_has_ime_action` | Espresso `hasImeAction()` — best effort: focusable + editable class. |

### Text & content description

| Tool | Args | Maps to |
| --- | --- | --- |
| `assert_text_equals` | `expected: string` | Compose `assertTextEquals()` |
| `assert_text_contains` | `substring: string` | Compose `assertTextContains()` |
| `assert_text_regex` | `pattern: string` | Espresso `withText(matchesPattern())` — Go regex (RE2). |
| `assert_content_description_equals` | `expected: string` | Compose `assertContentDescriptionEquals()` |
| `assert_content_description_contains` | `substring: string` | Substring complement to `_equals`. |
| `assert_error_text_equals` | `expected: string` | Espresso `hasErrorText(text)` for form-validation labels. |
| `assert_hint_equals` | `expected: string` | Espresso `withHint(text)`. |
| `assert_input_type` | `class_substring: string` | Espresso `withInputType(...)` — externally approximated by substring-matching the node's accessibility class (e.g. `"EditText"` matches any text input field; `"NumberDecimalInput"` is unlikely to match since the class is always `android.widget.EditText`). Does not expose the in-process `InputType` bitmask. |

### Geometry (require explicit `density`)

Compose's `dp` assertions need a density factor (px-per-dp). Call `device_get_screen_size` first to get the device's density, then pass it in.

| Tool | Args | Maps to |
| --- | --- | --- |
| `assert_width_dp` | `dp: int`, `density?: number` | Compose `assertWidthIsEqualTo(dp)` |
| `assert_height_dp` | `dp: int`, `density?: number` | Compose `assertHeightIsEqualTo(dp)` |
| `assert_width_at_least_dp` | `dp: int`, `density?: number` | Compose `assertWidthIsAtLeast(dp)` |
| `assert_height_at_least_dp` | `dp: int`, `density?: number` | Compose `assertHeightIsAtLeast(dp)` |
| `assert_position_in_root` | `x: int`, `y: int`, `tolerancePx?: int` | Compose `assertPositionInRootIsEqualTo` (pixel-based) |

### Tree shape

| Tool | Args | Maps to |
| --- | --- | --- |
| `assert_is_root` | — | `isRoot()` |
| `assert_has_child_count` | `count: int` | `hasChildCount(n)` |
| `assert_has_minimum_child_count` | `count: int` | `hasMinimumChildCount(n)` |
| `assert_has_descendant` | `descendant: <matcher>` | `hasDescendant(matcher)` |

### Collections

| Tool | Args | Maps to |
| --- | --- | --- |
| `assert_count_equals` | `expected: int` | Compose `assertCountEquals(n)` |
| `assert_any` | `sub: <matcher>` | Compose `assertAny(matcher)` |
| `assert_all` | `sub: <matcher>` | Compose `assertAll(matcher)` |

## Espresso ViewActions / Compose actions

All action tools return an `ActionResult`. They re-snapshot the tree, locate the matched node, compute its centre, and dispatch the action.

| Tool | Args | Maps to |
| --- | --- | --- |
| `click` | — | Espresso `click()` / Compose `performClick()` |
| `double_click` | — | Espresso `doubleClick()` |
| `long_click` | `durationMs?: int` (default 800) | Espresso `longClick()` |
| `type_text` | `text: string`, `submit?: bool` | Espresso `typeText()` / Compose `performTextInput()` (clicks first to focus) |
| `replace_text` | `text: string`, `submit?: bool` | Espresso `replaceText()` / Compose `performTextReplacement()` (clears via CTRL+A + DEL on Android 12+, MOVE_END + DEL spam otherwise) |
| `clear_text` | — | Espresso `clearText()` / Compose `performTextClearance()` |
| `submit_text` | — | Espresso `pressImeActionButton()` (focuses the matched field then presses ENTER) |
| `swipe_node` | `direction: "up"|"down"|"left"|"right"`, `durationMs?: int` | Espresso `swipeUp/Down/Left/Right` scoped to the matched view |
| `slow_swipe_node` | `direction` | Espresso `slowSwipeLeft`/etc. (~1500ms) |
| `scroll_to` | `container?: <matcher>`, `maxAttempts?: int` (12), `direction?: "auto"|"up"|"down"|"left"|"right"` | Espresso `scrollTo()` / Compose `performScrollToNode()`. With `container` set, restrict swipes to that scrollable; otherwise the largest visible scrollable is used. |
| `scroll_to_index` | `index: int`, `direction?: "up"|"down"|"left"|"right"` (default "up") | Compose `performScrollToIndex()`. External approximation — dispatches `index` page-sized swipes. |
| `drag_node` | `from: <matcher>`, `to: <matcher>`, `durationMs?: int` (600) | Drag the centre of `from` to the centre of `to`. Reorderable lists, drag-and-drop, slider thumbs. |
| `perform_ime_action` | — | Espresso `pressImeActionButton()` |
| `perform_key_press` | `key: string`, `ctrl?`, `shift?`, `alt?` | Compose `performKeyPress()`. Modifiers use `input keycombination` on Android 12+; on older devices the key alone is dispatched and the result reports the missing coverage. |
| `assert_clickable_and_click` | — | Convenience: assert clickable then click. Returns the combined assert + click result. |

## Synchronization

`wait_*` tools are read-only and return a `WaitResult`. Default `timeoutMs` 5000–10000 depending on tool; default `intervalMs` 250.

| Tool | Args | Behaviour |
| --- | --- | --- |
| `wait_until_visible` | `match`, `timeoutMs?`, `intervalMs?` | Poll until any matched element is displayed (Compose `waitUntilExists`). |
| `wait_until_not_visible` | `match`, `timeoutMs?`, `intervalMs?` | Poll until no matched element is displayed (Compose `waitUntilDoesNotExist`). |
| `wait_until_text` | `match`, `expected: string`, `timeoutMs?`, `intervalMs?` | Poll until any matched element contains `expected` text. |
| `wait_until_count` | `match`, `count: int`, `timeoutMs?`, `intervalMs?` | Poll until matcher resolves to exactly `count` nodes. |
| `wait_until_at_least_one_exists` | `match`, `count?: int` (default 1), `timeoutMs?`, `intervalMs?` | Poll until at least `count` nodes match (Compose `waitUntilAtLeastOneExists`). |
| `wait_until_enabled` | `match`, `timeoutMs?`, `intervalMs?` | Poll until the matched element is enabled (e.g. button enables after form fills). |
| `wait_until_clickable` | `match`, `timeoutMs?`, `intervalMs?` | Poll until the matched element is clickable. |
| `wait_until_checked` | `match`, `timeoutMs?`, `intervalMs?` | Poll until the matched element is checked. |
| `wait_until_focused` | `match`, `timeoutMs?`, `intervalMs?` | Poll until the matched element has focus. |
| `wait_for_idle` | `timeoutMs?`, `idleWindowMs?` (default 500) | Heuristic for `Espresso.onIdle()` — poll the tree, wait for two snapshots to hash identically over the idle window. |

## Espresso top-level

| Tool | Maps to |
| --- | --- |
| `espresso_press_back` | `pressBack()` |
| `press_back_unconditionally` | `pressBackUnconditionally()` (externally identical) |
| `close_soft_keyboard` | `closeSoftKeyboard()` (presses BACK) |
| `open_overflow_menu` | `openActionBarOverflowOrOptionsMenu()` |
| `open_contextual_action_mode_menu` | `openContextualActionModeOverflowMenu()` |

## Espresso-Intents (recording-only)

Stubbing (`intending().respondWith(...)`) is impossible from outside the app process. We support the *recording* half via `ActivityManager` logcat scrape.

| Tool | Read-only | Args | Behaviour |
| --- | --- | --- | --- |
| `intent_monitor_start` | no | `package?` | Clears the device logcat buffer and opens a recording window for this device. `Intents.init()` analogue. |
| `intent_monitor_stop` | no | — | Closes the recording window. `Intents.release()` analogue. |
| `intent_list_captured` | yes | — | Parses `ActivityManager: START …` lines into `[{ action, data, category, package, class, raw, when }]`. |
| `assert_intent_sent` | yes | `action?`, `data?`, `dataContains?`, `package?`, `category?` | Espresso `intended(matcher)`. |
| `assert_intent_count` | yes | `expected: int` + matcher fields | Espresso `intended(matcher, times(n))`. |

---

## Keycode map

`press_key` and `perform_key_press` accept these keynames (case-insensitive; `KEYCODE_` prefix optional):

```
BACK, HOME, RECENTS, APP_SWITCH, MENU, ENTER, TAB, SPACE, DEL, ESCAPE,
DPAD_UP, DPAD_DOWN, DPAD_LEFT, DPAD_RIGHT, DPAD_CENTER,
VOLUME_UP, VOLUME_DOWN, VOLUME_MUTE, MUTE, POWER, WAKEUP, SLEEP,
CAMERA, SEARCH, BRIGHTNESS_UP, BRIGHTNESS_DOWN,
MEDIA_PLAY_PAUSE, MEDIA_NEXT, MEDIA_PREVIOUS, MEDIA_STOP, NOTIFICATION,
PASTE, COPY, CUT, MOVE_END, MOVE_HOME, FORWARD_DEL,
CTRL_LEFT, CTRL_RIGHT, SHIFT_LEFT, SHIFT_RIGHT, ALT_LEFT, ALT_RIGHT,
PAGE_UP, PAGE_DOWN, NUMPAD_ENTER,
A..Z (single uppercase letter, mapped to Android keycodes 29..54),
0..9 (single digit, mapped to keycodes 7..16)
```
