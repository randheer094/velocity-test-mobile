package tools

// matcherSchemaDef is the JSON Schema fragment that defines the recursive
// Matcher type used by every testing tool. Stored as a literal so the
// `$ref` cycle (HasAncestor / HasDescendant / AllOf / etc.) parses cleanly,
// which the SDK's Go-type-based schema generator cannot do.
//
// We splice this into per-tool schemas by writing { ..., "match": {"$ref":"#/$defs/matcher"} }.
var matcherSchemaDef = map[string]any{
	"type":        "object",
	"description": "Espresso/Compose-style selector. At least one identifying field is required.",
	"properties": map[string]any{
		"text":                       map[string]any{"type": "string", "description": "exact text match"},
		"textContains":               map[string]any{"type": "string", "description": "substring of text"},
		"textRegex":                  map[string]any{"type": "string", "description": "Go regex matched against text"},
		"contentDescription":         map[string]any{"type": "string", "description": "exact accessibility description"},
		"contentDescriptionContains": map[string]any{"type": "string"},
		"resourceId":                 map[string]any{"type": "string", "description": "matches a resource-id; can be a fully qualified id or just the suffix after :id/"},
		"testTag":                    map[string]any{"type": "string", "description": "Compose testTag (matches resource-id; works when the app uses Modifier.semantics { testTagsAsResourceId = true })"},
		"className":                  map[string]any{"type": "string", "description": "substring match against the node's class name"},
		"hint":                       map[string]any{"type": "string"},
		"package":                    map[string]any{"type": "string"},
		"errorText":                  map[string]any{"type": "string"},

		"clickable":     map[string]any{"type": "boolean"},
		"longClickable": map[string]any{"type": "boolean"},
		"enabled":       map[string]any{"type": "boolean"},
		"checkable":     map[string]any{"type": "boolean"},
		"checked":       map[string]any{"type": "boolean"},
		"focused":       map[string]any{"type": "boolean"},
		"focusable":     map[string]any{"type": "boolean"},
		"selected":      map[string]any{"type": "boolean"},
		"scrollable":    map[string]any{"type": "boolean"},
		"displayed":     map[string]any{"type": "boolean", "description": "true means non-zero bounds AND visibleToUser"},

		"completelyDisplayed":      map[string]any{"type": "boolean", "description": "Espresso isCompletelyDisplayed — fully on-screen, not partially clipped"},
		"displayingAtLeastPercent": map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "description": "Espresso isDisplayingAtLeast — require N%% of area visible"},

		"on":         map[string]any{"type": "boolean", "description": "Compose isOn — alias for checked=true"},
		"off":        map[string]any{"type": "boolean", "description": "Compose isOff — alias for checked=false"},
		"toggleable": map[string]any{"type": "boolean", "description": "Compose isToggleable — alias for checkable"},

		"isRoot":        map[string]any{"type": "boolean", "description": "Espresso isRoot"},
		"childCount":    map[string]any{"type": "integer", "minimum": 0, "description": "Espresso hasChildCount"},
		"minChildCount": map[string]any{"type": "integer", "minimum": 0, "description": "Espresso hasMinimumChildCount"},
		"parentIndex":   map[string]any{"type": "integer", "minimum": 0, "description": "Espresso withParentIndex — Nth child of parent"},

		"hasImeAction": map[string]any{"type": "boolean", "description": "Espresso hasImeAction (best-effort externally)"},
		"inputType":    map[string]any{"type": "string", "description": "Espresso withInputType — substring match against class"},

		"hasAncestor":   map[string]any{"$ref": "#/$defs/matcher"},
		"hasDescendant": map[string]any{"$ref": "#/$defs/matcher"},
		"hasParent":     map[string]any{"$ref": "#/$defs/matcher"},
		"hasSibling":    map[string]any{"$ref": "#/$defs/matcher"},

		"not":   map[string]any{"$ref": "#/$defs/matcher"},
		"allOf": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/$defs/matcher"}},
		"anyOf": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/$defs/matcher"}},

		"instance": map[string]any{"type": "integer", "minimum": 0, "description": "0-indexed: pick the Nth match (default 0)"},
	},
	"additionalProperties": false,
}

// deviceProp is the shared device-selector field every per-tool schema embeds.
var deviceProp = map[string]any{
	"type":        "string",
	"description": "the target device serial; omit if exactly one device is connected",
}

// schemaWithMatcher builds a top-level schema with `device`, `match`, and any
// caller-supplied extra properties. The matcher recursion is exposed via $defs.
func schemaWithMatcher(extras map[string]any, requiredExtras []string) map[string]any {
	props := map[string]any{
		"device": deviceProp,
		"match":  map[string]any{"$ref": "#/$defs/matcher"},
	}
	for k, v := range extras {
		props[k] = v
	}
	required := append([]string{"match"}, requiredExtras...)
	return map[string]any{
		"type": "object",
		"$defs": map[string]any{
			"matcher": matcherSchemaDef,
		},
		"properties":           props,
		"required":             required,
		"additionalProperties": false,
	}
}

// schemaDeviceOnly is for tools that take a device but no matcher (Espresso
// conveniences and intent monitor lifecycle).
func schemaDeviceOnly(extras map[string]any, requiredExtras []string) map[string]any {
	props := map[string]any{"device": deviceProp}
	for k, v := range extras {
		props[k] = v
	}
	schema := map[string]any{
		"type":                 "object",
		"properties":           props,
		"additionalProperties": false,
	}
	// Only emit `required` when non-empty: a nil []string marshals to JSON
	// null, which violates JSON Schema (required must be an array) and
	// makes strict MCP clients reject the whole tools/list response.
	if len(requiredExtras) > 0 {
		schema["required"] = requiredExtras
	}
	return schema
}
