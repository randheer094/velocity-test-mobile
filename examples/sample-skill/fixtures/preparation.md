# preparation

Per-test setup. Every test in `tests/` references one of the named
procedures below as its **Pre-conditions**. The agent walks each step in
order; if any step fails, the test is marked FAIL before its first real
assertion runs.

## Standard pre-conditions

The default reset. Brings the device and the app to a known starting
state: animations off, app data cleared, runtime permissions pre-granted,
app launched on its home screen.

1. Disable system animations.
2. Reset app data for `com.example.notes`.
3. Pre-grant `android.permission.POST_NOTIFICATIONS` so the runtime
   prompt does not steal focus when the app first posts.
4. Launch `com.example.notes`.
5. Wait for the screen to settle.
6. *"All notes"* appears within 5s. (This is the home-screen marker; if
   it doesn't show, the launch failed or the app crashed during start.)

> **Why pre-grant before launch?** Android shows the runtime permission
> dialog the first time the app requests it, which races with the home
> screen and breaks deterministic selectors. Granting via
> `permission_grant` before `app_launch` avoids the prompt entirely.

### Step-to-tool mapping (for the agent)

The vocabulary covers the prose; this block exists so the agent does not
have to re-derive it for setup. Authoring tests should reference the
prose above, not this table.

| # | Tool call |
| --- | --- |
| 1 | `animations_set({ scale: 0 })` |
| 2 | `app_clear_data({ package: "com.example.notes" })` |
| 3 | `permission_grant({ package: "com.example.notes", permission: "android.permission.POST_NOTIFICATIONS" })` |
| 4 | `app_launch({ package: "com.example.notes" })` |
| 5 | `wait_for_idle({ idleWindowMs: 800 })` |
| 6 | `wait_until_visible({ match: { text: "All notes" }, timeoutMs: 5000 })` |

## resetNotes

Sub-procedure for tests that need a deterministic set of seeded notes.
Run **after** `Standard pre-conditions`. Two paths — prefer the intent
fast-path; fall back to the UI if the debug intent doesn't seed.

### Fast path: debug intent

The debug build of the app exposes a developer-only broadcast that wipes
the notes table and inserts a fixed seed:

> *Send a broadcast intent to `com.example.notes` with
> `action = com.example.notes.debug.RESET_NOTES`. Within 3s, the
> "All notes" list shows exactly 5 rows.*

The intent is registered only in the debug manifest, so this is safe:
release builds do nothing.

### Fallback: UI seeding

If `assert_count_equals` reports 0 rows after the broadcast (the manifest
was not updated, or the receiver was suppressed), seed via the UI.
Repeat five times:

1. Tap the *"+"* FAB on the home screen.
2. Type the note title (`Note 1`, `Note 2`, …) into the title field.
3. Tap *"Save"*.
4. *"All notes"* appears within 3s (back on the home screen).

After seeding, *"All notes"* shows exactly 5 rows.

## Notes on idempotency

`Standard pre-conditions` is intentionally idempotent — running it twice
in a row produces the same end state. `resetNotes` is also idempotent
once the home screen is reachable. Tests in the same file can re-apply
either procedure between cases without explicit cleanup, but most tests
prefer the canonical pattern: pre-conditions → steps → cleanup.
