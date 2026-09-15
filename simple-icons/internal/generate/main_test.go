package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSimpleIcon(t *testing.T) {
	icon, err := parse("github", `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0  0 24 24"><path d="M0 0H24V24H0z"/></svg>`)
	require.NoError(t, err)
	assert.Equal(t, "github-simple", icon.Name)
	assert.Equal(t, "github", icon.Label)
	assert.Equal(t, "0 0 24 24", icon.ViewBox)
	assert.Equal(t, []string{"M0 0H24V24H0z"}, icon.Paths)
}

func TestParseSimpleIconUsesTitle(t *testing.T) {
	icon, err := parse("example", `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><title>Example</title><path d="M1 1L2 2"/></svg>`)
	require.NoError(t, err)
	assert.Equal(t, "Example", icon.Label)
}

func TestParseSimpleIconRejectsIncompleteSVG(t *testing.T) {
	_, err := parse("broken", `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"></svg>`)
	require.Error(t, err)
}
