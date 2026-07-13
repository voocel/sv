package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ghRelease is the subset of the GitHub release API sv cares about.
type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// selfUpgrade replaces the running sv binary with the latest release.
func (a *App) selfUpgrade(ctx context.Context, force bool) error {
	infof("checking for updates...")

	release, err := a.fetchLatestSelf(ctx)
	if err != nil {
		return err
	}
	if !force && compareTags(version, release.TagName) >= 0 {
		successf("sv is already up to date (%s)", version)
		return nil
	}

	name := fmt.Sprintf("sv-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	var asset *ghAsset
	for i := range release.Assets {
		if release.Assets[i].Name == name {
			asset = &release.Assets[i]
			break
		}
	}
	if asset == nil {
		return fmt.Errorf("release %s has no binary for %s/%s", release.TagName, runtime.GOOS, runtime.GOARCH)
	}

	infof("upgrading to %s (%s)", release.TagName, asset.Name)
	if err := a.replaceSelf(ctx, asset); err != nil {
		return err
	}
	successf("upgraded to %s", release.TagName)
	return nil
}

func (a *App) fetchLatestSelf(ctx context.Context) (*ghRelease, error) {
	var release ghRelease
	if err := a.getJSON(ctx, a.cfg.UpgradeAPIURL, &release); err != nil {
		return nil, fmt.Errorf("check for updates: %w", err)
	}
	return &release, nil
}

func (a *App) replaceSelf(ctx context.Context, asset *ghAsset) error {
	tmp := filepath.Join(a.paths.Bin, ".sv.tmp")
	dl := newDownloader(a.client)
	if err := dl.fetchSingle(ctx, asset.BrowserDownloadURL, tmp, asset.Size, cyan("sv[upgrade]")); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("download %s: %w", asset.Name, err)
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		os.Remove(tmp)
		return err
	}

	target := filepath.Join(a.paths.Bin, "sv")
	if runtime.GOOS == "windows" {
		// Windows cannot overwrite a running executable, but it can be renamed.
		target += ".exe"
		old := target + ".old"
		os.Remove(old)
		os.Rename(target, old)
	}
	if err := os.Rename(tmp, target); err != nil {
		return fmt.Errorf("install new binary: %w", err)
	}
	return nil
}

// selfUninstall removes ~/.sv entirely along with shell/env configuration.
func (a *App) selfUninstall() error {
	if !strings.HasSuffix(a.paths.Home, ".sv") {
		return errors.New("unexpected sv home directory, refusing to uninstall")
	}

	warnf("This removes sv itself and every installed Go version.")
	warnf("Directory to be removed: %s", a.paths.Home)
	errorf("This action cannot be undone!")

	answer, err := inputPrompt("Type 'yes' to confirm uninstall:")
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(answer), "yes") {
		infof("uninstall cancelled")
		return nil
	}

	if runtime.GOOS == "windows" {
		a.cleanWindowsEnv()
		a.moveSelfAside()
	} else {
		a.cleanShellProfiles()
	}

	if err := os.RemoveAll(a.paths.Home); err != nil {
		return fmt.Errorf("remove %s: %w", a.paths.Home, err)
	}

	successf("sv has been uninstalled")
	if runtime.GOOS == "windows" {
		infof("open a new terminal for changes to take effect")
	} else {
		infof("restart your terminal or run: source ~/.bashrc (or ~/.zshrc)")
	}
	return nil
}

// moveSelfAside renames the running sv.exe out of ~/.sv so RemoveAll can
// delete the directory — Windows cannot delete a running executable. The
// destination stays next to ~/.sv because os.Rename cannot cross volumes.
func (a *App) moveSelfAside() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	if !strings.HasPrefix(strings.ToLower(exe), strings.ToLower(a.paths.Home)) {
		return
	}
	dest := filepath.Join(filepath.Dir(a.paths.Home), ".sv-uninstalled.exe.old")
	os.Remove(dest)
	if err := os.Rename(exe, dest); err == nil {
		warnf("leftover binary %s can be deleted after this terminal closes", dest)
	}
}

// cleanShellProfiles removes the line the installer added to shell rc files.
func (a *App) cleanShellProfiles() {
	home, err := os.UserHomeDir()
	if err != nil {
		warnf("failed to resolve home directory: %v", err)
		return
	}

	const envLine = `. "$HOME/.sv/env"`
	cleaned := false
	for _, rc := range []string{".bashrc", ".bash_profile", ".zshrc", ".profile"} {
		if removeLineFromFile(filepath.Join(home, rc), envLine) {
			cleaned = true
		}
	}

	if cleaned {
		successf("cleaned shell profile")
	} else {
		infof("no sv configuration found in shell profiles")
	}
}

// removeLineFromFile removes the exact line (and the installer comment
// right above it, if present). Reports whether anything was removed.
func removeLineFromFile(path, line string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	lines := strings.Split(string(content), "\n")
	var out []string
	removed := false

	for _, l := range lines {
		if strings.TrimSpace(l) != line {
			out = append(out, l)
			continue
		}
		removed = true
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "# Added by sv installer" {
			out = out[:len(out)-1]
		}
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
			out = out[:len(out)-1]
		}
	}
	if !removed {
		return false
	}

	text := strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n"
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		warnf("failed to update %s: %v", path, err)
		return false
	}
	return true
}

// cleanWindowsEnv removes sv's PATH entry and GOROOT from the user scope.
func (a *App) cleanWindowsEnv() {
	infof("cleaning Windows environment variables...")

	script := fmt.Sprintf(`
$binDir = '%s'
$goRoot = '%s'
$goBinDir = "$goRoot\bin"

$userPath = [System.Environment]::GetEnvironmentVariable('PATH', 'User')
if ($userPath) {
    $paths = $userPath -split ';' | Where-Object { $_ -ne $binDir -and $_ -ne $goBinDir -and $_ -ne '' }
    [System.Environment]::SetEnvironmentVariable('PATH', ($paths -join ';'), 'User')
}

$currentGoRoot = [System.Environment]::GetEnvironmentVariable('GOROOT', 'User')
if ($currentGoRoot -eq $goRoot) {
    [System.Environment]::SetEnvironmentVariable('GOROOT', $null, 'User')
}
`, a.paths.Bin, a.paths.Root)

	if err := exec.Command("powershell", "-NoProfile", "-Command", script).Run(); err != nil {
		warnf("failed to clean environment variables: %v", err)
		warnf("please remove '%s' from PATH and GOROOT manually", a.paths.Bin)
		return
	}
	successf("cleaned Windows environment variables")
}
