// Package main implements the Page Report plugin.
package main

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"slices"
	"strconv"
	"strings"

	sdk "github.com/kumbuka-me/sdk"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// macroOptions controls one {{pages}} report invocation.
type macroOptions struct {
	// Query is the required Kumbuka search expression.
	Query string
	// Columns contains prepared report columns in display order.
	Columns []string
	// View selects table, list, or cards presentation.
	View string
	// Sort selects the page ordering strategy.
	Sort string
	// Limit bounds the number of report rows.
	Limit int
}

// pageSource supplies page discovery and full page metadata for a report.
type pageSource interface {
	// Search returns pages matching a Kumbuka search expression.
	Search(context.Context, string, int) ([]sdk.Page, error)
	// GetPage returns complete metadata for one canonical page slug.
	GetPage(context.Context, string) (sdk.Page, error)
}

// newRenderer returns a request-bound renderer backed by the page catalog.
func newRenderer(ctx context.Context, source pageSource) func(macroOptions) (string, error) {
	return func(options macroOptions) (string, error) {
		pages, err := source.Search(ctx, options.Query, options.Limit)
		if err != nil {
			return "", err
		}
		if err := loadPageMetadata(ctx, source, pages); err != nil {
			return "", err
		}
		sortPages(pages, options.Sort)
		return render(options, pages)
	}
}

// loadPageMetadata replaces search summaries with complete page metadata in place.
func loadPageMetadata(ctx context.Context, source pageSource, pages []sdk.Page) error {
	for index := range pages {
		page, err := source.GetPage(ctx, pages[index].Slug)
		if err != nil {
			return err
		}
		pages[index] = page
	}
	return nil
}

// sortPages applies the configured deterministic report ordering.
func sortPages(pages []sdk.Page, sort string) {
	switch sort {
	case "title":
		slices.SortFunc(pages, func(left, right sdk.Page) int {
			return strings.Compare(strings.ToLower(left.Title), strings.ToLower(right.Title))
		})
	case "path":
		slices.SortFunc(pages, func(left, right sdk.Page) int {
			return strings.Compare(strings.ToLower(left.Slug), strings.ToLower(right.Slug))
		})
	case "updated":
		slices.SortFunc(pages, func(left, right sdk.Page) int {
			return right.UpdatedAt.Compare(left.UpdatedAt)
		})
	}
}

// tableData contains the template data for a rendered page-report table.
type tableData struct {
	// Columns contains prepared report columns in display order.
	Columns []columnData
	// Rows contains prepared report rows in result order.
	Rows []rowData
}

// columnData describes one page-report table column.
type columnData struct {
	// Key identifies the report field represented by this column.
	Key string
	// Label is the human-readable report column heading.
	Label string
}

// rowData contains one page-report table row.
type rowData struct {
	// Slug is the canonical page path for this report row.
	Slug string
	// Cells contains rendered cell values in column order.
	Cells []string
}

//go:embed template.gohtml
var templateSource string

var reportTemplate = template.Must(template.New("page-report").Parse(templateSource))

// render executes the selected report presentation for the resolved pages.
func render(options macroOptions, pages []sdk.Page) (string, error) {
	data := buildTableData(options.Columns, pages)
	var output strings.Builder
	if err := reportTemplate.ExecuteTemplate(&output, options.View, data); err != nil {
		return "", fmt.Errorf("render page report: %w", err)
	}
	return output.String(), nil
}

// buildTableData prepares template columns and row values for resolved pages.
func buildTableData(columns []string, pages []sdk.Page) tableData {
	data := tableData{
		Columns: make([]columnData, 0, len(columns)),
		Rows:    make([]rowData, 0, len(pages)),
	}
	for _, column := range columns {
		data.Columns = append(data.Columns, columnData{Key: column, Label: columnLabel(column)})
	}
	for _, page := range pages {
		cells := make([]string, 0, len(columns))
		for _, column := range columns {
			cells = append(cells, pageValue(page, column))
		}
		data.Rows = append(data.Rows, rowData{Slug: page.Slug, Cells: cells})
	}
	return data
}

// columnLabel returns the human-readable heading for a report column.
func columnLabel(column string) string {
	if key, ok := strings.CutPrefix(column, "property:"); ok {
		return key
	}
	return strings.ToUpper(column[:1]) + column[1:]
}

// pageValue resolves one report cell from page metadata.
func pageValue(page sdk.Page, column string) string {
	switch column {
	case "title":
		return page.Title
	case "path":
		return page.Slug
	case "status":
		return page.Status
	case "owner":
		return page.OwnerGroup
	case "updated":
		return page.UpdatedAt.Format("2006-01-02")
	case "author":
		return page.Author
	case "tags":
		return strings.Join(page.Tags, ", ")
	case "views":
		return strconv.FormatInt(page.ViewCount, 10)
	default:
		return propertyValue(page.Properties, column)
	}
}

// propertyValue resolves one property: column value case-insensitively.
func propertyValue(properties []sdk.Property, column string) string {
	key, ok := strings.CutPrefix(column, "property:")
	if !ok {
		return ""
	}
	for _, property := range properties {
		if strings.EqualFold(property.Key, key) {
			return property.Value
		}
	}
	return ""
}
