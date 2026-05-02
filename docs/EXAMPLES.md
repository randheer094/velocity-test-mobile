# Examples

Realistic LLM-agent test flows expressed as MCP tool calls. Each example is a sequence of `tools/call` JSON-RPC payloads — you can copy them straight into the MCP Inspector or send them from any MCP client.

## 1. The basic verify-and-click pattern

> *"Verify Login is visible. Verify Continue is clickable, then click it."*

```jsonc
// 1. Assert Login is visible.
{ "tool": "assert_visible",
  "args": { "match": { "text": "Login" } } }

// 2. Assert + click Continue in one shot.
{ "tool": "assert_clickable_and_click",
  "args": { "match": { "text": "Continue" } } }
```

## 2. Login flow with text input + idle wait

```jsonc
// Disable animations once at session start.
{ "tool": "animations_set", "args": { "scale": 0 } }

// Clean slate.
{ "tool": "app_clear_data", "args": { "package": "com.example.app" } }
{ "tool": "app_launch",     "args": { "package": "com.example.app" } }

// Wait for the splash to settle.
{ "tool": "wait_for_idle", "args": { "idleWindowMs": 800 } }

// Type credentials. testTag is fine if the app set testTagsAsResourceId=true,
// otherwise switch to contentDescription / hint.
{ "tool": "type_text",
  "args": { "match": { "testTag": "username" }, "text": "alice" } }

{ "tool": "type_text",
  "args": { "match": { "testTag": "password" }, "text": "s3cret",
            "submit": false } }

{ "tool": "click",
  "args": { "match": { "text": "Sign in" } } }

// Wait for the home screen to appear.
{ "tool": "wait_until_visible",
  "args": { "match": { "text": "Welcome, alice" }, "timeoutMs": 10000 } }
```

## 3. Asserting list contents

```jsonc
// Exactly five orders should be visible.
{ "tool": "assert_count_equals",
  "args": { "match":    { "resourceId": "order_row" },
            "expected": 5 } }

// The third row should be currently selected.
{ "tool": "assert_selected",
  "args": { "match": { "resourceId": "order_row", "parentIndex": 2 } } }

// Every row should be clickable.
{ "tool": "assert_all",
  "args": { "match": { "resourceId": "order_row" },
            "sub":   { "clickable": true } } }
```

## 4. Scrolling to a target row in a long list

```jsonc
// Scroll the LazyColumn until "Item 137" is visible, then click it.
{ "tool": "scroll_to",
  "args": { "match":     { "text": "Item 137" },
            "container": { "resourceId": "items_list", "scrollable": true },
            "direction": "up",
            "maxAttempts": 30 } }

{ "tool": "click", "args": { "match": { "text": "Item 137" } } }
```

## 5. Compose `mainClock` substitute: poll a counter

Compose's in-process `mainClock.advanceTimeBy(...)` doesn't exist externally. Use `wait_until_text`:

```jsonc
{ "tool": "wait_until_text",
  "args": { "match":     { "testTag": "counter" },
            "expected":  "10",
            "timeoutMs": 15000 } }
```

## 6. Visual regression

```jsonc
// 1. Capture a screenshot to a known path.
{ "tool": "screen_capture",
  "args": { "saveTo": "/tmp/baseline.png" } }

// 2. After a code change, capture a candidate.
{ "tool": "screen_capture",
  "args": { "saveTo": "/tmp/candidate.png" } }

// 3. Diff with a 5% tolerance and emit a highlight image.
{ "tool": "screen_diff",
  "args": { "pathA": "/tmp/baseline.png",
            "pathB": "/tmp/candidate.png",
            "diffOutput":   "/tmp/diff.png",
            "tolerance":    4,
            "thresholdPct": 5.0 } }
```

## 7. Deep-link test (intent dispatch + assertion)

```jsonc
// Start recording dispatched intents.
{ "tool": "intent_monitor_start",
  "args": { "package": "com.example.app" } }

// Send a deep-link intent.
{ "tool": "intent_send",
  "args": { "action":  "android.intent.action.VIEW",
            "data":    "myapp://order/42",
            "package": "com.example.app" } }

// Verify the app issued the expected outgoing intent during this window.
{ "tool": "assert_intent_sent",
  "args": { "action":       "android.intent.action.VIEW",
            "dataContains": "/order/42" } }

// Tear down.
{ "tool": "intent_monitor_stop", "args": {} }
```

## 8. Geometry assertion (Compose `assertWidthIsEqualTo(dp)`)

```jsonc
// Find the device density first.
{ "tool": "device_get_screen_size", "args": {} }
//   → { width: 1080, height: 2400, density: 3 }

// Now assert the button is exactly 56 dp wide.
{ "tool": "assert_width_dp",
  "args": { "match":   { "testTag": "fab" },
            "dp":      56,
            "density": 3 } }
```

## 9. Sibling correctness — RecyclerView with duplicate text

When a list has multiple rows with **identical content** (e.g. three "Item" entries), use `parentIndex` rather than relying on natural ordering:

```jsonc
// The middle of three sibling items.
{ "tool": "click",
  "args": { "match": {
      "className":   "Item",
      "parentIndex": 1,
      "hasAncestor": { "resourceId": "list_container" }
  } } }
```

## 10. Permission-driven test setup

```jsonc
// Grant the runtime camera permission for this test.
{ "tool": "permission_grant",
  "args": { "package":    "com.example.app",
            "permission": "android.permission.CAMERA" } }

// Verify the granted state.
{ "tool": "app_get_info", "args": { "package": "com.example.app" } }
//   → grantedPermissions includes android.permission.CAMERA
```

## 11. Run-as data inspection (debuggable builds only)

```jsonc
// Read the SharedPreferences XML file written by the app under test.
{ "tool": "app_data_read",
  "args": { "package":      "com.example.app",
            "relativePath": "shared_prefs/settings.xml" } }
```

## 12. Espresso conveniences

```jsonc
// Open the action-bar overflow, then click "Settings".
{ "tool": "open_overflow_menu", "args": {} }
{ "tool": "wait_until_visible",
  "args": { "match": { "text": "Settings" }, "timeoutMs": 2000 } }
{ "tool": "click", "args": { "match": { "text": "Settings" } } }

// Press BACK to leave.
{ "tool": "espresso_press_back", "args": {} }
```

## 13. CTRL+A clear (Android 12+) via perform_key_press

```jsonc
// Focus the field then chord CTRL+A and DEL.
{ "tool": "click",
  "args": { "match": { "testTag": "noteField" } } }
{ "tool": "perform_key_press",
  "args": { "match": { "testTag": "noteField" }, "key": "A", "ctrl": true } }
{ "tool": "press_key", "args": { "key": "DEL" } }
```

(Or just call `clear_text({match})` — it does this internally with a fallback for older devices.)
