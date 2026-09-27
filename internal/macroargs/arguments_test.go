package macroargs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUnique(t *testing.T) {
	t.Parallel()

	arguments, ok := ParseUnique(`query="status:active owner:Platform" limit=12 columns=title,status`)
	require.True(t, ok)
	assert.Equal(t, map[string]string{
		"query":   "status:active owner:Platform",
		"limit":   "12",
		"columns": "title,status",
	}, arguments)
}

func TestParseUniqueRejectsDuplicateOrMalformedArguments(t *testing.T) {
	t.Parallel()

	for _, value := range []string{`id=one id=two`, `id="unterminated`, `=value`, `id`} {
		_, ok := ParseUnique(value)
		assert.False(t, ok, value)
	}
}

func TestNextQuoted(t *testing.T) {
	t.Parallel()

	name, value, remaining, ok := NextQuoted(`show-provider="true" note="line 1: review"`)
	require.True(t, ok)
	assert.Equal(t, "show-provider", name)
	assert.Equal(t, "true", value)
	assert.Equal(t, ` note="line 1: review"`, remaining)
}

func TestNextQuotedRejectsMalformedAttributes(t *testing.T) {
	t.Parallel()

	for _, value := range []string{`name=value`, `name="unterminated`, `="value"`, `name="value"next="x"`} {
		_, _, _, ok := NextQuoted(value)
		assert.False(t, ok, value)
	}
}
