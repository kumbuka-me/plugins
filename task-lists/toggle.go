package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
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

// toggleTaskMarker changes one task marker in document order while ignoring
// fenced code blocks. The state fingerprint rejects stale rendered controls.
func toggleTaskMarker(source string, target int, expectedChecked bool, expectedFingerprint string) (string, error) {
	lines := strings.SplitAfter(source, "\n")
	states := taskMarkerStates(lines)
	if checklistFingerprint(states) != expectedFingerprint {
		return "", fmt.Errorf("checklist changed; reload the page")
	}

	itemIndex := 0
	fence := byte(0)
	fenceLength := 0
	for lineIndex, line := range lines {
		content := strings.TrimSuffix(line, "\n")
		trimmed := strings.TrimLeft(content, " \t")
		if marker, length := fenceMarker(trimmed); marker != 0 {
			if fence == 0 {
				fence, fenceLength = marker, length
			} else if marker == fence && length >= fenceLength {
				fence, fenceLength = 0, 0
			}
			continue
		}
		if fence != 0 {
			continue
		}
		match := taskMarker.FindStringSubmatchIndex(content)
		if match == nil {
			continue
		}
		if itemIndex == target {
			checked := content[match[4]:match[5]] != " "
			if checked != expectedChecked {
				return "", fmt.Errorf("checklist item changed; reload the page")
			}
			replacement := "x"
			if checked {
				replacement = " "
			}
			lines[lineIndex] = content[:match[4]] + replacement + content[match[5]:] + strings.TrimPrefix(line, content)
			return strings.Join(lines, ""), nil
		}
		itemIndex++
	}
	return "", fmt.Errorf("checklist item no longer exists")
}

func taskMarkerStates(lines []string) []bool {
	var states []bool
	fence := byte(0)
	fenceLength := 0
	for _, line := range lines {
		content := strings.TrimSuffix(line, "\n")
		if marker, length := fenceMarker(strings.TrimLeft(content, " \t")); marker != 0 {
			if fence == 0 {
				fence, fenceLength = marker, length
			} else if marker == fence && length >= fenceLength {
				fence, fenceLength = 0, 0
			}
			continue
		}
		if fence != 0 {
			continue
		}
		match := taskMarker.FindStringSubmatchIndex(content)
		if match != nil {
			states = append(states, content[match[4]:match[5]] != " ")
		}
	}
	return states
}

func fenceMarker(line string) (byte, int) {
	if len(line) < 3 || (line[0] != '`' && line[0] != '~') {
		return 0, 0
	}
	marker := line[0]
	length := 1
	for length < len(line) && line[length] == marker {
		length++
	}
	if length < 3 {
		return 0, 0
	}
	return marker, length
}
