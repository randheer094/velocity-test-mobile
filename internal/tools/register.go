package tools

import "github.com/modelcontextprotocol/go-sdk/mcp"

// RegisterAll wires every testing-related tool surface onto the server.
//
// This server is testing-only: it exposes Espresso- and Compose-test-style
// verbs plus the minimum supporting infrastructure (animations, app launch,
// permissions, intents, layout, screenshot, logcat, clipboard).
func RegisterAll(s *mcp.Server, d *Deps) {
	RegisterDevice(s, d)
	RegisterApp(s, d)
	RegisterUI(s, d)
	RegisterInput(s, d)
	RegisterDiagnostics(s, d)
	RegisterSystem(s, d)
	RegisterTesting(s, d)
}

// Catalog returns a static list of tool names registered by RegisterAll —
// kept up to date by hand so `--list-tools` produces a documentation-quality
// catalogue. If you add or remove a tool, update this list.
func Catalog() []string {
	return []string{
		// device
		"device_list", "device_get_screen_size", "device_get_orientation",
		"device_set_orientation", "device_get_props",

		// app lifecycle / state / verification
		"app_list", "app_launch", "app_terminate", "app_clear_data", "app_get_info",
		"permission_grant", "permission_revoke",
		"appops_set", "appops_get",
		"intent_send",
		"app_data_list", "app_data_read",

		// screen capture & visual regression
		"screen_capture", "screen_layout", "screen_resolve", "screen_diff",

		// input utilities (semantic verbs live in the testing surface)
		"clipboard_get", "clipboard_set", "press_key", "type_into_focused",

		// logs (test debug)
		"logcat_tail", "logcat_clear",

		// system state required by Espresso/Compose tests
		"animations_set", "animations_get",

		// activity / service / location / notification / shell introspection
		"activity_get_top", "activity_wait_for_top", "activity_start",
		"service_get_state", "service_wait_for_state",
		"location_get_last_known",
		"notification_list", "notification_shade_set", "notification_tap",
		"shell_exec",

		// testing — finders / debug
		"find_node", "find_all_nodes", "count_nodes", "print_tree",

		// testing — assertions
		"assert_visible", "assert_not_visible",
		"assert_completely_displayed", "assert_displaying_at_least",
		"assert_exists", "assert_does_not_exist",
		"assert_clickable", "assert_enabled", "assert_disabled",
		"assert_focused", "assert_selected", "assert_checked", "assert_unchecked",
		"assert_on", "assert_off", "assert_toggleable",
		"assert_text_equals", "assert_text_contains",
		"assert_content_description_equals",
		"assert_count_equals", "assert_has_descendant",
		"assert_width_dp", "assert_height_dp",
		"assert_width_at_least_dp", "assert_height_at_least_dp",
		"assert_position_in_root",
		"assert_any", "assert_all",
		"assert_is_root",
		"assert_has_child_count", "assert_has_minimum_child_count",

		// testing — actions
		"click", "double_click", "long_click",
		"type_text", "replace_text", "clear_text", "submit_text",
		"swipe_node", "slow_swipe_node", "scroll_to", "scroll_to_index",
		"perform_ime_action", "perform_key_press",
		"assert_clickable_and_click",

		// testing — synchronization
		"wait_until_visible", "wait_until_not_visible",
		"wait_until_text", "wait_until_count",
		"wait_until_at_least_one_exists", "wait_for_idle",

		// testing — Espresso conveniences
		"espresso_press_back", "press_back_unconditionally",
		"close_soft_keyboard",
		"open_overflow_menu", "open_contextual_action_mode_menu",

		// testing — Espresso-Intents (recording-only; stubbing not supported externally)
		"intent_monitor_start", "intent_monitor_stop",
		"intent_list_captured",
		"assert_intent_sent", "assert_intent_count",
	}
}
