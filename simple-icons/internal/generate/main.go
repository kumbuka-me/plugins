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

type resource struct {
	Format int    `json:"format"`
	Source string `json:"source"`
	Icons  []icon `json:"icons"`
}

type icon struct {
	Name    string   `json:"name"`
	Label   string   `json:"label"`
	ViewBox string   `json:"view_box"`
	Paths   []string `json:"paths"`
}

type svgDocument struct {
	ViewBox string    `xml:"viewBox,attr"`
	Title   string    `xml:"title"`
	Paths   []svgPath `xml:"path"`
}

type svgPath struct {
	Data string `xml:"d,attr"`
}

func main() {
	icons := make([]icon, 0, len(simpleicons.Names()))
	for _, name := range simpleicons.Names() {
		parsed, err := parse(name, simpleicons.Icon(name))
		if err != nil {
			panic(err)
		}
		icons = append(icons, parsed)
	}

	data, err := json.MarshalIndent(resource{Format: 1, Source: "Simple Icons", Icons: icons}, "", "  ")
	if err != nil {
		panic(err)
	}
	data = append(data, '\n')

	filename := filepath.Join("simple-icons", "assets", "icons.json")
	if err := os.WriteFile(filename, data, 0o644); err != nil {
		panic(err)
	}
}

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
