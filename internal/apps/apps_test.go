package apps

import (
	"strings"
	"testing"
)

func TestParseLauncherActivities(t *testing.T) {
	out := `
  Activity #0:
    priority=0 preferredOrder=0 match=0x108000 specificIndex=-1 isDefault=false
    ActivityInfo:
      name=com.android.settings.Settings
      packageName=com.android.settings
      MainActivity: com.android.settings/.Settings
  Activity #1:
    ActivityInfo:
      name=com.example.Foo
      packageName=com.example.app
      MainActivity: com.example.app/.MainActivity
`
	apps := parseLauncherActivities(out)
	if len(apps) < 2 {
		t.Fatalf("got %d apps, want >=2: %+v", len(apps), apps)
	}
	want := map[string]bool{
		"com.android.settings": false,
		"com.example.app":      false,
	}
	for _, a := range apps {
		if _, ok := want[a.Package]; ok {
			want[a.Package] = true
		}
	}
	for k, ok := range want {
		if !ok {
			t.Errorf("missing %s in %+v", k, apps)
		}
	}
}

func TestParsePackageInfo(t *testing.T) {
	out := `
Packages:
  Package [com.example.app] (abc123):
    versionName=1.2.3
    versionCode=42 minSdk=21 targetSdk=33
    firstInstallTime=2024-01-01 10:00:00
    lastUpdateTime=2024-06-01 12:00:00
    dataDir=/data/user/0/com.example.app
    nativeLibraryDir=/data/app/com.example.app/lib/arm64
    installerPackageName=com.android.vending
    requested permissions:
      android.permission.INTERNET
      android.permission.CAMERA
      android.permission.ACCESS_FINE_LOCATION
    install permissions:
      android.permission.INTERNET: granted=true
    runtime permissions:
      android.permission.CAMERA: granted=true
      android.permission.ACCESS_FINE_LOCATION: granted=false
`
	info, err := parsePackageInfo("com.example.app", out)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if info.VersionName != "1.2.3" {
		t.Errorf("version: %q", info.VersionName)
	}
	if info.VersionCode != "42" {
		t.Errorf("versionCode: %q", info.VersionCode)
	}
	if info.TargetSdk != "33" || info.MinSdk != "21" {
		t.Errorf("sdk: target=%q min=%q", info.TargetSdk, info.MinSdk)
	}
	if info.DataDir != "/data/user/0/com.example.app" {
		t.Errorf("dataDir: %q", info.DataDir)
	}
	if !contains(info.Requested, "android.permission.CAMERA") {
		t.Errorf("requested missing CAMERA: %v", info.Requested)
	}
	if !contains(info.Granted, "android.permission.INTERNET") {
		t.Errorf("granted missing INTERNET: %v", info.Granted)
	}
	if contains(info.Granted, "android.permission.ACCESS_FINE_LOCATION") {
		t.Errorf("ACCESS_FINE_LOCATION should be filtered out (granted=false): %v", info.Granted)
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func TestIsAmbiguousActivityOutput(t *testing.T) {
	ambiguous := `App loaded: com.example.app
Debuggable: true
Multiple candidates found for type ACTIVITY:
  - com.example.app.OtherActivity
  - com.example.app.MainActivity
Please specify component using --activity <name>`
	if !isAmbiguousActivityOutput(ambiguous) {
		t.Errorf("expected ambiguous-activity output to be detected")
	}

	clean := `App loaded: com.example.app
Debuggable: true
Activity started: com.example.app/.MainActivity`
	if isAmbiguousActivityOutput(clean) {
		t.Errorf("expected clean install+launch output not to be flagged")
	}
}

func TestInterpretRunResult(t *testing.T) {
	// All captured live against android CLI v1.0.15985488 / a real emulator.
	const success = `Standard Install: Sent 11.35 MB (1 APK)
Launching com.example.installprobe.MainActivity
Activation completed successfully`

	if out, ok := interpretRunResult(success, ""); !ok || out != success {
		t.Errorf("clean success: got (%q, %v), want (%q, true)", out, ok, success)
	}

	// `android run` exits 0 even when the APK doesn't exist / fails to
	// parse, reporting the real error on stderr with an empty stdout.
	const missingApkErr = `Error: Error during check: java.io.FileNotFoundException: /tmp/nonexistent.apk (No such file or directory)
Component check failed`
	if out, ok := interpretRunResult("", missingApkErr); ok {
		t.Errorf("missing-apk failure (empty stdout, error on stderr) should not be treated as success, got %q", out)
	}

	// Observed case: stdout gets a success-looking "Sent ... MB" line, but
	// a later deploy stage fails and only stderr carries the real error.
	const partialStdout = "Standard Install: Sent 11.35 MB (1 APK)"
	const laterStageErr = "Installation failed. Code: UNKNOWN. Details: ..."
	if out, ok := interpretRunResult(partialStdout, laterStageErr); ok {
		t.Errorf("stdout with success-looking text but non-empty stderr should not be treated as success, got %q", out)
	}

	// Ambiguous-activity disambiguation prompt must still fall through
	// even though it's non-empty stdout with empty stderr.
	const ambiguous = "Multiple candidates found for type ACTIVITY:\n  - a\n  - b"
	if out, ok := interpretRunResult(ambiguous, ""); ok {
		t.Errorf("ambiguous-activity output should not be treated as success, got %q", out)
	}

	if out, ok := interpretRunResult("", ""); ok {
		t.Errorf("empty stdout and stderr should not be treated as success, got %q", out)
	}
}

func TestSafeRelPath(t *testing.T) {
	good := map[string]string{
		"":                          ".",
		"shared_prefs":              "shared_prefs",
		"shared_prefs/settings.xml": "shared_prefs/settings.xml",
		"./databases/foo":           "databases/foo",
	}
	for in, want := range good {
		got, err := SafeRelPath(in)
		if err != nil {
			t.Errorf("SafeRelPath(%q) error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("SafeRelPath(%q) = %q, want %q", in, got, want)
		}
	}
	bad := []string{
		"/etc/passwd",
		"../etc",
		"shared_prefs/../../etc",
		"a/b/$(rm)",
		"a`whoami`",
		"a\nb",
		"a\"b",
	}
	for _, p := range bad {
		if _, err := SafeRelPath(p); err == nil {
			t.Errorf("expected error for %q", p)
		}
	}
	// sanity: no surprising stripping of valid leading components
	if got, _ := SafeRelPath("a/b/c"); !strings.HasPrefix(got, "a/") {
		t.Errorf("unexpected normalisation: %q", got)
	}
}
