// Package ascii provides shared ASCII character classification helpers.
package ascii

// IsNumeric reports whether character is an ASCII decimal digit.
func IsNumeric(character byte) bool {
	return character >= '0' && character <= '9'
}

// IsAlphabetic reports whether character is an ASCII letter.
func IsAlphabetic(character byte) bool {
	return character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z'
}

// IsAlphaNumeric reports whether character is an ASCII letter or decimal digit.
func IsAlphaNumeric(character byte) bool {
	return IsAlphabetic(character) || IsNumeric(character)
}

// IsLowerAlphabetic reports whether character is a lowercase ASCII letter.
func IsLowerAlphabetic(character byte) bool {
	return character >= 'a' && character <= 'z'
}

// IsLowerAlphaNumeric reports whether character is a lowercase ASCII letter or decimal digit.
func IsLowerAlphaNumeric(character byte) bool {
	return IsLowerAlphabetic(character) || IsNumeric(character)
}

// IsLowerHex reports whether character is a lowercase hexadecimal digit.
func IsLowerHex(character byte) bool {
	return IsNumeric(character) || character >= 'a' && character <= 'f'
}
