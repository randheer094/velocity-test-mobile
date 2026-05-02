package apps

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/randheer094/velocity-mcp-mobile/internal/adb"
)

// PackageInfo is a structured subset of `dumpsys package <pkg>`.
type PackageInfo struct {
	Package         string   `json:"package"`
	VersionName     string   `json:"versionName,omitempty"`
	VersionCode     string   `json:"versionCode,omitempty"`
	TargetSdk       string   `json:"targetSdk,omitempty"`
	MinSdk          string   `json:"minSdk,omitempty"`
	FirstInstall    string   `json:"firstInstallTime,omitempty"`
	LastUpdate      string   `json:"lastUpdateTime,omitempty"`
	DataDir         string   `json:"dataDir,omitempty"`
	NativeLibDir    string   `json:"nativeLibraryDir,omitempty"`
	Requested       []string `json:"requestedPermissions"`
	Granted         []string `json:"grantedPermissions"`
	Signers         []string `json:"signers,omitempty"`
	InstallerSource string   `json:"installerPackage,omitempty"`
}

// Info parses `dumpsys package <pkg>`.
func (c *Client) Info(ctx context.Context, deviceID, pkg string) (PackageInfo, error) {
	if _, err := adb.MustQuotePackage(pkg); err != nil {
		return PackageInfo{}, err
	}
	res, err := c.Adb.ShellArgv(ctx, deviceID, "dumpsys", "package", pkg)
	if err != nil {
		return PackageInfo{}, err
	}
	return parsePackageInfo(pkg, string(res.Stdout))
}

func parsePackageInfo(pkg, out string) (PackageInfo, error) {
	info := PackageInfo{Package: pkg}
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	section := ""
	requestedSet := make(map[string]struct{})
	grantedSet := make(map[string]struct{})

	for sc.Scan() {
		raw := sc.Text()
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "requested permissions:"):
			section = "requested"
			continue
		case strings.HasPrefix(line, "install permissions:") ||
			strings.HasPrefix(line, "runtime permissions:"):
			section = "granted"
			continue
		case strings.HasPrefix(line, "User ") && strings.Contains(line, "runtime permissions:"):
			section = "granted"
			continue
		case strings.HasPrefix(line, "Activity Resolver Table:"),
			strings.HasPrefix(line, "Receiver Resolver Table:"),
			strings.HasPrefix(line, "Service Resolver Table:"):
			section = ""
			continue
		}

		switch section {
		case "requested":
			if line != "" && !strings.Contains(line, ":") {
				requestedSet[line] = struct{}{}
				continue
			}
			// trailing colon variants like "android.permission.CAMERA: granted=true"
			if strings.HasPrefix(line, "android.permission.") || strings.HasPrefix(line, "com.") {
				name := line
				if i := strings.Index(name, ":"); i >= 0 {
					name = name[:i]
				}
				requestedSet[name] = struct{}{}
				continue
			}
			if line == "" {
				section = ""
			}
		case "granted":
			if strings.HasPrefix(line, "android.permission.") || strings.HasPrefix(line, "com.") {
				name := line
				if i := strings.Index(name, ":"); i >= 0 {
					name = name[:i]
				}
				if strings.Contains(line, "granted=true") || !strings.Contains(line, "granted=") {
					grantedSet[name] = struct{}{}
				}
				continue
			}
			if line == "" {
				section = ""
			}
		}

		// `versionCode=42 minSdk=21 targetSdk=33` packs multiple key=value
		// pairs on one line; check each whitespace-separated token.
		for _, tok := range strings.Fields(line) {
			eq := strings.IndexByte(tok, '=')
			if eq <= 0 {
				continue
			}
			k, v := tok[:eq], tok[eq+1:]
			switch k {
			case "versionCode":
				info.VersionCode = v
			case "minSdk":
				info.MinSdk = v
			case "targetSdk":
				info.TargetSdk = v
			}
		}

		switch {
		case strings.HasPrefix(line, "versionName="):
			info.VersionName = strings.TrimPrefix(line, "versionName=")
		case strings.HasPrefix(line, "firstInstallTime="):
			info.FirstInstall = strings.TrimPrefix(line, "firstInstallTime=")
		case strings.HasPrefix(line, "lastUpdateTime="):
			info.LastUpdate = strings.TrimPrefix(line, "lastUpdateTime=")
		case strings.HasPrefix(line, "dataDir="):
			info.DataDir = strings.TrimPrefix(line, "dataDir=")
		case strings.HasPrefix(line, "nativeLibraryDir="):
			info.NativeLibDir = strings.TrimPrefix(line, "nativeLibraryDir=")
		case strings.HasPrefix(line, "installerPackageName="):
			info.InstallerSource = strings.TrimPrefix(line, "installerPackageName=")
		case strings.HasPrefix(line, "Signing KeySets:") ||
			strings.HasPrefix(line, "PackageSignatures{"):
			info.Signers = append(info.Signers, line)
		}
	}

	if err := sc.Err(); err != nil {
		return info, fmt.Errorf("scanning dumpsys output: %w", err)
	}
	info.Requested = sortedKeys(requestedSet)
	info.Granted = sortedKeys(grantedSet)
	return info, nil
}

func sortedKeys(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// sort.Strings without dragging in another import path on this small list:
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
