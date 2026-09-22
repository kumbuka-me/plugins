package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTableDirectiveMarkerUsesNearestPrecedingTable(t *testing.T) {
	t.Parallel()

	rendered := `<div class="table-wrapper"><table><thead><tr><th>Column 1</th></tr></thead><tbody><tr><td>Value</td></tr></tbody></table></div>` +
		`<div class="marker-wrapper"><div class="kumbuka-table-style-marker" data-table-style="{table header=gray sortable filterable}"></div></div>`

	got, err := applyTableDirectiveMarkers(rendered, tableOptions{Tables: true, TableStyles: true, TableSorting: true, TableFiltering: true})

	require.NoError(t, err)
	assert.Contains(t, got, `class="kumbuka-table-sortable kumbuka-table-filterable kumbuka-table-styled"`)
	assert.Contains(t, got, `class="table-tone-gray"`)
	assert.NotContains(t, got, `kumbuka-table-style-marker`)
}

func TestTableDimensions(t *testing.T) {
	rendered := `<table><thead><tr><th>A</th><th>B</th></tr></thead><tbody><tr><td>C</td><td>D</td></tr></tbody></table><div class="kumbuka-table-style-marker" data-table-style="{table widths=120,180 heights=32,64 cell:1,1=blue}"></div>`
	got, err := applyTableDirectiveMarkers(rendered, tableOptions{Tables: true, TableStyles: true})
	require.NoError(t, err)
	assert.Contains(t, got, `table-layout:fixed;width:300px`)
	assert.Contains(t, got, `height:64px`)
	assert.Contains(t, got, `width:180px`)
	assert.Contains(t, got, `table-tone-blue`)
	for _, input := range []string{"{table widths=-1}", "{table heights=4001}", "{table widths=1px}", "{table heights=10;display:none}"} {
		_, ok := parseTableDirective(input)
		assert.False(t, ok)
	}
}
