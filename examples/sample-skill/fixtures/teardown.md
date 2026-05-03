# teardown

Per-test cleanup. Runs after every test that lists `Cleanup → Standard
cleanup`. Brings the device back to the same end state regardless of
whether the test passed or failed, so the next test's pre-conditions
start from a predictable baseline.

## Standard cleanup

1. If the soft keyboard is up, close it. (Many tests end mid-edit; an
   open keyboard masks selectors in the next test's pre-conditions.)
2. If a dialog is visible (any node with `className` containing
   `Dialog`), press BACK once. Then assert no dialog is visible. If a
   dialog is still up after BACK, the runbook left state behind —
   capture evidence and FAIL the test.
3. Terminate `com.example.notes`. (`Standard pre-conditions` will
   re-launch on the next test.)
4. Clear logcat so the next test's evidence stays scoped to its own
   run.

### Step-to-tool mapping (for the agent)

| # | Tool call |
| --- | --- |
| 1 | `close_soft_keyboard({})` (no-op if the keyboard is already down) |
| 2a | `find_node({ match: { className: "Dialog" } })` — if found, `espresso_press_back({})` |
| 2b | `assert_not_visible({ match: { className: "Dialog" } })` |
| 3 | `app_terminate({ package: "com.example.notes" })` |
| 4 | `logcat_clear({})` |

## When to skip cleanup

A test file may chain two tests that intentionally share state — e.g.
*"Test 1: log in"* followed by *"Test 2: from the home screen, edit a
note"*. The second test's pre-conditions already assume an authenticated
session; running cleanup in between would terminate the app and force
re-login.

In that case the test file says so explicitly:

> *Cleanup: skip — `Test 2` chains from this state.*

If a test does **not** include a Cleanup line, the default is `Standard
cleanup`. Skipping is opt-in.

## Failure handling during cleanup

Cleanup steps must not mask test failures. The rules:

- A test marked PASS that **fails its cleanup** is downgraded to FAIL
  with the reason *"cleanup: <step>"*. Future tests would inherit a
  bad state; better to surface it.
- A test marked FAIL that **also fails cleanup** keeps its original
  failure reason. The cleanup failure is appended in parentheses for
  the report.
