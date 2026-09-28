package localize

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranslationsCoverEnglishAndGerman(t *testing.T) {
	english := translations["en"]
	german := translations["de"]
	require.NotEmpty(t, english)
	require.NotEmpty(t, german)

	for key := range english {
		assert.Contains(t, german, key, "German translation missing for %q", key)
	}
	for key := range german {
		assert.Contains(t, english, key, "English translation missing for %q", key)
	}
}

func TestForUsesEnglishByDefaultAndGermanWhenRequested(t *testing.T) {
	assert.Equal(t, "en", For("").Locale())
	assert.Equal(t, "Continue working", For("").Text("continue.title"))
	assert.Equal(t, "en", For("fr").Locale())
	assert.Equal(t, "Continue working", For("fr").Text("continue.title"))
	assert.Equal(t, "de", For("de").Locale())
	assert.Equal(t, "Weiterarbeiten", For("de").Text("continue.title"))
}
