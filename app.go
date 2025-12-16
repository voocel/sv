package main

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

type app struct {
	opts   *startOpts
	client *http.Client
}

type startOpts struct {
	cmd    string
	target string
	latest string
	remote bool
	force  bool
	// prune options
	keep   int
	all    bool
	dryRun bool
}

func newApp(opts *startOpts) *app {
	return &app{
		opts: opts,
		client: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
	}
}

func (a *app) Start() error {
	switch a.opts.cmd {
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
	case "upgrade":
		return a.handleUpgrade()
	default:
		return ErrUnsupportedCommand()
	}
}

func (a *app) handleList() error {
	if a.opts.remote {
		return a.listRemote()
	}
	return a.listLocal()
}

func (a *app) handleUse() error {
	if a.opts.target == "" {
		return ErrTagEmpty()
	}

	p := &Package{
		Tag:  a.opts.target,
		Name: a.tagToName(a.opts.target),
	}

	if err := p.useLocal(); err != nil {
		return a.promptRemoteInstall(p)
	}
	return nil
}

func (a *app) handleInstall() error {
	if a.opts.target == "" {
		return ErrTagEmpty()
	}

	p := &Package{
		Tag:  a.opts.target,
		Name: a.tagToName(a.opts.target),
		URL:  a.tagToURL(a.opts.target),
	}
	return p.install()
}

func (a *app) handleUninstall() error {
	if a.opts.target == "" {
		return ErrTagEmpty()
	}

	p := &Package{
		Tag:  a.opts.target,
		Name: a.tagToName(a.opts.target),
	}
	return p.remove()
}

func (a *app) handleUpgrade() error {
	u := NewUpgrade(a.opts.force)
	return u.checkUpgrade()
}

func (a *app) promptRemoteInstall(p *Package) error {
	var ok bool
	err := survey.AskOne(&survey.Confirm{
		Message: "Do you like to download and install from remote?",
	}, &ok)
	if err != nil {
		return err
	}

	if ok {
		p.URL = a.tagToURL(p.Tag)
		return p.useRemote()
	}
	return nil
}

func (a *app) listRemote() error {
	resp, err := a.client.Get(cfg.BaseURL + "/dl")
	if err != nil {
		return fmt.Errorf("failed to fetch version list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to fetch version list: status %d", resp.StatusCode)
	}

	respDate, err := a.client.Get(cfg.ReleaseURL)
	if err != nil {
		return fmt.Errorf("failed to fetch release dates: %w", err)
	}
	defer respDate.Body.Close()

	parser, err := NewParser(resp.Body)
	if err != nil {
		return err
	}

	dateParser, err := NewDateParser(respDate.Body)
	if err != nil {
		// Non-fatal: continue without release dates
		Warnf("Could not parse release dates: %v", err)
	}

	var releases map[string]string
	if dateParser != nil {
		releases = dateParser.findReleaseDate()
		parser.setReleases(releases)
	}

	archive := parser.AllVersions()
	if len(archive) == 0 {
		return NewError("no versions found, the page structure may have changed")
	}

	versions := a.formatVersions(archive, releases)
	target, err := a.selectVersions(versions)
	if err != nil {
		return err
	}

	targetPkg := a.getPackage(target, archive)
	if targetPkg == nil {
		return NewError("package not found for version: " + target)
	}

	return targetPkg.use()
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
	pkg.Name = a.tagToName(target)
	return pkg.useLocal()
}

func (a *app) formatVersions(archive map[string]*Version, releases map[string]string) []string {
	versions := make([]string, 0, len(archive))
	for name := range archive {
		if release := releases[name]; release != "" {
			versions = append(versions, fmt.Sprintf("%v (%v)", name, release))
		} else {
			versions = append(versions, name)
		}
	}
	return versions
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
	}, &target, survey.WithValidator(survey.Required), a.surveyIcon())
	if err != nil {
		return "", err
	}
	if i := strings.Index(target, "("); i != -1 {
		target = strings.TrimSpace(target[:i])
	}

	return target, nil
}

func (a *app) surveyIcon() survey.AskOpt {
	return survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Text = "→"
	})
}

func (a *app) getPackage(target string, m map[string]*Version) *Package {
	archive, ok := m[target]
	if !ok {
		return nil
	}

	filename := a.tagToName(target)
	for _, v := range archive.Packages {
		if strings.HasPrefix(v.Name, filename) {
			return v
		}
	}
	return nil
}

func (a *app) tagToName(tag string) string {
	return generateFileName(tag)
}

func (a *app) tagToURL(tag string) string {
	return generateDownloadURL(tag)
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

	// Sort versions (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return versionCompare(versions[i]) > versionCompare(versions[j])
	})

	currentVersion := getCurrentVersion()

	var toRemove []string
	var toKeep []string

	if a.opts.all {
		// Remove all except current
		for _, v := range versions {
			if v == currentVersion {
				toKeep = append(toKeep, v)
			} else {
				toRemove = append(toRemove, v)
			}
		}
	} else {
		// Keep the N most recent versions
		keep := a.opts.keep
		if keep < 1 {
			keep = 2
		}

		for i, v := range versions {
			if v == currentVersion {
				toKeep = append(toKeep, v)
			} else if i < keep || (currentVersion != "" && i < keep+1) {
				toKeep = append(toKeep, v)
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

	if a.opts.dryRun {
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
		p := &Package{
			Tag:  v,
			Name: a.tagToName(v),
		}
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
