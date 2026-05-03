# 01 — login

**Source:** none — this is a worked example, not a port from a real
TypeScript suite. Treat the prose as the spec.

**Defaults:** package `com.example.notes`; default `wait_until_visible`
timeout `5s`; default scroll attempts `30`.

## File-level pre-conditions (run before every test in this file)

Run `fixtures/preparation.md` → **Standard pre-conditions**, except
*step 6* — these tests start on the **login** screen, not the home
screen. Replace step 6 with:

> *"Sign in" is shown within 5s.* (The login screen's title.)

This per-file override is the one place pre-conditions deviate; tests in
files 02 and 03 use the standard sequence verbatim.

---

## Test 1: valid credentials land on the home screen

### Steps

1. The login screen shows "Username" and "Password" fields and a "Sign
   in" button.
2. Type `alice` into the Username field.
3. Type `s3cret` into the Password field.
4. Tap "Sign in".
5. "Welcome, alice" appears within 10s.
6. The notes list shows exactly 5 rows, all clickable.

### Cleanup

Run `fixtures/teardown.md` → **Standard cleanup**.

---

## Test 2: wrong password keeps the user on the login screen

### Steps

1. Type `alice` into the Username field.
2. Type `nope` into the Password field.
3. Tap "Sign in".
4. "Invalid credentials" appears within 3s.
5. "Welcome, alice" is not shown.
6. The Password field is empty (the app clears it on rejection).

   > *Match the field by `contentDescription = "password"` rather than
   > the placeholder text "Password" — the placeholder collides with the
   > error label "Password too short" that the app sometimes renders
   > below the field.*

### Cleanup

Run `fixtures/teardown.md` → **Standard cleanup**.
