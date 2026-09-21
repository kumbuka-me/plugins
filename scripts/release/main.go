package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	defaultRemote = "origin"
	defaultBranch = "main"
)

// releasePlugin contains one selected plugin release from planning through tagging.
type releasePlugin struct {
	// Name is the plugin directory name.
	Name string
	// Manifest is the repository-relative plugin manifest path.
	Manifest string
	// Current is the version currently declared by the plugin.
	Current string
	// Bump is the selected semantic-version increment.
	Bump string
	// Next is the version produced by Bump.
	Next string
	// Tag is the plugin-scoped Git tag created for Next.
	Tag string
}

// main runs the interactive plugin release wizard.
func main() {
	if err := run(context.Background(), os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "release:", err)
		os.Exit(1)
	}
}

// run plans, validates, commits, tags, and pushes one or more plugin releases.
func run(ctx context.Context, input io.Reader, stdout, stderr io.Writer) error {
	root, err := repositoryRoot(ctx)
	if err != nil {
		return err
	}

	runner := commandRunner{root: root, stdout: stdout, stderr: stderr}
	if err := ensureReleaseRepository(ctx, runner); err != nil {
		return err
	}

	branch := envOrDefault("RELEASE_REF", defaultBranch)
	remote := envOrDefault("RELEASE_REMOTE", defaultRemote)
	makeCommand := envOrDefault("RELEASE_MAKE", "make")
	if err := ensureReleaseTarget(ctx, runner, remote, branch); err != nil {
		return err
	}

	plugins, err := discoverPlugins(root)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(input)
	selected, err := choosePlugins(reader, stdout, plugins)
	if err != nil {
		return err
	}
	if err := chooseBumps(reader, stdout, selected); err != nil {
		return err
	}

	preparePlan(selected)
	printPlan(stdout, selected, remote, branch)

	confirmed, err := confirm(reader, stdout, "Create these releases? [y/N]: ")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(stdout, "Release cancelled.") // nolint:errcheck
		return nil
	}
	if err := checkTagsAvailable(ctx, runner, remote, selected); err != nil {
		return err
	}

	manifests := pluginManifests(selected)
	versionsChanged := false
	commitsStarted := false
	rollback := func(releaseErr error) error {
		if !versionsChanged || commitsStarted {
			return releaseErr
		}
		if restoreErr := restoreVersions(ctx, runner, manifests); restoreErr != nil {
			return errors.Join(releaseErr, restoreErr)
		}
		return releaseErr
	}

	for _, plugin := range selected {
		if err := runner.run(ctx, "./scripts/version-plugins.sh", plugin.Name, plugin.Bump); err != nil {
			return rollback(err)
		}
		versionsChanged = true
	}

	if err := ensureExpectedChanges(ctx, runner, manifests); err != nil {
		return rollback(err)
	}
	if err := validateReleases(ctx, runner, makeCommand, selected); err != nil {
		return rollback(err)
	}
	if err := ensureExpectedChanges(ctx, runner, manifests); err != nil {
		return rollback(err)
	}

	fmt.Fprintln(stdout, "\nCreating release commits and tags...") // nolint:errcheck
	commitsStarted = true
	if err := commitAndTag(ctx, runner, selected); err != nil {
		return err
	}
	if err := ensureClean(ctx, runner); err != nil {
		return fmt.Errorf("release commits created but working tree is not clean: %w", err)
	}

	if err := pushReleases(ctx, runner, stdout, remote, branch, selected); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "\nReleased:") // nolint:errcheck
	for _, plugin := range selected {
		fmt.Fprintf(stdout, "  %s\n", plugin.Tag) // nolint:errcheck
	}
	fmt.Fprintln(stdout, "Each tag was pushed separately so every tag-triggered release workflow can run.") // nolint:errcheck
	return nil
}

// validateReleases runs repository checks and builds every selected plugin before committing.
func validateReleases(ctx context.Context, runner commandRunner, makeCommand string, plugins []releasePlugin) error {
	fmt.Fprintln(runner.stdout, "\nValidating repository...") // nolint:errcheck
	if err := runner.run(ctx, makeCommand, "test"); err != nil {
		return err
	}
	if err := runner.run(ctx, makeCommand, "lint"); err != nil {
		return err
	}
	for _, plugin := range plugins {
		if err := runner.run(ctx, makeCommand, "build-plugin", "PLUGIN="+plugin.Name); err != nil {
			return err
		}
	}
	return nil
}

// commitAndTag creates one release commit and matching tag per selected plugin.
func commitAndTag(ctx context.Context, runner commandRunner, plugins []releasePlugin) error {
	for _, plugin := range plugins {
		if err := runner.run(ctx, "git", "add", "--", plugin.Manifest); err != nil {
			return err
		}
		message := fmt.Sprintf("chore: release %s v%s", plugin.Name, plugin.Next)
		if err := runner.run(ctx, "git", "commit", "-m", message); err != nil {
			return err
		}
		if err := runner.run(ctx, "git", "tag", plugin.Tag); err != nil {
			return err
		}
	}
	return nil
}

// pushReleases pushes the branch once and each tag separately to preserve GitHub tag events.
func pushReleases(ctx context.Context, runner commandRunner, output io.Writer, remote, branch string, plugins []releasePlugin) error {
	fmt.Fprintf(output, "\nPushing %s and release tags...\n", branch) // nolint:errcheck
	if err := runner.run(ctx, "git", "push", remote, branch); err != nil {
		return fmt.Errorf("release commits and tags remain local: %w", err)
	}
	for _, plugin := range plugins {
		if err := runner.run(ctx, "git", "push", remote, plugin.Tag); err != nil {
			return fmt.Errorf("push %s: %w", plugin.Tag, err)
		}
	}
	return nil
}
