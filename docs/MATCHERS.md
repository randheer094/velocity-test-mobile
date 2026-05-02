# Matchers

Every testing tool takes a `match` selector. The vocabulary is the same across find / assert / act / wait, modelled on Espresso ViewMatchers and Compose SemanticsMatchers.

A matcher is a JSON object with any combination of fields. **An empty matcher matches nothing** — tools reject empty matchers up front so the LLM can't accidentally select random nodes.

## Identity

| Field | Type | Maps to |
| --- | --- | --- |
| `text` | string | `withText(text)` / `hasText(text)` |
| `textContains` | string | `withSubstring(s)` |
| `textRegex` | string (Go regex) | (no direct equivalent) |
| `contentDescription` | string | `withContentDescription(d)` / `hasContentDescription(d)` |
| `contentDescriptionContains` | string | substring variant |
| `resourceId` | string | `withResourceName(id)` — accepts a fully-qualified id (`com.app:id/foo`) or just the suffix (`foo`) |
| `testTag` | string | Compose `hasTestTag(tag)` — accepts the suffix or the full resource-id |
| `className` | string | `withClassName(s)` (substring match) |
| `hint` | string | `withHint(s)` |
| `package` | string | restrict to a specific app's nodes |
| `errorText` | string | `hasErrorText(s)` |

### `testTag` and Compose

For Compose `Modifier.testTag("loginBtn")` to be matchable from outside the app, **the app must enable** `Modifier.semantics { testTagsAsResourceId = true }`:

```kotlin
Button(
  modifier = Modifier
    .testTag("loginBtn")
    .semantics { testTagsAsResourceId = true }
) { … }
```

Without that flag, the testTag never reaches the accessibility tree and external tools can't see it. Fall back to `contentDescription`/`text` for app builds that haven't opted in.

## State flags (any subset)

All boolean fields are tri-state via JSON: omit = "don't care", `true` = require, `false` = require not.

| Field | Maps to |
| --- | --- |
| `clickable` | `isClickable` / `isNotClickable` / Compose `hasClickAction` |
| `longClickable` | `isLongClickable` |
| `enabled` | `isEnabled` / `isNotEnabled` / Compose `assertIsEnabled` |
| `checkable` | (Compose `isToggleable` analogue) |
| `checked` | `isChecked` / `isNotChecked` |
| `focused` | `isFocused` / `hasFocus` |
| `focusable` | `isFocusable` / `isNotFocusable` |
| `selected` | `isSelected` |
| `scrollable` | (UIAutomator scrollable flag) |
| `displayed` | Espresso `isDisplayed` / Compose `assertIsDisplayed` — non-zero bounds AND `visibleToUser` |
| `completelyDisplayed` | Espresso `isCompletelyDisplayed` — node fully on-screen, not partially clipped |
| `displayingAtLeastPercent` | Espresso `isDisplayingAtLeast(percent)` — integer 1..100 |
| `on` | Compose `isOn` — alias for `checked: true` |
| `off` | Compose `isOff` — alias for `checked: false` |
| `toggleable` | Compose `isToggleable` — alias for `checkable` |

## Tree shape

| Field | Maps to |
| --- | --- |
| `isRoot` | `isRoot()` |
| `childCount` | `hasChildCount(n)` (exact direct-child count) |
| `minChildCount` | `hasMinimumChildCount(n)` |
| `parentIndex` | `withParentIndex(i)` (Nth child of parent, 0-indexed) |

Note: `parentIndex` works correctly even when sibling nodes have **identical content** (text/bounds/class) — it uses the actual flat-tree index, not content equality.

## IME / input type

| Field | Maps to |
| --- | --- |
| `hasImeAction` | `hasImeAction()` — best-effort externally (focusable + editable class) |
| `inputType` | `withInputType(...)` — substring match against the node's class |

## Hierarchy combinators

Each is a nested matcher:

| Field | Maps to |
| --- | --- |
| `hasAncestor` | `isDescendantOfA(matcher)` |
| `hasDescendant` | `hasDescendant(matcher)` |
| `hasParent` | `withParent(matcher)` |
| `hasSibling` | `hasSibling(matcher)` |

```jsonc
// "Item 1" inside any scrollable container with an Item 2 sibling somewhere.
{
  "text": "Item 1",
  "hasAncestor": { "scrollable": true },
  "hasSibling":  { "text": "Item 2" }
}
```

## Logical combinators

| Field | Maps to |
| --- | --- |
| `not` | `not(matcher)` |
| `allOf` | `allOf(matcher, …)` |
| `anyOf` | `anyOf(matcher, …)` |

```jsonc
// Buttons that are enabled and not disabled-styled.
{
  "allOf": [{ "className": "Button" }, { "enabled": true }],
  "not":   { "contentDescription": "Disabled" }
}
```

## Disambiguation

| Field | Behaviour |
| --- | --- |
| `instance` | When multiple nodes match, pick the Nth (0-indexed). Default 0. |

## Worked examples

```jsonc
// 1. The first visible Login button.
{ "text": "Login", "displayed": true }

// 2. A specific row in a list — the second item.
{ "resourceId": "list_item", "instance": 1 }

// 3. A toggle inside a settings list, currently on.
{ "className": "Switch", "on": true,
  "hasAncestor": { "resourceId": "settings_container" } }

// 4. A clickable row that contains a particular icon.
{ "clickable": true,
  "hasDescendant": { "contentDescription": "Star" } }

// 5. The third sibling Item — works even when Item children share text.
{ "className": "Item", "parentIndex": 2 }

// 6. Match by regex over text.
{ "textRegex": "^\\$[0-9]+\\.[0-9]{2}$" }

// 7. A compose testTag on a build that opted in.
{ "testTag": "loginBtn" }

// 8. An EditText for password input.
{ "className": "EditText", "inputType": "Password" }

// 9. Any error-state form field.
{ "errorText": "Required" }
```

## How matchers are evaluated

`FindAll` flattens the hierarchy into a slice once and walks every node, applying:

1. **Local predicates** (text / state / class / hint / errorText / IME / input type / on/off/toggleable / counts) directly.
2. **Tree-position predicates** (`isRoot`, `parentIndex`, `completelyDisplayed`, `displayingAtLeastPercent`) using the flat slice and the root viewport.
3. **Hierarchy combinators** by recursively re-running the matcher engine against ancestors/descendants/siblings/parents — by **flat index**, not by content, so duplicate sibling content can't confuse the result.
4. **Logical combinators** (`not`/`allOf`/`anyOf`).

`Find(root, m)` returns the matcher's `instance`-th match (default 0). Tools that need the count (`count_nodes`, `assert_count_equals`, `wait_until_count`) call `Count`/`FindAll` and read `len(...)`.
