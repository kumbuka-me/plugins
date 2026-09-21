package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// commandRunner executes repository commands with shared output streams.
type commandRunner struct {
	// root is the repository working directory used by every command.
	root string
	// stdout receives normal command output.
	stdout io.Writer
	// stderr receives command diagnostics.
	stderr io.Writer
}

// repositoryRoot resolves the current Git repository root.
func repositoryRoot(ctx context.Context) (string, error) {
	command := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("find repository root: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// envOrDefault returns a trimmed environment value or its fallback.
func envOrDefault(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

// ensureReleaseRepository verifies the repository is clean and Git can create commits.
func ensureReleaseRepository(ctx context.Context, runner commandRunner) error {
	if err := ensureClean(ctx, runner); err != nil {
		return err
	}
	if _, err := runner.output(ctx, "git", "var", "GIT_AUTHOR_IDENT"); err != nil {
		return fmt.Errorf("git author identity is not configured: %w", err)
	}
	return nil
}

// ensureReleaseTarget verifies the current branch and configured release remote.
func ensureReleaseTarget(ctx context.Context, runner commandRunner, remote, branch string) error {
	currentBranch, err := runner.output(ctx, "git", "branch", "--show-current")
	if err != nil {
		return err
	}
	if currentBranch != branch {
		return fmt.Errorf("release must run on %q, current branch is %q", branch, currentBranch)
	}
	if _, err := runner.output(ctx, "git", "remote", "get-url", remote); err != nil {
		return fmt.Errorf("release remote %q is unavailable: %w", remote, err)
	}
	return nil
}

// checkTagsAvailable rejects release tags that already exist locally or remotely.
func checkTagsAvailable(ctx context.Context, runner commandRunner, remote string, plugins []releasePlugin) error {
	local, err := runner.output(ctx, "git", "tag", "--list")
	if err != nil {
		return err
	}
	remoteRefs, err := runner.output(ctx, "git", "ls-remote", "--tags", remote)
	if err != nil {
		return err
	}

	localTags := tagSet(local, false)
	remoteTags := tagSet(remoteRefs, true)
	for _, plugin := range plugins {
		if localTags[plugin.Tag] {
			return fmt.Errorf("tag %s already exists locally", plugin.Tag)
		}
		if remoteTags[plugin.Tag] {
			return fmt.Errorf("tag %s already exists on %s", plugin.Tag, remote)
		}
	}
	return nil
}

// tagSet parses local tag names or git ls-remote tag references into a lookup set.
func tagSet(value string, remote bool) map[string]bool {
	result := make(map[string]bool)
	for _, line := range strings.Split(value, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		if remote {
			if len(fields) < 2 || !strings.HasPrefix(fields[1], "refs/tags/") {
				continue
			}
			name = strings.TrimPrefix(fields[1], "refs/tags/")
			name = strings.TrimSuffix(name, "^{}")
		}
		result[name] = true
	}
	return result
}

// pluginManifests returns selected manifest paths in release order.
func pluginManifests(plugins []releasePlugin) []string {
	manifests := make([]string, 0, len(plugins))
	for _, plugin := range plugins {
		manifests = append(manifests, plugin.Manifest)
	}
	return manifests
}

// restoreVersions restores selected manifests after a pre-commit validation failure.
func restoreVersions(ctx context.Context, runner commandRunner, manifests []string) error {
	args := append([]string{"restore", "--"}, manifests...)
	if err := runner.run(ctx, "git", args...); err != nil {
		return fmt.Errorf("restore version changes: %w", err)
	}
	return nil
}

// ensureClean requires a completely clean tracked and untracked working tree.
func ensureClean(ctx context.Context, runner commandRunner) error {
	status, err := runner.output(ctx, "git", "status", "--porcelain")
	if err != nil {
		return err
	}
	if status != "" {
		return errors.New("working tree must be clean before releasing")
	}
	return nil
}

// ensureExpectedChanges verifies that validation changed only selected plugin manifests.
func ensureExpectedChanges(ctx context.Context, runner commandRunner, manifests []string) error {
	output, err := runner.output(ctx, "git", "diff", "--name-only")
	if err != nil {
		return err
	}

	expected := make(map[string]bool, len(manifests))
	for _, manifest := range manifests {
		expected[manifest] = true
	}
	seen := make(map[string]bool, len(manifests))
	for _, name := range strings.Fields(output) {
		if !expected[name] {
			return fmt.Errorf("validation unexpectedly changed %s", name)
		}
		seen[name] = true
	}
	for _, manifest := range manifests {
		if !seen[manifest] {
			return fmt.Errorf("expected version change missing for %s", manifest)
		}
	}
	return nil
}

// run executes one command while streaming its output.
func (r commandRunner) run(ctx context.Context, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = r.root
	command.Stdout = r.stdout
	command.Stderr = r.stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

// output executes one command and returns its trimmed standard output.
func (r commandRunner) output(ctx context.Context, name string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = r.root
	command.Stderr = r.stderr
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output)), nil
}
