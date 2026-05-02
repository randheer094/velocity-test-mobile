# velocity-mcp-mobile

An **Android-only** [Model Context Protocol](https://modelcontextprotocol.io) server, written in Go, that gives an LLM agent everything it needs to drive, observe, and verify an Android app on a device or emulator.

Inspired by [`mobile-next/mobile-mcp`](https://github.com/mobile-next/mobile-mcp). Differences:

- **Android-only** — no iOS / WebDriverAgent.
- **No telemetry**, no phone-home — fully local.
- **Single static Go binary**, sub-10ms cold start, no Node/Python runtime.
- Prefers Google's new agent CLI ([`android`](https://developer.android.com/tools/agents/android-cli)) where it offers something cleaner; falls back to `adb` everywhere else.
- Wider verification surface — pixel diffs, perfetto/atrace, animations control, doze simulation, network/airplane/Wi-Fi/data toggles, time/timezone, mock GPS, app-data inspection (run-as), wireless ADB, and more.

## Runtime requirements

| Tool | Required? | Used for |
| --- | --- | --- |
| `adb` | **always** | every interaction with the device |
| `android` ([agent CLI](https://developer.android.com/tools/agents/android-cli)) | recommended | `screen_resolve`, `screen_layout` (preferred path), `screen_capture` (preferred path), `app_install` (preferred path), `emulator_*`, `docs_*` |

Both must be on `PATH`. If `android` is missing, the server logs a single warning and the affected tools return an actionable error pointing at the install page; everything else still works through plain `adb`.

## Install

```bash
git clone https://github.com/randheer094/velocity-mcp-mobile.git
cd velocity-mcp-mobile
go install .            # or: make build
```

Or build from a checkout:

```bash
make build
./velocity-mcp-mobile --version
./velocity-mcp-mobile --list-tools | wc -l
```

## Hooking up to a client

### Claude Desktop / Claude Code

```jsonc
// ~/Library/Application Support/Claude/claude_desktop_config.json (macOS)
{
  "mcpServers": {
    "android": {
      "command": "/absolute/path/to/velocity-mcp-mobile"
    }
  }
}
```

### MCP Inspector (interactive)

```bash
npx @modelcontextprotocol/inspector ./velocity-mcp-mobile
```

## Tool surface

The server exposes ~65 tools. Use `--list-tools` to print the canonical names. Highlights:

| Category | Tools |
| --- | --- |
| Device | `device_list`, `device_get_screen_size`, `device_get_props`, `device_get_orientation`, `device_set_orientation` |
| Emulator | `emulator_list`, `emulator_start`, `emulator_stop` |
| Apps | `app_list`, `app_install`, `app_uninstall`, `app_launch`, `app_terminate`, `app_clear_data`, `app_get_info`, `permission_grant`, `permission_revoke`, `intent_send`, `app_data_list`, `app_data_read` |
| UI capture | `screen_capture`, `screen_layout`, `screen_resolve` |
| UI input | `tap`, `double_tap`, `long_press`, `swipe`, `drag`, `fling`, `type_keys`, `press_button` |
| Clipboard | `clipboard_get`, `clipboard_set` |
| Test asserts | `wait_for_element`, `assert_text_visible`, `screen_diff` |
| Diagnostics | `logcat_tail`, `logcat_clear`, `dumpsys_meminfo`, `dumpsys_gfxinfo`, `dumpsys_battery`, `dumpsys_activity` |
| Tracing | `atrace_capture`, `perfetto_capture` |
| Recording | `screen_record_start`, `screen_record_stop` |
| Files | `file_push`, `file_pull` |
| System state | `screen_wake`, `screen_lock`, `animations_set`, `animations_get`, `doze_simulate`, `time_set_timezone`, `network_set_airplane`, `network_set_wifi`, `network_set_mobile_data`, `location_set` |
| Maintenance | `device_reboot`, `wireless_enable`, `wireless_connect`, `wireless_pair`, `wireless_disconnect` |
| Knowledge | `docs_search`, `docs_fetch` |

### Designed for stable UI tests

`animations_set 0` plus `wait_for_element` is the recommended way to drive flake-free UI flows. Pair `screen_capture` with `screen_diff` for visual regression. `screen_layout` returns a flat list of interactive nodes with bounds so an agent can click by intent rather than blind coordinates; `screen_resolve` (Android CLI) is even more direct for label-based clicks.

## Notes & caveats

- `app_data_list` / `app_data_read` use `run-as` and therefore only work on **debuggable** builds. Release builds will return an actionable error.
- `network_set_wifi` / `network_set_mobile_data` rely on the `svc` helper; on some OEM builds `svc` requires root.
- `location_set` uses `adb emu geo fix` on emulators; on physical devices it cannot truly mock a location without a developer-options mock-location-provider app, so it returns a `device` mode result describing what's required.
- `device_reboot` is destructive — it requires `confirm: true`.
- `screenrecord` runs on-device and the file is pulled to host on stop. Long sessions are bounded by Android's own `screenrecord` ceiling (typically 180s) per chunk.
- `perfetto` requires API 28+ and a working `tracebox` on most devices.

## Development

```bash
make vet      # go vet ./...
make test     # go test ./...
make build    # static binary
make lint     # vet + gofmt
```

The entire codebase is `internal/` packages plus a thin `main.go` and a `tools/` directory of MCP handlers, one file per surface. Every subprocess goes through `internal/runner` so timeouts, output caps, and error wrapping are uniform.

## License

Source provided as-is for the user's project; choose a license at the repository level.
