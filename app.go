package main

import (
	"fmt"
	"net/http"
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
	switch a.ctx.Command.Name {
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
	case "upgrade":
		return a.handleUpgrade()
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
