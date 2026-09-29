package ascii

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCharacterClasses(t *testing.T) {
	t.Parallel()

	assert.True(t, IsNumeric('0'))
	assert.True(t, IsNumeric('9'))
	assert.False(t, IsNumeric('a'))

	assert.True(t, IsAlphabetic('a'))
	assert.True(t, IsAlphabetic('Z'))
	assert.False(t, IsAlphabetic('0'))

	assert.True(t, IsAlphaNumeric('a'))
	assert.True(t, IsAlphaNumeric('7'))
	assert.False(t, IsAlphaNumeric('-'))

	assert.True(t, IsLowerAlphabetic('a'))
	assert.False(t, IsLowerAlphabetic('A'))

	assert.True(t, IsLowerAlphaNumeric('z'))
	assert.True(t, IsLowerAlphaNumeric('4'))
	assert.False(t, IsLowerAlphaNumeric('Z'))

	assert.True(t, IsLowerHex('0'))
	assert.True(t, IsLowerHex('f'))
	assert.False(t, IsLowerHex('F'))
	assert.False(t, IsLowerHex('g'))
}
