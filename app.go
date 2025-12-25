package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/urfave/cli/v2"
)

type app struct {
	ctx    *cli.Context
	client *http.Client
}

func newApp(ctx *cli.Context) *app {
	return &app{
		ctx:    ctx,
		client: &http.Client{Timeout: cfg.HTTPTimeout},
	}
}

func (a *app) Run() error {
	// Handle subcommands by checking lineage
	cmdName := a.ctx.Command.Name
	// Check if this is a subcommand under "self" by looking at Lineage
	if len(a.ctx.Lineage()) > 2 {
		// Lineage: [current context, self context, app context]
		cmdName = "self " + cmdName
	}

	switch cmdName {
	case "list":
		return a.handleList()
	case "use":
		return a.handleUse()
	case "install":
		return a.handleInstall()
	case "uninstall":
		return a.handleUninstall()
	case "prune":
		return a.handlePrune()
	case "current":
		return a.handleCurrent()
	case "where":
		return a.handleWhere()
	case "latest":
		return a.handleLatest()
	case "outdated":
		return a.handleOutdated()
	case "self upgrade":
		return a.handleUpgrade()
	case "self uninstall":
		return a.handleSelfUninstall()
	default:
		return ErrUnsupportedCommand()
	}
}

func (a *app) handleList() error {
	if a.ctx.Bool("remote") {
		return a.listRemote()
	}
	return a.listLocal()
}

func (a *app) handleUse() error {
	target := a.ctx.Args().First()
	if target == "" {
		return ErrTagEmpty()
	}

	tag := normalizeVersionTag(target)
	p := &Package{
		Tag:  tag,
		Name: generateFileName(tag),
	}

	if err := p.useLocal(); err != nil {
		return a.promptRemoteInstall(tag)
	}
	return nil
}

func (a *app) handleInstall() error {
	releases, err := FetchReleases(a.client, true)
	if err != nil {
		return err
	}

	var tag string
	if a.ctx.Bool("latest") {
		for _, r := range releases {
			if r.Stable {
				tag = r.Version
				break
			}
		}
		if tag == "" {
			return NewError("no stable version found")
		}
	} else {
		target := a.ctx.Args().First()
		if target == "" {
			return ErrTagEmpty()
		}
		tag = normalizeVersionTag(target)
	}

	release := FindRelease(releases, tag)
	if release == nil {
		return NewError("version not found: " + tag)
	}

	file := release.FindMatchingFile()
	if file == nil {
		return NewError(fmt.Sprintf("no package found for %s/%s", runtime.GOOS, runtime.GOARCH))
	}

	return file.ToPackage(release.Version).install()
}

func (a *app) handleUninstall() error {
	target := a.ctx.Args().First()
	if target == "" {
		return ErrTagEmpty()
	}

	tag := normalizeVersionTag(target)
	p := &Package{
		Tag:  tag,
		Name: generateFileName(tag),
	}
	return p.remove()
}

func (a *app) handleUpgrade() error {
	u := NewUpgrade(a.ctx.Bool("force"))
	return u.checkUpgrade()
}

func (a *app) promptRemoteInstall(tag string) error {
	var ok bool
	err := survey.AskOne(&survey.Confirm{
		Message: "Version not found locally. Download and install from remote?",
	}, &ok)
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

	releases, err := FetchReleases(a.client, true)
	if err != nil {
		return err
	}

	release := FindRelease(releases, tag)
	if release == nil {
		return NewError("version not found: " + tag)
	}

	file := release.FindMatchingFile()
	if file == nil {
		return NewError(fmt.Sprintf("no package found for %s/%s", runtime.GOOS, runtime.GOARCH))
	}

	return file.ToPackage(release.Version).useRemote()
}

func (a *app) listRemote() error {
	releases, err := FetchReleases(a.client, true)
	if err != nil {
		return err
	}

	if len(releases) == 0 {
		return NewError("no versions found")
	}

	versions := GetVersionList(releases)
	target, err := a.selectVersions(versions)
	if err != nil {
		return err
	}

	release := FindRelease(releases, target)
	if release == nil {
		return NewError("version not found: " + target)
	}

	file := release.FindMatchingFile()
	if file == nil {
		return NewError(fmt.Sprintf("no package found for %s/%s", runtime.GOOS, runtime.GOARCH))
	}

	return file.ToPackage(release.Version).use()
}

func (a *app) listLocal() error {
	pkg := &Package{}
	versions, err := pkg.getLocalVersion()
	if err != nil {
		return err
	}

	target, err := a.selectVersions(versions)
	if err != nil {
		return err
	}

	pkg.Tag = target
	pkg.Name = generateFileName(target)
	return pkg.useLocal()
}

func (a *app) selectVersions(versions []string) (string, error) {
	if len(versions) == 0 {
		return "", ErrNoVersionsAvailable()
	}

	sort.Slice(versions, func(i, j int) bool {
		return versionCompare(versions[i]) > versionCompare(versions[j])
	})

	var target string
	err := survey.AskOne(&survey.Select{
		Message: "Choose a version:",
		Help:    "Enter to install the selected version",
		Options: versions,
	}, &target, survey.WithValidator(survey.Required), surveyIcon())
	if err != nil {
		return "", err
	}

	// Remove any suffix like " (date)"
	if i := strings.Index(target, "("); i != -1 {
		target = strings.TrimSpace(target[:i])
	}

	return target, nil
}

func surveyIcon() survey.AskOpt {
	return survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Text = "→"
	})
}

func (a *app) handlePrune() error {
	pkg := &Package{}
	versions, err := pkg.getLocalVersion()
	if err != nil {
		return err
	}

	if len(versions) == 0 {
		return NewInfo("no installed versions to prune")
	}

	sort.Slice(versions, func(i, j int) bool {
		return versionCompare(versions[i]) > versionCompare(versions[j])
	})

	currentVersion := getCurrentVersion()
	keep := a.ctx.Int("keep")
	if keep < 1 {
		keep = 2
	}

	var toRemove, toKeep []string
	kept := 0

	for _, v := range versions {
		isCurrent := v == currentVersion

		if a.ctx.Bool("all") {
			if isCurrent {
				toKeep = append(toKeep, v)
			} else {
				toRemove = append(toRemove, v)
			}
		} else {
			if isCurrent {
				toKeep = append(toKeep, v)
			} else if kept < keep {
				toKeep = append(toKeep, v)
				kept++
			} else {
				toRemove = append(toRemove, v)
			}
		}
	}

	if len(toRemove) == 0 {
		PrintGreen("Nothing to prune. All versions are within the keep limit.")
		return nil
	}

	PrintCyan(fmt.Sprintf("Installed versions: %d", len(versions)))
	if currentVersion != "" {
		PrintCyan(fmt.Sprintf("Current version: %s", currentVersion))
	}
	PrintCyan(fmt.Sprintf("Versions to keep: %v", toKeep))
	PrintYellow(fmt.Sprintf("Versions to remove: %v", toRemove))

	if a.ctx.Bool("dry-run") {
		PrintBlue("Dry run mode - no changes made")
		return nil
	}

	var confirm bool
	err = survey.AskOne(&survey.Confirm{
		Message: fmt.Sprintf("Remove %d version(s)?", len(toRemove)),
		Default: false,
	}, &confirm)
	if err != nil {
		return err
	}

	if !confirm {
		PrintBlue("Prune cancelled")
		return nil
	}

	removed := 0
	for _, v := range toRemove {
		p := &Package{Tag: v, Name: generateFileName(v)}
		if err := p.removeLocal(); err != nil {
			Warnf("Failed to remove %s: %v", v, err)
			continue
		}
		PrintGreen(fmt.Sprintf("Removed: %s", v))
		removed++
	}

	PrintGreen(fmt.Sprintf("Pruned %d version(s), kept %d version(s)", removed, len(toKeep)))
	return nil
}

func (a *app) handleCurrent() error {
	current := getCurrentVersion()
	if current == "" {
		return NewInfo("no Go version is currently active")
	}
	fmt.Println(current)
	return nil
}

func (a *app) handleWhere() error {
	target := a.ctx.Args().First()
	if target == "" {
		current := getCurrentVersion()
		if current == "" {
			return NewInfo("no Go version is currently active, specify a version: sv where <version>")
		}
		target = current
	}

	tag := normalizeVersionTag(target)
	versionPath := filepath.Join(paths.Cache, tag)

	if !Exists(versionPath) {
		return NewError(fmt.Sprintf("version %s is not installed", target))
	}

	fmt.Println(versionPath)
	return nil
}

func (a *app) handleLatest() error {
	latest, err := GetLatestVersion(a.client)
	if err != nil {
		return err
	}
	fmt.Println(latest)
	return nil
}

func (a *app) handleOutdated() error {
	pkg := &Package{}
	localVersions, err := pkg.getLocalVersion()
	if err != nil {
		return err
	}

	if len(localVersions) == 0 {
		return NewInfo("no installed versions")
	}

	latest, err := GetLatestVersion(a.client)
	if err != nil {
		return err
	}

	current := getCurrentVersion()

	sort.Slice(localVersions, func(i, j int) bool {
		return versionCompare(localVersions[i]) > versionCompare(localVersions[j])
	})

	PrintCyan(fmt.Sprintf("Latest available: %s", latest))
	if current != "" {
		PrintCyan(fmt.Sprintf("Current active:   %s", current))
	}
	fmt.Println()

	hasOutdated := false
	for _, v := range localVersions {
		marker := "  "
		if v == current {
			marker = "* "
		}

		if versionCompare(v) < versionCompare(latest) {
			hasOutdated = true
			PrintYellow(fmt.Sprintf("%s%s -> %s (outdated)", marker, v, latest))
		} else if v == latest {
			PrintGreen(fmt.Sprintf("%s%s (latest)", marker, v))
		} else {
			PrintGreen(fmt.Sprintf("%s%s", marker, v))
		}
	}

	if !hasOutdated {
		fmt.Println()
		PrintGreen("All versions are up to date!")
	}

	return nil
}

func (a *app) handleSelfUninstall() error {
	// Safety check: ensure we're only deleting the .sv directory
	if paths.Home == "" || !strings.HasSuffix(paths.Home, ".sv") {
		return NewError("invalid sv home directory, refusing to uninstall")
	}

	PrintYellow("This will remove sv and all installed Go versions.")
	PrintYellow(fmt.Sprintf("Directory to be removed: %s", paths.Home))
	PrintRed("WARNING: This action cannot be undone!")

	var confirmText string
	err := survey.AskOne(&survey.Input{
		Message: "Type 'yes' to confirm uninstall:",
	}, &confirmText)
	if err != nil {
		return err
	}

	if confirmText != "yes" {
		PrintBlue("Uninstall cancelled")
		return nil
	}

	// Clean environment configuration
	if runtime.GOOS == "windows" {
		cleanWindowsEnv()
	} else {
		cleanShellProfile()
	}

	// Remove sv directory
	if err := os.RemoveAll(paths.Home); err != nil {
		return NewError(fmt.Sprintf("failed to remove %s: %v", paths.Home, err))
	}

	PrintGreen("sv has been uninstalled successfully!")
	if runtime.GOOS == "windows" {
		PrintCyan("Please open a new terminal for changes to take effect")
	} else {
		PrintCyan("Please restart your terminal or run: source ~/.bashrc (or ~/.zshrc)")
	}
	return nil
}

func cleanShellProfile() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		Warnf("Failed to get home directory: %v", err)
		return
	}

	profiles := []string{
		filepath.Join(homeDir, ".bashrc"),
		filepath.Join(homeDir, ".bash_profile"),
		filepath.Join(homeDir, ".zshrc"),
		filepath.Join(homeDir, ".profile"),
	}

	// Only match the exact line added by sv installer
	envLine := `. "$HOME/.sv/env"`
	cleaned := false

	for _, profile := range profiles {
		if removeLineFromFile(profile, envLine) {
			cleaned = true
		}
	}

	if cleaned {
		PrintGreen("Cleaned shell profile")
	} else {
		PrintCyan("No sv configuration found in shell profiles")
	}
}

func removeLineFromFile(filePath, lineToRemove string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false // File doesn't exist or can't be read
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	removed := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Only remove the exact sv env line
		if trimmed == lineToRemove {
			removed = true
			// Also remove "# Added by sv installer" comment if it's the previous line
			if len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) == "# Added by sv installer" {
				newLines = newLines[:len(newLines)-1]
			}
			// Remove empty line before comment if exists
			if len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) == "" {
				newLines = newLines[:len(newLines)-1]
			}
			continue
		}
		// Preserve file structure
		if i == len(lines)-1 && line == "" {
			continue
		}
		newLines = append(newLines, line)
	}

	if !removed {
		return false
	}

	if err := os.WriteFile(filePath, []byte(strings.Join(newLines, "\n")+"\n"), 0644); err != nil {
		Warnf("Failed to update %s: %v", filePath, err)
		return false
	}
	return true
}

func cleanWindowsEnv() {
	PrintCyan("Cleaning Windows environment variables...")

	// PowerShell script to clean user environment variables
	script := fmt.Sprintf(`
$binDir = '%s'
$goRoot = '%s'

# Clean PATH
$userPath = [System.Environment]::GetEnvironmentVariable('PATH', 'User')
if ($userPath) {
    $paths = $userPath -split ';' | Where-Object { $_ -ne $binDir -and $_ -ne '' }
    $newPath = $paths -join ';'
    [System.Environment]::SetEnvironmentVariable('PATH', $newPath, 'User')
}

# Clean GOROOT if it points to sv
$currentGoRoot = [System.Environment]::GetEnvironmentVariable('GOROOT', 'User')
if ($currentGoRoot -eq $goRoot) {
    [System.Environment]::SetEnvironmentVariable('GOROOT', $null, 'User')
}
`, paths.Bin, paths.Root)

	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	if err := cmd.Run(); err != nil {
		Warnf("Failed to clean environment variables: %v", err)
		PrintYellow("Please manually remove the following from your user environment variables:")
		PrintYellow(fmt.Sprintf("  - Remove '%s' from PATH", paths.Bin))
		PrintYellow(fmt.Sprintf("  - Remove GOROOT if it points to '%s'", paths.Root))
	} else {
		PrintGreen("Cleaned Windows environment variables")
	}
}
