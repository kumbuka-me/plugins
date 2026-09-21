package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	root, docs := t.TempDir(), t.TempDir()
	files := map[string]string{"sample/plugin.yaml": "id: me.kumbuka.sample\nname: Sample\nversion: 1.0.0\ndescription: Sample plugin\npermissions: [pages:read]\n", "sample/README.md": "# Sample\n\n## Usage\n\nExample.\n", "sample/assets/preview.png": "preview"}
	for path, data := range files {
		if err := write(filepath.Join(root, path), []byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	stale := filepath.Join(docs, "content/plugins/packages/removed.md")
	manual := filepath.Join(docs, "content/plugins/packages/manual.md")
	for path, data := range map[string]string{stale: header + "old", manual: "# Manual", filepath.Join(docs, "assets/plugins/removed/preview.png"): "old"} {
		if err := write(path, []byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := generate(root, docs); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(docs, "content/plugins/packages/sample.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## Usage", "`pages:read`", "/assets/plugins/sample/preview.png"} {
		if !strings.Contains(string(page), want) {
			t.Fatalf("missing %s", want)
		}
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatal("stale page retained")
	}
	if _, err := os.Stat(manual); err != nil {
		t.Fatal("manual page removed")
	}
	if err := generate(root, docs); err != nil {
		t.Fatal(err)
	}
	again, _ := os.ReadFile(filepath.Join(docs, "content/plugins/packages/sample.md"))
	if string(again) != string(page) {
		t.Fatal("generation is not deterministic")
	}
	if err := os.Remove(filepath.Join(root, "sample/assets/preview.png")); err != nil {
		t.Fatal(err)
	}
	if err := generate(root, docs); err == nil {
		t.Fatal("missing preview accepted")
	}
}

func TestRewriteLinks(t *testing.T) {
	got := rewriteLinks("[usage](guide.md) ![image](assets/example.png) [web](https://example.com) [anchor](#usage)", "sample")
	for _, want := range []string{"blob/main/sample/guide.md", "raw.githubusercontent.com/kumbuka-me/plugins/main/sample/assets/example.png", "[web](https://example.com)", "[anchor](#usage)"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %s", want)
		}
	}
}
