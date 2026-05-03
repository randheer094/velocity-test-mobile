# Run E2E (sample-notes)

Black-box tests for a hypothetical Notes app (`com.example.notes`),
expressed as Markdown runbooks an LLM agent walks step-by-step. The runbook
talks in **prose** ("tap *Save*", "the list shows 5 rows"); the agent
translates each verb into the right `velocity-test-mobile` MCP tool call
using the [vocabulary](fixtures/vocabulary.md). Five tests across three
files plus three shared fixtures.

This folder is the canonical worked example for the prose-runbook style.
Copy it, swap the package id and the screen labels for your own app, and
adapt the fixtures.

## Layout

```
examples/sample-skill/
  SKILL.md                       ← you are here
  fixtures/
    vocabulary.md                ← verb → tool translation key
    preparation.md               ← Standard pre-conditions + resetNotes
    teardown.md                  ← Standard cleanup
    flows.md                     ← login, goToHome, logout (reusable named procedures)
  tests/
    01-login.md                  ← 2 tests (verifies flows.md → login)
    02-note-list.md              ← 2 tests (uses flows.md → login + goToHome)
    03-edit-note.md              ← 1 test (uses flows.md, exception cases)
```

## Preflight — run before walking any test

Run all three. Any failure: stop, report, do **not** start tests. The skill
does not boot the emulator or build the APK on the user's behalf.

1. **MCP reachable.** `ToolSearch select:mcp__velocity-test-mobile__app_launch`
   returns a schema. Failure: ask the user to check `claude mcp list`.
2. **Emulator booted.** `mcp__velocity-test-mobile__device_list` returns
   at least one device. Failure: ask the user to start their emulator.
3. **Debug APK installed.** `mcp__velocity-test-mobile__app_list` contains
   `com.example.notes`. Failure: ask the user to install the debug build
   (e.g. `./gradlew :app:installDebug`).

## Mapping user requests → what to run

| User asks | What to do |
| --- | --- |
| "Run all tests" / "run the suite" | Walk `tests/01..03` in order. Continue past failures; report a results table at the end. |
| "Run `tests/02-note-list.md`" | Walk every test in that file; report per-test pass/fail. |
| "Run test 2 of 01-login" | Walk just that one test. |
| "Smoke test" | Run only `tests/01-login.md` (cheapest signal). |

## Per-test execution loop

For every test in a file:

1. Read the test file end-to-end, plus every fixture it references
   (`fixtures/preparation.md`, `fixtures/teardown.md`,
   `fixtures/flows.md`, and `fixtures/vocabulary.md` for the verb
   mapping).
2. Resolve schemas for unfamiliar MCP tools the first time you hit them
   via `ToolSearch select:mcp__velocity-test-mobile__<name>`.
3. Apply **Pre-conditions** — usually `fixtures/preparation.md → Standard
   pre-conditions`, optionally with a sub-procedure like `resetNotes`.
4. Walk **Steps** top-to-bottom. Translate each prose verb using
   `fixtures/vocabulary.md`. Default selectors prefer `text` /
   `contentDescription`; fall back to `testTag` / `resourceId` only if
   the test calls them out.
5. On any failed assertion: capture `screen_capture` and `print_tree` for
   evidence, mark the test FAIL, **continue to the next test**.
6. Apply **Cleanup** — usually `fixtures/teardown.md → Standard cleanup`.
   Skip if the test explicitly says so.

## Standard timeouts

Picked once and reused. Raise locally if the device flakes; do not lower
without a reason.

| Timeout | Used for |
| --- | --- |
| `1000` ms | Negative assertions ("X must not appear"). |
| `3000` ms | Dialog dismiss, fast follow-up checks. |
| `5000` ms | Default `wait_until_visible`, `service_wait_for_state`. |
| `10000` ms | Auth flows, list rehydration. |
| `15000` ms | System UI navigation (Settings cold start). |

When a test needs a non-default timeout, it must say so in prose
(*"X appears within 10s"*) so the agent picks the right value.

## Failure-handling rules

- **Apparent transient** (RPC drop, single-step race): re-run the failing
  test once. If it still fails, mark FAIL — do not retry a third time.
- **Pre-condition fails before step 1:** the device or app is in a state
  the runbook didn't expect. Capture evidence, fail the test, escalate
  to the user — do **not** patch the runbook to make it pass.
- **Runbook itself looks wrong** (selector drifted, label changed):
  surface it. The skill does not silently rewrite tests.

## Reporting format

End every run with a results table the user can scan:

```
Test                                   Result
tests/01-login.md test 1               PASS
tests/01-login.md test 2               PASS
tests/02-note-list.md test 1           PASS
tests/02-note-list.md test 2           FAIL — step 4: scroll_to never found "Note 137" after 30 attempts
tests/03-edit-note.md test 1           PASS
```

For each FAIL include: failing step number, the assertion that fired, and
a one-line excerpt from `screen_capture` / `print_tree` if it clarifies
the cause.

A full pass is ~3 minutes against a single emulator.

## What this skill does NOT do

- It does **not** build the APK or boot the emulator. Those are
  prerequisites — fail fast with a clear instruction if they're missing.
- It does **not** modify the runbooks themselves. If a runbook is wrong,
  surface it; don't patch tests to make them pass.
- It does **not** retry beyond once per test. Repeated failures signal a
  real defect or a stale device — escalate.
