# vocabulary

The verb → MCP-tool translation key. Sits in one place so test files don't
have to spell out tool names step-by-step. When you read a step in a
runbook, find the verb here and dispatch the corresponding tool.

All matcher-bearing tools take a `match` object. The matcher fields below
are the common ones; see `docs/MATCHERS.md` in the velocity-test-mobile
repo root for the full vocabulary.

## Visibility & existence

| Verb in the runbook | Tool the agent dispatches |
| --- | --- |
| *"X is shown"* / *"shows X"* / *"X is visible"* | `assert_visible({ match: { text: "X" } })` |
| *"X is not shown"* / *"X is hidden"* | `assert_not_visible({ match: { text: "X" } })` |
| *"X exists"* / *"X is present in the tree"* | `assert_exists({ match: ... })` |
| *"X does not exist"* | `assert_does_not_exist({ match: ... })` |

## Tap, type, swipe

| Verb in the runbook | Tool the agent dispatches |
| --- | --- |
| *"tap X"* | `find_node({ match: { text: "X" } })` then `click({ match: ... })` |
| *"long-press X"* | `long_click({ match: ... })` |
| *"double-tap X"* | `double_click({ match: ... })` |
| *"type Y into the X field"* | `type_text({ match: ..., text: "Y" })` |
| *"replace the X field with Y"* | `replace_text({ match: ..., text: "Y" })` |
| *"clear the X field"* | `clear_text({ match: ... })` |
| *"submit the X field"* | `type_text({ match: ..., text: "...", submit: true })` |
| *"press BACK"* | `espresso_press_back({})` |
| *"press HOME"* | `press_key({ key: "HOME" })` |
| *"close the keyboard"* | `close_soft_keyboard({})` |

## Wait & sync

| Verb in the runbook | Tool the agent dispatches |
| --- | --- |
| *"X appears within Ns"* | `wait_until_visible({ match: ..., timeoutMs: N*1000 })` |
| *"X disappears within Ns"* | `wait_until_not_visible({ match: ..., timeoutMs: N*1000 })` |
| *"X reads exactly Y within Ns"* | `wait_until_text({ match: ..., expected: "Y", timeoutMs: N*1000 })` |
| *"wait for the screen to settle"* | `wait_for_idle({ idleWindowMs: 800 })` |

If the test omits "within Ns", use the default 5000ms from `SKILL.md`.

## Lists

| Verb in the runbook | Tool the agent dispatches |
| --- | --- |
| *"the list shows N rows"* | `assert_count_equals({ match: ..., expected: N })` |
| *"every row is clickable"* | `assert_all({ match: ..., sub: { clickable: true } })` |
| *"at least one row matches X"* | `assert_any({ match: ..., sub: { ... } })` |
| *"the Kth row is selected"* | `assert_selected({ match: { ..., parentIndex: K-1 } })` |
| *"scroll the list until X is visible"* | `scroll_to({ match: { text: "X" }, container: ..., maxAttempts: 30 })` |

## App / system lifecycle

| Verb in the runbook | Tool the agent dispatches |
| --- | --- |
| *"reset app data"* | `app_clear_data({ package: "com.example.notes" })` |
| *"launch the app"* | `app_launch({ package: "com.example.notes" })` |
| *"terminate the app"* | `app_terminate({ package: "com.example.notes" })` |
| *"disable animations"* | `animations_set({ scale: 0 })` |
| *"grant the X permission"* | `permission_grant({ package: ..., permission: "android.permission.X" })` |
| *"the foreground service is running"* | `service_wait_for_state({ bundle_id: ..., foreground: true, timeout_ms: 5000 })` |
| *"the top activity is X"* | `activity_wait_for_top({ bundle_id: ..., activity: "X", timeout_ms: 5000 })` |

> Note the parameter casing: the testing surface (matchers, `timeoutMs`,
> `maxAttempts`) is camelCase, while the system surface
> (`bundle_id`, `timeout_ms`, `channel_id`) is snake_case. The vocabulary
> hides this from the runbooks; the agent picks the right casing per
> tool.

## Diagnostics

| Verb in the runbook | Tool the agent dispatches |
| --- | --- |
| *"capture a screenshot"* | `screen_capture({ saveTo: "/tmp/<test-id>.png" })` |
| *"dump the tree"* | `print_tree({})` |
| *"clear logcat"* | `logcat_clear({})` |

## When prose is not enough — drop to explicit syntax

Stay in prose unless one of these is true. Keep just the disambiguator
inline; do not revert whole steps to call syntax.

### 1. Two visible labels collide

If two nodes share the same text — e.g. a card titled *"Settings"* and a
button labelled *"Settings"* — `text` is ambiguous. Annotate the
disambiguator inline:

> *Tap "Settings" — match by `contentDescription = "open-settings"`; the
> card title above collides under text matching.*

The agent uses `find_node({ match: { contentDescription: "open-settings" } })
→ click` rather than the default `text` lookup.

### 2. Non-default timeout

Don't bury a 15s settings-app wait in the default. Call it out:

> *The Settings app's top activity is `com.android.settings/.Settings`
> within 15s.*

### 3. Numeric tolerance / polled equality

Coordinate, dp, count, or count-down assertions with an explicit ε:

> *Within 5s, `location_get_last_known` returns `lat ≈ 59.3383` and
> `lng ≈ 18.0549` (±0.01° per axis).*

### 4. Recording-only intent assertions

The intent monitor lifecycle (start → act → assert → stop) is too
tool-shaped for prose to add value:

> *Start the intent monitor on `com.example.notes`. Send the deep-link
> `notes://note/42`. Assert one outgoing VIEW intent with `data`
> containing `/note/42`. Stop the monitor.*

### 5. Visual regression

Pixel diffs need explicit paths and tolerance:

> *Capture a baseline to `/tmp/note-edit.baseline.png`. After the edit,
> capture a candidate to `/tmp/note-edit.candidate.png`. Diff with
> `tolerance = 4` and `thresholdPct = 1.0`; emit the highlight to
> `/tmp/note-edit.diff.png`.*

For everything else, prose is the default.
