package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// discoverPlugins loads first-party plugin names and semantic versions in stable order.
func discoverPlugins(root string) ([]releasePlugin, error) {
	manifests, err := filepath.Glob(filepath.Join(root, "*", "plugin.yaml"))
	if err != nil {
		return nil, err
	}
	if len(manifests) == 0 {
		return nil, errors.New("no plugin manifests found")
	}
	sort.Strings(manifests)

	plugins := make([]releasePlugin, 0, len(manifests))
	for _, manifest := range manifests {
		version, err := readVersion(manifest)
		if err != nil {
			return nil, err
		}
		name := filepath.Base(filepath.Dir(manifest))
		plugins = append(plugins, releasePlugin{
			Name:     name,
			Manifest: filepath.ToSlash(filepath.Join(name, "plugin.yaml")),
			Current:  version,
		})
	}
	return plugins, nil
}

// readVersion reads and validates the single semantic version from one plugin manifest.
func readVersion(manifest string) (string, error) {
	file, err := os.Open(manifest)
	if err != nil {
		return "", err
	}
	defer file.Close() // nolint:errcheck

	scanner := bufio.NewScanner(file)
	version := ""
	count := 0
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || fields[0] != "version:" {
			continue
		}
		count++
		if len(fields) > 1 {
			version = fields[1]
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if count != 1 {
		return "", fmt.Errorf("%s: manifest must contain exactly one version", manifest)
	}
	if _, err := nextVersion(version, "patch"); err != nil {
		return "", fmt.Errorf("%s: %w", manifest, err)
	}
	return version, nil
}

// choosePlugins prompts until the user selects one, several, or all plugins.
func choosePlugins(reader *bufio.Reader, output io.Writer, plugins []releasePlugin) ([]releasePlugin, error) {
	fmt.Fprintln(output, "Select plugins:") // nolint:errcheck
	fmt.Fprintln(output)                    // nolint:errcheck
	for index, plugin := range plugins {
		fmt.Fprintf(output, "  %2d) %-24s %s\n", index+1, plugin.Name, plugin.Current) // nolint:errcheck
	}
	fmt.Fprintln(output, "\nEnter numbers/ranges such as 2,5-7 or 'all'.") // nolint:errcheck

	for {
		value, err := promptLine(reader, output, "Plugins: ")
		if err != nil {
			return nil, err
		}
		indices, err := parseSelection(value, len(plugins))
		if err != nil {
			fmt.Fprintf(output, "Invalid selection: %v\n", err) // nolint:errcheck
			continue
		}

		selected := make([]releasePlugin, 0, len(indices))
		for _, index := range indices {
			selected = append(selected, plugins[index])
		}
		return selected, nil
	}
}

// parseSelection converts a numeric/range selection into stable zero-based indexes.
func parseSelection(value string, count int) ([]int, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "all" || value == "*" {
		indices := make([]int, count)
		for index := range indices {
			indices[index] = index
		}
		return indices, nil
	}
	if value == "" {
		return nil, errors.New("select at least one plugin")
	}

	selected := make([]bool, count)
	tokens := strings.FieldsSeq(strings.ReplaceAll(value, ",", " "))
	for token := range tokens {
		startText, endText, ranged := strings.Cut(token, "-")
		start, err := selectionNumber(startText, count)
		if err != nil {
			return nil, err
		}
		end := start
		if ranged {
			if endText == "" || strings.Contains(endText, "-") {
				return nil, fmt.Errorf("invalid range %q", token)
			}
			end, err = selectionNumber(endText, count)
			if err != nil {
				return nil, err
			}
			if end < start {
				return nil, fmt.Errorf("range %q is reversed", token)
			}
		}
		for number := start; number <= end; number++ {
			selected[number-1] = true
		}
	}

	indices := make([]int, 0)
	for index, chosen := range selected {
		if chosen {
			indices = append(indices, index)
		}
	}
	if len(indices) == 0 {
		return nil, errors.New("select at least one plugin")
	}
	return indices, nil
}

// selectionNumber parses one one-based plugin selection number.
func selectionNumber(value string, count int) (int, error) {
	number, err := strconv.Atoi(value)
	if err != nil || number < 1 || number > count {
		return 0, fmt.Errorf("plugin number %q must be between 1 and %d", value, count)
	}
	return number, nil
}

// chooseBumps selects one bump for all plugins or prompts for each plugin individually.
func chooseBumps(reader *bufio.Reader, output io.Writer, plugins []releasePlugin) error {
	fmt.Fprintln(output, "\nVersion bump:")                     // nolint:errcheck
	fmt.Fprintln(output)                                        // nolint:errcheck
	fmt.Fprintln(output, "  1) patch for all selected plugins") // nolint:errcheck
	fmt.Fprintln(output, "  2) minor for all selected plugins") // nolint:errcheck
	fmt.Fprintln(output, "  3) major for all selected plugins") // nolint:errcheck
	fmt.Fprintln(output, "  4) choose per plugin")              // nolint:errcheck

	for {
		value, err := promptLine(reader, output, "Bump: ")
		if err != nil {
			return err
		}
		if value == "4" || strings.EqualFold(value, "custom") {
			for index := range plugins {
				bump, err := choosePluginBump(reader, output, plugins[index])
				if err != nil {
					return err
				}
				plugins[index].Bump = bump
			}
			return nil
		}

		bump, ok := parseBump(value)
		if !ok {
			fmt.Fprintln(output, "Invalid bump. Choose patch, minor, major, or custom.") // nolint:errcheck
			continue
		}
		for index := range plugins {
			plugins[index].Bump = bump
		}
		return nil
	}
}

// choosePluginBump prompts for one plugin-specific semantic-version increment.
func choosePluginBump(reader *bufio.Reader, output io.Writer, plugin releasePlugin) (string, error) {
	patch, _ := nextVersion(plugin.Current, "patch")
	minor, _ := nextVersion(plugin.Current, "minor")
	major, _ := nextVersion(plugin.Current, "major")
	for {
		prompt := fmt.Sprintf("%s %s [patch=%s, minor=%s, major=%s]: ", plugin.Name, plugin.Current, patch, minor, major)
		value, err := promptLine(reader, output, prompt)
		if err != nil {
			return "", err
		}
		bump, ok := parseBump(value)
		if ok {
			return bump, nil
		}
		fmt.Fprintln(output, "Invalid bump. Choose patch, minor, or major.") // nolint:errcheck
	}
}

// parseBump normalizes numeric and textual semantic-version bump choices.
func parseBump(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "p", "patch":
		return "patch", true
	case "2", "m", "minor":
		return "minor", true
	case "3", "major":
		return "major", true
	default:
		return "", false
	}
}

// nextVersion applies one patch, minor, or major increment to a strict semantic version.
func nextVersion(current, bump string) (string, error) {
	parts := strings.Split(current, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid plugin version %q (expected MAJOR.MINOR.PATCH)", current)
	}

	major, err := versionPart(parts[0])
	if err != nil {
		return "", fmt.Errorf("invalid plugin version %q", current)
	}
	minor, err := versionPart(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid plugin version %q", current)
	}
	patch, err := versionPart(parts[2])
	if err != nil {
		return "", fmt.Errorf("invalid plugin version %q", current)
	}

	switch bump {
	case "patch":
		patch++
	case "minor":
		minor++
		patch = 0
	case "major":
		major++
		minor = 0
		patch = 0
	default:
		return "", fmt.Errorf("invalid bump %q", bump)
	}
	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}

// versionPart parses one non-negative decimal semantic-version component.
func versionPart(value string) (int, error) {
	if value == "" || len(value) > 1 && value[0] == '0' {
		return 0, errors.New("invalid version component")
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return 0, errors.New("invalid version component")
		}
	}
	return strconv.Atoi(value)
}

// preparePlan calculates each selected plugin's target version and release tag.
func preparePlan(plugins []releasePlugin) {
	for index := range plugins {
		next, _ := nextVersion(plugins[index].Current, plugins[index].Bump)
		plugins[index].Next = next
		plugins[index].Tag = plugins[index].Name + "/v" + next
	}
}

// printPlan displays the exact release commits, tags, and push destination.
func printPlan(output io.Writer, plugins []releasePlugin, remote, branch string) {
	fmt.Fprintln(output, "\nRelease plan:") // nolint:errcheck
	fmt.Fprintln(output)                    // nolint:errcheck
	for _, plugin := range plugins {
		fmt.Fprintf(output, "  %-24s %s -> %-10s %s\n", plugin.Name, plugin.Current, plugin.Next, plugin.Tag) // nolint:errcheck
	}
	fmt.Fprintf(output, "\nThe wizard will run tests/lint, build every selected plugin, create one commit and tag per plugin, push %s to %s, then push each tag separately.\n\n", branch, remote) // nolint:errcheck
}

// promptLine reads one trimmed line from an interactive prompt.
func promptLine(reader *bufio.Reader, output io.Writer, prompt string) (string, error) {
	fmt.Fprint(output, prompt) // nolint:errcheck
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if errors.Is(err, io.EOF) && value == "" {
		return "", io.EOF
	}
	return strings.TrimSpace(value), nil
}

// confirm prompts until the user answers yes or no, defaulting an empty answer to no.
func confirm(reader *bufio.Reader, output io.Writer, prompt string) (bool, error) {
	for {
		value, err := promptLine(reader, output, prompt)
		if err != nil {
			return false, err
		}
		switch strings.ToLower(value) {
		case "y", "yes":
			return true, nil
		case "", "n", "no":
			return false, nil
		default:
			fmt.Fprintln(output, "Please answer yes or no.") // nolint:errcheck
		}
	}
}
