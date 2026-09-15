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
