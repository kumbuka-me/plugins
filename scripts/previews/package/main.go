// Command preview-package rewrites plugin packages for the static preview renderer.
package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var staticPreviewPermissions = map[string]bool{
	"browser:render": true,
	"pages:content":  true,
	"pages:read":     true,
}

// main rewrites every supplied package in place and exits non-zero on failure.
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/previews/package <package> [package ...]")
		os.Exit(2)
	}

	for _, filename := range os.Args[1:] {
		if err := rewritePackage(filename); err != nil {
			fmt.Fprintf(os.Stderr, "prepare preview package %s: %v\n", filename, err)
			os.Exit(1)
		}
	}
}

// rewritePackage replaces one package manifest with its static-preview permission subset.
func rewritePackage(filename string) error {
	archive, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	files, err := readArchive(archive)
	if err != nil {
		return err
	}

	manifest, ok := files["plugin.yaml"]
	if !ok {
		return fmt.Errorf("missing plugin.yaml")
	}
	files["plugin.yaml"], err = previewManifest(manifest)
	if err != nil {
		return err
	}

	rewritten, err := writeArchive(files)
	if err != nil {
		return err
	}

	temporary, err := os.CreateTemp(filepath.Dir(filename), ".preview-package-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer func() { _ = os.Remove(temporaryName) }()

	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(rewritten); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}

	return os.Rename(temporaryName, filename)
}

// previewManifest keeps only permissions supported by the pinned static-site CLI renderer.
func previewManifest(source []byte) ([]byte, error) {
	lines := strings.Split(string(source), "\n")
	start := -1
	end := -1
	var permissions []string

	for index, line := range lines {
		if strings.TrimLeft(line, " \t") != line || !strings.HasPrefix(line, "permissions:") {
			continue
		}

		start = index
		remainder := strings.TrimSpace(strings.TrimPrefix(line, "permissions:"))
		if remainder != "" {
			var err error
			permissions, err = inlinePermissions(remainder)
			if err != nil {
				return nil, err
			}
			end = index + 1
			break
		}

		end = index + 1
		for end < len(lines) {
			candidate := lines[end]
			trimmed := strings.TrimSpace(candidate)
			if trimmed == "" || strings.TrimLeft(candidate, " \t") == candidate {
				break
			}
			if !strings.HasPrefix(trimmed, "- ") {
				return nil, fmt.Errorf("unsupported permissions entry %q", candidate)
			}
			permission := unquote(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
			if permission == "" {
				return nil, fmt.Errorf("empty permission")
			}
			permissions = append(permissions, permission)
			end++
		}
		break
	}

	if start < 0 {
		return nil, fmt.Errorf("plugin.yaml is missing permissions")
	}

	filtered := previewPermissions(permissions)
	replacement := []string{"permissions: []"}
	if len(filtered) > 0 {
		replacement = []string{"permissions:"}
		for _, permission := range filtered {
			replacement = append(replacement, "  - "+permission)
		}
	}

	result := make([]string, 0, len(lines)-(end-start)+len(replacement))
	result = append(result, lines[:start]...)
	result = append(result, replacement...)
	result = append(result, lines[end:]...)
	return []byte(strings.Join(result, "\n")), nil
}

// previewPermissions returns the unique render-safe permissions in deterministic order.
func previewPermissions(permissions []string) []string {
	filtered := make([]string, 0, len(permissions))
	seen := make(map[string]bool)
	for _, permission := range permissions {
		if staticPreviewPermissions[permission] && !seen[permission] {
			seen[permission] = true
			filtered = append(filtered, permission)
		}
	}
	sort.Strings(filtered)
	return filtered
}

// inlinePermissions parses the simple YAML flow sequence used by plugin manifests.
func inlinePermissions(value string) ([]string, error) {
	if value == "[]" {
		return nil, nil
	}
	if len(value) < 2 || value[0] != '[' || value[len(value)-1] != ']' {
		return nil, fmt.Errorf("unsupported inline permissions %q", value)
	}

	body := strings.TrimSpace(value[1 : len(value)-1])
	if body == "" {
		return nil, nil
	}

	parts := strings.Split(body, ",")
	permissions := make([]string, 0, len(parts))
	for _, part := range parts {
		permission := unquote(strings.TrimSpace(part))
		if permission == "" {
			return nil, fmt.Errorf("empty permission")
		}
		permissions = append(permissions, permission)
	}
	return permissions, nil
}

// unquote removes optional simple quotes around a permission name.
func unquote(value string) string {
	if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"' || value[0] == '\'' && value[len(value)-1] == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}

// readArchive reads regular package files without extracting them to disk.
func readArchive(data []byte) (map[string][]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("read archive: %w", err)
	}

	files := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if _, exists := files[file.Name]; exists {
			return nil, fmt.Errorf("duplicate archive path %q", file.Name)
		}
		entry, err := file.Open()
		if err != nil {
			return nil, err
		}
		content, readErr := io.ReadAll(entry)
		closeErr := entry.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		files[file.Name] = content
	}
	return files, nil
}

// writeArchive serializes package files using deterministic names and metadata.
func writeArchive(files map[string][]byte) ([]byte, error) {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0o644)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			return nil, err
		}
		if _, err := entry.Write(files[name]); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
