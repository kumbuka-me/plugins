package main

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"testing"
)

// TestPreviewManifest verifies static previews retain only render-safe permissions.
func TestPreviewManifest(t *testing.T) {
	t.Run("block list", func(t *testing.T) {
		source := []byte("name: Example\npermissions:\n  - browser:render\n  - settings:read\n  - storage:write\n  - pages:content\n")
		result, err := previewManifest(source)
		if err != nil {
			t.Fatalf("previewManifest() error = %v", err)
		}

		text := string(result)
		if !stringsContain(text, "  - browser:render\n") || !stringsContain(text, "  - pages:content\n") {
			t.Fatalf("previewManifest() = %q", text)
		}
		if stringsContain(text, "settings:read") || stringsContain(text, "storage:write") {
			t.Fatalf("previewManifest() retained unsupported permissions: %q", text)
		}
	})

	t.Run("inline list", func(t *testing.T) {
		source := []byte("name: Example\npermissions: [settings:read, pages:read, browser:render]\n")
		result, err := previewManifest(source)
		if err != nil {
			t.Fatalf("previewManifest() error = %v", err)
		}

		text := string(result)
		if !stringsContain(text, "  - browser:render\n") || !stringsContain(text, "  - pages:read\n") {
			t.Fatalf("previewManifest() = %q", text)
		}
		if stringsContain(text, "settings:read") {
			t.Fatalf("previewManifest() retained unsupported permission: %q", text)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		source := []byte("name: Example\npermissions: []\n")
		result, err := previewManifest(source)
		if err != nil {
			t.Fatalf("previewManifest() error = %v", err)
		}
		if string(result) != string(source) {
			t.Fatalf("previewManifest() = %q, want %q", result, source)
		}
	})

	t.Run("unsupported only", func(t *testing.T) {
		source := []byte("name: Example\npermissions: [settings:read, storage:read]\n")
		result, err := previewManifest(source)
		if err != nil {
			t.Fatalf("previewManifest() error = %v", err)
		}
		if string(result) != "name: Example\npermissions: []\n" {
			t.Fatalf("previewManifest() = %q", result)
		}
	})
}

// TestRewritePackage verifies package files survive an in-place preview rewrite.
func TestRewritePackage(t *testing.T) {
	archive, err := writeArchive(map[string][]byte{
		"plugin.yaml":       []byte("name: Example\npermissions:\n  - settings:read\n  - pages:content\n"),
		"README.md":         []byte("# Example\n"),
		"plugin.wasm":       {'\x00', 'a', 's', 'm', 1, 0, 0, 0},
		"assets/plugin.css": []byte(".example { display: block; }\n"),
	})
	if err != nil {
		t.Fatalf("writeArchive() error = %v", err)
	}

	filename := t.TempDir() + "/example.kumbukaplugin"
	if err := os.WriteFile(filename, archive, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := rewritePackage(filename); err != nil {
		t.Fatalf("rewritePackage() error = %v", err)
	}

	rewritten, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	manifest, err := archiveFile(rewritten, "plugin.yaml")
	if err != nil {
		t.Fatalf("archiveFile(plugin.yaml) error = %v", err)
	}
	if string(manifest) != "name: Example\npermissions:\n  - pages:content\n" {
		t.Fatalf("plugin.yaml = %q", manifest)
	}
	wasm, err := archiveFile(rewritten, "plugin.wasm")
	if err != nil {
		t.Fatalf("archiveFile(plugin.wasm) error = %v", err)
	}
	if !bytes.Equal(wasm, []byte{'\x00', 'a', 's', 'm', 1, 0, 0, 0}) {
		t.Fatalf("plugin.wasm = %v", wasm)
	}
	css, err := archiveFile(rewritten, "assets/plugin.css")
	if err != nil {
		t.Fatalf("archiveFile(plugin.css) error = %v", err)
	}
	if string(css) != ".example { display: block; }\n" {
		t.Fatalf("plugin.css = %q", css)
	}
}

// stringsContain reports whether value contains substring without adding test dependencies.
func stringsContain(value, substring string) bool {
	return bytes.Contains([]byte(value), []byte(substring))
}

// archiveFile returns one file from an in-memory ZIP archive.
func archiveFile(data []byte, name string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, file := range reader.File {
		if file.Name != name {
			continue
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
		return content, nil
	}
	return nil, io.EOF
}
