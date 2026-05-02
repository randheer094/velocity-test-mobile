package tools

import "github.com/modelcontextprotocol/go-sdk/mcp"

// RegisterAll wires every tool surface onto the server.
func RegisterAll(s *mcp.Server, d *Deps) {
	RegisterDevice(s, d)
	RegisterEmulator(s, d)
	RegisterApp(s, d)
	RegisterUI(s, d)
	RegisterInput(s, d)
	RegisterDiagnostics(s, d)
	RegisterRecording(s, d)
	RegisterFiles(s, d)
	RegisterSystem(s, d)
	RegisterMaintenance(s, d)
	RegisterDocs(s, d)
}

// Catalog returns a static list of tool names registered by RegisterAll.
// Used by the --list-tools CLI flag for documentation.
func Catalog() []string {
	return []string{
		// device
		"device_list", "device_get_screen_size", "device_get_orientation", "device_set_orientation", "device_get_props",
		// emulator
		"emulator_list", "emulator_start", "emulator_stop",
		// apps
		"app_list", "app_install", "app_uninstall", "app_launch", "app_terminate",
		"app_clear_data", "app_get_info", "permission_grant", "permission_revoke",
		"intent_send", "app_data_list", "app_data_read",
		// ui
		"screen_capture", "screen_layout", "screen_resolve", "wait_for_element", "assert_text_visible", "screen_diff",
		// input
		"tap", "double_tap", "long_press", "swipe", "fling", "drag", "type_keys", "press_button",
		"clipboard_get", "clipboard_set",
		// diagnostics
		"logcat_tail", "logcat_clear", "dumpsys_meminfo", "dumpsys_gfxinfo", "dumpsys_battery", "dumpsys_activity",
		"atrace_capture", "perfetto_capture",
		// recording
		"screen_record_start", "screen_record_stop",
		// files
		"file_push", "file_pull",
		// system
		"screen_wake", "screen_lock", "animations_set", "animations_get", "doze_simulate",
		"time_set_timezone", "network_set_airplane", "network_set_wifi", "network_set_mobile_data",
		"location_set",
		// maintenance
		"device_reboot", "wireless_enable", "wireless_connect", "wireless_pair", "wireless_disconnect",
		// docs
		"docs_search", "docs_fetch",
	}
}
