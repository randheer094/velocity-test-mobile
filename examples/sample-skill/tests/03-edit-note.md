# 03 — edit-note

**Source:** worked example.

**Defaults:** package `com.example.notes`; default timeout `5s`.

This file deliberately exercises the **exception cases** from
`fixtures/vocabulary.md` so consumers see when prose gives way to
explicit annotations: a label collision, a non-default timeout, and a
numeric tolerance.

## File-level pre-conditions

1. Run `fixtures/preparation.md` → **Standard pre-conditions**.
2. Run `fixtures/preparation.md` → **resetNotes** (seeds five notes).

---

## Test 1: edit the first note and save — exception cases

### Steps

1. Tap the first row in the notes list. (Match by
   `resourceId = "note_row"` plus `parentIndex = 0`; the row's text
   "Note 1" also collides with the *body* of "Note 10" once a deeper
   seed is loaded, so the resource id is the safer selector even at
   `count = 5`.)

2. The detail screen shows "Note #1" within 3s.

3. Tap the edit affordance.

   > *Match by `contentDescription = "edit-note"` — the toolbar title
   > "Edit Note" and the floating *"Edit"* button both render the text
   > "Edit", and `text` matching is ambiguous between them. The button
   > carries a `contentDescription`; the toolbar title does not.*

4. Clear the Body field. (Match by `testTag = "noteBody"`.)

5. Type `updated body` into the Body field.

6. Tap "Save".

   > *Match "Save" by `contentDescription = "save-note"`. The body
   > field's placeholder text *"Save your thoughts…"* matches "Save"
   > under `textContains` and produces a false positive on slow
   > emulators where the placeholder is still in the tree.*

7. The toast "Saved" appears within 15s. (Saves go through a debounced
   network round-trip in this sample app — the default 5s default is
   not enough on cold starts.)

8. Within 5s, the detail screen's "Updated" timestamp matches the
   current wall clock to within ±2 seconds.

   > *This is a polled equality with a numeric tolerance — keep it in
   > explicit form. Drive the assertion with
   > `wait_until_text({ match: { testTag: "updatedAt" }, expected:
   > "<format(now)>", timeoutMs: 5000 })`, where `<format(now)>` is the
   > app's `MM/DD HH:mm` rendering of "now". On failure, capture the
   > actual rendered text via `find_node` so the report says what the
   > app showed instead.*

9. Press BACK.

10. The notes list's first row's title reads "Note 1" (the title is
    unchanged; only the body and timestamp moved).

### Cleanup

Run `fixtures/teardown.md` → **Standard cleanup**.

> **Why three annotations in one test?** This is the demo for the
> exception-case rules in `fixtures/vocabulary.md`. A real test should
> not need three; if yours does, the matchers / labels in your app
> probably warrant a refactor.
