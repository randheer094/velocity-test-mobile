# 02 — note-list

**Source:** worked example.

**Defaults:** package `com.example.notes`; default timeout `5s`.

## File-level pre-conditions (run before every test in this file)

1. Run `fixtures/preparation.md` → **Standard pre-conditions**.
2. Run `fixtures/preparation.md` → **resetNotes** (seeds five notes
   titled `Note 1` … `Note 5`).

These tests assume the user is already authenticated; the Notes app
remembers the session across launches in debug builds. If a future
release breaks that assumption, prepend Test 1 of `01-login.md` here.

---

## Test 1: the seeded list renders five clickable rows, third selected

### Steps

1. The notes list shows exactly 5 rows.
2. Every row is clickable.
3. The third row (titled "Note 3") is currently selected.

   > *Match the row by `resourceId = "note_row"` plus `parentIndex = 2`.
   > Plain text matching on "Note 3" works too, but the resource id
   > version stays correct if the seeded titles are renamed.*

4. The first row's title reads "Note 1".
5. The fifth row's title reads "Note 5".

### Cleanup

Run `fixtures/teardown.md` → **Standard cleanup**.

---

## Test 2: scroll to a deep row and open it

This test exercises a deeper seed (137 notes via the debug intent's
`count` extra). The notes list lazily renders, so the target row is not
in the tree until scrolled into view.

### Steps

1. Run `fixtures/preparation.md` → **resetNotes** with `count = 137`
   (the debug intent accepts an int extra; the UI fallback is too slow
   for this case — fail the test if the intent path doesn't seed).
2. The notes list shows exactly 137 rows.
3. Scroll the notes list until "Note 137" is visible (max 30 attempts;
   container is the scrollable with `resourceId = "notes_list"`).
4. Tap "Note 137".
5. The detail screen shows "Note #137" within 3s.
6. The detail screen's "Body" field is not empty.

### Cleanup

Run `fixtures/teardown.md` → **Standard cleanup**.
