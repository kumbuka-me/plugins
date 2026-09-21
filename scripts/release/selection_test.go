package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestParseSelection verifies individual, range, all, and invalid plugin selections.
func TestParseSelection(t *testing.T) {
	t.Run("individual and ranges", func(t *testing.T) {
		indices, err := parseSelection("1, 3-4 3", 5)
		if err != nil {
			t.Fatal(err)
		}
		want := []int{0, 2, 3}
		if !reflect.DeepEqual(indices, want) {
			t.Fatalf("indices = %v, want %v", indices, want)
		}
	})

	t.Run("all", func(t *testing.T) {
		indices, err := parseSelection("all", 4)
		if err != nil {
			t.Fatal(err)
		}
		want := []int{0, 1, 2, 3}
		if !reflect.DeepEqual(indices, want) {
			t.Fatalf("indices = %v, want %v", indices, want)
		}
	})

	t.Run("empty", func(t *testing.T) {
		_, err := parseSelection("", 4)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("out of range", func(t *testing.T) {
		_, err := parseSelection("5", 4)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("reversed range", func(t *testing.T) {
		_, err := parseSelection("4-2", 4)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("malformed range", func(t *testing.T) {
		_, err := parseSelection("1-2-3", 4)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// TestNextVersion verifies strict semantic-version increments.
func TestNextVersion(t *testing.T) {
	t.Run("patch", func(t *testing.T) {
		version, err := nextVersion("1.2.3", "patch")
		if err != nil {
			t.Fatal(err)
		}
		if version != "1.2.4" {
			t.Fatalf("version = %q, want %q", version, "1.2.4")
		}
	})

	t.Run("minor", func(t *testing.T) {
		version, err := nextVersion("1.2.3", "minor")
		if err != nil {
			t.Fatal(err)
		}
		if version != "1.3.0" {
			t.Fatalf("version = %q, want %q", version, "1.3.0")
		}
	})

	t.Run("major", func(t *testing.T) {
		version, err := nextVersion("1.2.3", "major")
		if err != nil {
			t.Fatal(err)
		}
		if version != "2.0.0" {
			t.Fatalf("version = %q, want %q", version, "2.0.0")
		}
	})

	t.Run("leading zero", func(t *testing.T) {
		_, err := nextVersion("1.02.3", "patch")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("pre release", func(t *testing.T) {
		_, err := nextVersion("1.2.3-rc.1", "patch")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("unknown bump", func(t *testing.T) {
		_, err := nextVersion("1.2.3", "breaking")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// TestParseBump verifies accepted interactive bump aliases.
func TestParseBump(t *testing.T) {
	t.Run("patch", func(t *testing.T) {
		bump, ok := parseBump("p")
		if !ok || bump != "patch" {
			t.Fatalf("bump = %q, ok = %v", bump, ok)
		}
	})

	t.Run("minor", func(t *testing.T) {
		bump, ok := parseBump("2")
		if !ok || bump != "minor" {
			t.Fatalf("bump = %q, ok = %v", bump, ok)
		}
	})

	t.Run("major", func(t *testing.T) {
		bump, ok := parseBump("major")
		if !ok || bump != "major" {
			t.Fatalf("bump = %q, ok = %v", bump, ok)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		_, ok := parseBump("other")
		if ok {
			t.Fatal("expected invalid bump")
		}
	})
}

// TestReadVersion verifies manifest versions use the same strict shape as release scripts.
func TestReadVersion(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		manifest := filepath.Join(t.TempDir(), "plugin.yaml")
		if err := os.WriteFile(manifest, []byte("name: Test\nversion: 1.2.3\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		version, err := readVersion(manifest)
		if err != nil {
			t.Fatal(err)
		}
		if version != "1.2.3" {
			t.Fatalf("version = %q, want %q", version, "1.2.3")
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		manifest := filepath.Join(t.TempDir(), "plugin.yaml")
		if err := os.WriteFile(manifest, []byte("version: 1.2.3\n  version: 1.2.4\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := readVersion(manifest)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid", func(t *testing.T) {
		manifest := filepath.Join(t.TempDir(), "plugin.yaml")
		if err := os.WriteFile(manifest, []byte("version: 1.2\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := readVersion(manifest)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
