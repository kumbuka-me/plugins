package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostprocessTablesKeepsLargeUnrelatedHTMLOutsideTableFragments(t *testing.T) {
	t.Parallel()

	prefix := strings.Repeat(`<p>before content</p>`, 20_000)
	suffix := strings.Repeat(`<p>after content</p>`, 20_000)
	source := prefix +
		`<table><thead><tr><th>Service</th><th>Status</th></tr></thead><tbody><tr><td>API</td><td>Healthy</td></tr></tbody></table>` +
		`<div class="kumbuka-table-style-marker" data-table-style="{table header=accent col:2=info sortable filterable}"></div>` +
		suffix

	got, err := processRenderedTables(source, tableOptions{Tables: true, TableStyles: true, TableSorting: true, TableFiltering: true}, true)

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(got, prefix), "large prefix changed")
	assert.True(t, strings.HasSuffix(got, suffix), "large suffix changed")
	assert.Contains(t, got, `class="kumbuka-plugin-block"`)
	assert.Contains(t, got, `data-kumbuka-module="interactive"`)
	assert.Contains(t, got, `kumbuka-table-styled`)
	assert.Contains(t, got, `table-tone-accent`)
	assert.Contains(t, got, `table-tone-info`)
	assert.NotContains(t, got, `kumbuka-table-style-marker`)
}

func TestPostprocessTablesAppliesDirectivesToNearestPrecedingTables(t *testing.T) {
	t.Parallel()

	source := `<table><thead><tr><th>First</th></tr></thead><tbody><tr><td>A</td></tr></tbody></table>` +
		`<div class="kumbuka-table-style-marker" data-table-style="{table header=blue}"></div>` +
		`<p>between</p>` +
		`<table><thead><tr><th>Second</th></tr></thead><tbody><tr><td>B</td></tr></tbody></table>` +
		`<div><div class="kumbuka-table-style-marker" data-table-style="{table header=green sortable}"></div></div>`

	got, err := processRenderedTables(source, tableOptions{Tables: true, TableStyles: true, TableSorting: true}, true)

	require.NoError(t, err)
	assert.Equal(t, 2, strings.Count(got, `class="kumbuka-plugin-block"`))
	assert.Contains(t, got, `table-tone-blue`)
	assert.Contains(t, got, `table-tone-green`)
	assert.Contains(t, got, `kumbuka-table-sortable`)
	assert.NotContains(t, got, `kumbuka-table-style-marker`)
}

func TestPostprocessTablesConsumesRenderedDirectiveParagraph(t *testing.T) {
	t.Parallel()

	source := `<table><thead><tr><th>Service</th><th>Status</th></tr></thead><tbody><tr><td>API</td><td>Healthy</td></tr></tbody></table>` +
		"\n\n" + `<p>{table header=accent col:2=info sortable filterable}</p>`

	got, err := processRenderedTables(source, tableOptions{Tables: true, TableStyles: true, TableSorting: true, TableFiltering: true}, true)

	require.NoError(t, err)
	assert.Contains(t, got, `kumbuka-table-styled`)
	assert.Contains(t, got, `table-tone-accent`)
	assert.Contains(t, got, `table-tone-info`)
	assert.Contains(t, got, `kumbuka-table-sortable`)
	assert.Contains(t, got, `kumbuka-table-filterable`)
	assert.NotContains(t, got, `{table`)
}

func TestPostprocessTablesLeavesUnrelatedDirectiveParagraphVisible(t *testing.T) {
	t.Parallel()

	source := `<table><thead><tr><th>Service</th></tr></thead><tbody><tr><td>API</td></tr></tbody></table>` +
		`<p>Not part of the table.</p><p>{table header=accent sortable}</p>`

	got, err := processRenderedTables(source, tableOptions{Tables: true, TableStyles: true, TableSorting: true}, true)

	require.NoError(t, err)
	assert.Contains(t, got, `<p>{table header=accent sortable}</p>`)
	assert.NotContains(t, got, `kumbuka-table-styled`)
	assert.NotContains(t, got, `kumbuka-table-sortable`)
}

func TestPostprocessTablesLeavesInactiveDirectiveParagraphVisible(t *testing.T) {
	t.Parallel()

	source := `<table><thead><tr><th>Service</th></tr></thead><tbody><tr><td>API</td></tr></tbody></table><p>{table sortable}</p>`

	got, err := processRenderedTables(source, tableOptions{Tables: true}, true)

	require.NoError(t, err)
	assert.Contains(t, got, `<p>{table sortable}</p>`)
}

func TestPostprocessTablesStreamingPathPreservesExistingAttributes(t *testing.T) {
	t.Parallel()

	source := `<table class="existing" data-test="yes"><thead><tr><th class="heading">Service</th><th>Status</th></tr></thead><tbody><tr><td>API</td><td>Healthy</td></tr></tbody></table>` +
		`<div class="kumbuka-table-style-marker" data-table-style="{table header=accent col:2=info sortable filterable}"></div>`

	got, err := processRenderedTables(source, tableOptions{Tables: true, TableStyles: true, TableSorting: true, TableFiltering: true}, true)

	require.NoError(t, err)
	assert.Contains(t, got, `class="existing kumbuka-table-sortable kumbuka-table-filterable kumbuka-table-styled"`)
	assert.Contains(t, got, `data-test="yes"`)
	assert.Contains(t, got, `class="heading table-tone-accent"`)
	assert.Contains(t, got, `table-tone-info`)
	assert.Contains(t, got, `data-kumbuka-module="interactive"`)
}
