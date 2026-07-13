package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v3"
)

func (a *App) cmdList(ctx context.Context, cmd *cli.Command) error {
	if cmd.Bool("remote") {
		return a.listRemote(ctx)
	}
	return a.listLocal()
}

func (a *App) listLocal() error {
	versions, err := a.localVersions()
	if err != nil {
		return err
	}
	if len(versions) == 0 {
		infof("no Go versions installed yet — try `sv list -r` to browse remote versions")
		return nil
	}

	title := "Choose a version:"
	if current := a.currentVersion(); current != "" {
		title = fmt.Sprintf("Choose a version (current: %s):", current)
	}
	tag, err := selectPrompt(title, versions)
	if err != nil {
		return err
	}
	return a.switchTo(tag)
}

func (a *App) listRemote(ctx context.Context) error {
	releases, err := a.fetchReleases(ctx, true)
	if err != nil {
		return err
	}
	if len(releases) == 0 {
		return errors.New("no versions found")
	}

	tag, err := selectPrompt("Choose a version:", releaseTags(releases))
	if err != nil {
		return err
	}
	return a.ensureAndSwitch(ctx, releases, tag)
}

// ensureAndSwitch installs tag if needed, then makes it active.
func (a *App) ensureAndSwitch(ctx context.Context, releases []GoRelease, tag string) error {
	if !a.isInstalled(tag) {
		release := findRelease(releases, tag)
		if release == nil {
			return fmt.Errorf("version not found: %s", tag)
		}
		if err := a.installRelease(ctx, release); err != nil {
			return err
		}
	}
	return a.switchTo(tag)
}

func (a *App) cmdUse(ctx context.Context, cmd *cli.Command) error {
	target := cmd.Args().First()
	if target == "" {
		return errors.New("usage: sv use <version>")
	}

	tag := normalizeTag(target)
	if a.isInstalled(tag) {
		return a.switchTo(tag)
	}

	// A previously downloaded archive can be installed without network.
	if archive := a.localArchive(tag); archive != "" {
		if err := a.installFromArchive(tag, archive); err != nil {
			return err
		}
		return a.switchTo(tag)
	}

	ok, err := confirmPrompt(fmt.Sprintf("%s is not installed. Download and install it?", tag))
	if err != nil || !ok {
		return err
	}

	releases, err := a.fetchReleases(ctx, true)
	if err != nil {
		return err
	}
	return a.ensureAndSwitch(ctx, releases, tag)
}

func (a *App) cmdInstall(ctx context.Context, cmd *cli.Command) error {
	releases, err := a.fetchReleases(ctx, true)
	if err != nil {
		return err
	}

	var tag string
	if cmd.Bool("latest") {
		if tag = latestStableTag(releases); tag == "" {
			return errors.New("no stable version found")
		}
	} else {
		target := cmd.Args().First()
		if target == "" {
			return errors.New("usage: sv install <version>")
		}
		tag = normalizeTag(target)
	}

	release := findRelease(releases, tag)
	if release == nil {
		return fmt.Errorf("version not found: %s", tag)
	}
	if err := a.installRelease(ctx, release); err != nil {
		return err
	}
	return a.switchTo(release.Version)
}

func (a *App) cmdUninstall(_ context.Context, cmd *cli.Command) error {
	target := cmd.Args().First()
	if target == "" {
		return errors.New("usage: sv uninstall <version>")
	}

	tag := normalizeTag(target)
	if err := a.removeVersion(tag); err != nil {
		return err
	}
	successf("uninstalled %s", tag)
	return nil
}

func (a *App) cmdPrune(_ context.Context, cmd *cli.Command) error {
	versions, err := a.localVersions()
	if err != nil {
		return err
	}
	if len(versions) == 0 {
		infof("no installed versions to prune")
		return nil
	}

	current := a.currentVersion()
	keep := cmd.Int("keep")
	if keep < 1 {
		keep = 2 // fall back to the default on nonsense input
	}
	kept, removed := planPrune(versions, current, keep, cmd.Bool("all"))
	if len(removed) == 0 {
		successf("nothing to prune — all versions are within the keep limit")
		return nil
	}

	infof("installed versions: %d", len(versions))
	if current != "" {
		infof("current version:    %s", current)
	}
	infof("versions to keep:   %v", kept)
	warnf("versions to remove: %v", removed)

	if cmd.Bool("dry-run") {
		infof("dry run — no changes made")
		return nil
	}

	ok, err := confirmPrompt(fmt.Sprintf("Remove %d version(s)?", len(removed)))
	if err != nil || !ok {
		return err
	}

	pruned := 0
	for _, tag := range removed {
		if err := a.removeVersion(tag); err != nil {
			warnf("failed to remove %s: %v", tag, err)
			continue
		}
		successf("removed %s", tag)
		pruned++
	}
	successf("pruned %d version(s), kept %d", pruned, len(kept))
	return nil
}

func (a *App) cmdCurrent(context.Context, *cli.Command) error {
	current := a.currentVersion()
	if current == "" {
		infof("no Go version is currently active")
		return nil
	}
	fmt.Println(current)
	return nil
}

func (a *App) cmdWhere(_ context.Context, cmd *cli.Command) error {
	target := cmd.Args().First()
	if target == "" {
		if target = a.currentVersion(); target == "" {
			infof("no Go version is currently active — specify one: sv where <version>")
			return nil
		}
	}

	tag := normalizeTag(target)
	if !a.isInstalled(tag) {
		return fmt.Errorf("version %s is not installed", tag)
	}
	fmt.Println(a.versionPath(tag))
	return nil
}

func (a *App) cmdLatest(ctx context.Context, _ *cli.Command) error {
	latest, err := a.latestStable(ctx)
	if err != nil {
		return err
	}
	fmt.Println(latest)
	return nil
}

func (a *App) cmdOutdated(ctx context.Context, _ *cli.Command) error {
	versions, err := a.localVersions()
	if err != nil {
		return err
	}
	if len(versions) == 0 {
		infof("no installed versions")
		return nil
	}

	latest, err := a.latestStable(ctx)
	if err != nil {
		return err
	}
	current := a.currentVersion()

	infof("latest available: %s", latest)
	if current != "" {
		infof("current active:   %s", current)
	}
	fmt.Println()

	outdated := false
	for _, tag := range versions {
		marker := "  "
		if tag == current {
			marker = "* "
		}
		switch {
		case compareTags(tag, latest) < 0:
			outdated = true
			warnf("%s%s -> %s (outdated)", marker, tag, latest)
		case tag == latest:
			successf("%s%s (latest)", marker, tag)
		default:
			successf("%s%s", marker, tag)
		}
	}
	if !outdated {
		fmt.Println()
		successf("all versions are up to date!")
	}
	return nil
}

func (a *App) cmdSelfUpgrade(ctx context.Context, cmd *cli.Command) error {
	return a.selfUpgrade(ctx, cmd.Bool("force"))
}

func (a *App) cmdSelfUninstall(context.Context, *cli.Command) error {
	return a.selfUninstall()
}
