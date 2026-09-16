package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	simpleicons "github.com/go-icons/simple-icons"
)

// resource is the generated versioned icon catalog written to assets/icons.json.
type resource struct {
	// Format identifies the Kumbuka icon-resource JSON format version.
	Format int `json:"format"`
	// Icons contains every generated Simple Icons record.
	Icons []icon `json:"icons"`
}

// icon is one normalized Simple Icons record emitted for the host icon catalog.
type icon struct {
	// Name is the Kumbuka icon identifier with the -simple suffix.
	Name string `json:"name"`
	// Label is the human-readable upstream icon title.
	Label string `json:"label"`
	// ViewBox is the normalized SVG view box.
	ViewBox string `json:"view_box"`
	// Paths contains the SVG path data rendered by the host.
	Paths []string `json:"paths"`
}

// svgDocument is the minimal upstream SVG structure required by the generator.
type svgDocument struct {
	// ViewBox contains the source SVG coordinate system.
	ViewBox string `xml:"viewBox,attr"`
	// Title contains the upstream human-readable icon name.
	Title string `xml:"title"`
	// Paths contains every source SVG path element.
	Paths []svgPath `xml:"path"`
}

// svgPath contains one SVG path data attribute from the upstream icon.
type svgPath struct {
	// Data is the SVG path d attribute.
	Data string `xml:"d,attr"`
}

// main generates the bounded Simple Icons resource consumed by the plugin package.
func main() {
	icons := make([]icon, 0, len(simpleicons.Names()))
	for _, name := range simpleicons.Names() {
		parsed, err := parse(name, simpleicons.Icon(name))
		if err != nil {
			panic(err)
		}
		icons = append(icons, parsed)
	}

	data, err := json.MarshalIndent(resource{Format: 1, Icons: icons}, "", "  ")
	if err != nil {
		panic(err)
	}
	data = append(data, '\n')

	filename := filepath.Join("simple-icons", "assets", "icons.json")
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filename, data, 0o644); err != nil {
		panic(err)
	}
}

// parse converts one upstream Simple Icons SVG document into a Kumbuka icon record.
func parse(name, source string) (icon, error) {
	var document svgDocument
	if err := xml.Unmarshal([]byte(source), &document); err != nil {
		return icon{}, fmt.Errorf("parse %s: %w", name, err)
	}

	label := strings.TrimSpace(document.Title)
	if label == "" {
		label = name
	}
	viewBox := strings.Join(strings.Fields(document.ViewBox), " ")
	if viewBox == "" || len(document.Paths) == 0 {
		return icon{}, fmt.Errorf("parse %s: incomplete SVG", name)
	}

	paths := make([]string, 0, len(document.Paths))
	for _, path := range document.Paths {
		if strings.TrimSpace(path.Data) == "" {
			return icon{}, fmt.Errorf("parse %s: empty path", name)
		}
		paths = append(paths, path.Data)
	}

	return icon{
		Name:    name + "-simple",
		Label:   label,
		ViewBox: viewBox,
		Paths:   paths,
	}, nil
}
