package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/kumbuka-me/sdk/markdown"
)

var (
	checklistAction = regexp.MustCompile(`^toggle-([0-9]+)-([01])-([a-f0-9]{16})$`)
	taskMarker      = regexp.MustCompile(`^((?:[ \t]{0,3}>[ \t]?)*[ \t]*(?:[-+*]|[0-9]+[.)])[ \t]+\[)([ xX])(\])`)
)

// checklistFingerprint binds a rendered action to the task count, order, and
// states that produced it. A stale page therefore cannot target a shifted row.
func checklistFingerprint(states []bool) string {
	encoded := make([]byte, len(states))
	for index, checked := range states {
		if checked {
			encoded[index] = '1'
		} else {
			encoded[index] = '0'
		}
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:8])
}

// toggleTaskMarker changes one task marker only when the rendered checklist state still matches.
func toggleTaskMarker(source string, target int, expectedChecked bool, expectedFingerprint string) (string, error) {
	lines := strings.SplitAfter(source, "\n")
	markers := scanTaskMarkers(lines)
	states := make([]bool, len(markers))
	for index, marker := range markers {
		states[index] = marker.checked
	}
	if checklistFingerprint(states) != expectedFingerprint {
		return "", fmt.Errorf("checklist changed; reload the page")
	}
	if target < 0 || target >= len(markers) {
		return "", fmt.Errorf("checklist item no longer exists")
	}

	marker := markers[target]
	if marker.checked != expectedChecked {
		return "", fmt.Errorf("checklist item changed; reload the page")
	}
	replacement := "x"
	if marker.checked {
		replacement = " "
	}
	line := lines[marker.line]
	lines[marker.line] = line[:marker.offset] + replacement + line[marker.offset+1:]
	return strings.Join(lines, ""), nil
}

// taskMarkerPosition identifies one editable checkbox marker in the original Markdown.
type taskMarkerPosition struct {
	// line is the marker's zero-based source line index.
	line int
	// offset is the byte offset of the checkbox state within that line.
	offset int
	// checked records whether the source checkbox is selected.
	checked bool
}

// scanTaskMarkers locates checkbox states once, excluding fenced code examples.
func scanTaskMarkers(lines []string) []taskMarkerPosition {
	var markers []taskMarkerPosition
	fence := ""
	for index, line := range lines {
		content := strings.TrimRight(line, "\r\n")
		trimmed := strings.TrimLeft(content, " \t")
		if fence != "" {
			if markdown.Closes(trimmed, fence) {
				fence = ""
			}
			continue
		}
		if opening := markdown.Fence(trimmed); opening != "" {
			fence = opening
			continue
		}

		match := taskMarker.FindStringSubmatchIndex(content)
		if match != nil {
			markers = append(markers, taskMarkerPosition{
				line: index, offset: match[4], checked: content[match[4]:match[5]] != " ",
			})
		}
	}
	return markers
}
