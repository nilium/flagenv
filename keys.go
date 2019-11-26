package flagenv

import (
	"path"
	"strings"
	"unicode"
)

// WithPrefix returns a KeyFunc that prefixes all keys with the given prefix string before passing
// them to the keyfn.
func WithPrefix(prefix string, keyfn KeyFunc) KeyFunc {
	return func(key string) string {
		return keyfn(prefix + key)
	}
}

// Uppercased returns a KeyFunc that uppercases all keys before passing them to the keyfn.
func Uppercased(keyfn KeyFunc) KeyFunc {
	return func(key string) string {
		return strings.ToUpper(keyfn(key))
	}
}

// Lowercased returns a KeyFunc that lowercases all keys before passing them to the keyfn.
func Lowercased(keyfn KeyFunc) KeyFunc {
	return func(key string) string {
		return strings.ToLower(keyfn(key))
	}
}

// Simple casing functions. Most are not guaranteed to be unicode-friendly key functions for simple casing.

func isAlnum(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r)
}

// SnakeCase is a KeyFunc that converts names to snake_case (i.e., underscore-separated). It does
// this by replacing any non-alphanumeric runes with underscores.
func SnakeCase(name string) string {
	var last rune = -1
	runeMap := func(r rune) (m rune) {
		defer func() { last = r }()
		if isAlnum(r) {
			return r
		}
		m = '_'
		if last == m {
			return -1
		}
		return m
	}
	return strings.Map(runeMap, name)
}

// PathLike is a KeyFunc that converts names to path/case (i.e., slash-separated). It does this by
// splitting the input name on any non-alphanumeric and non-hyphen rune and rejoining them as
// a slash-separated (/) path.
func PathLike(name string) string {
	return path.Join(strings.FieldsFunc(name, func(r rune) bool {
		return !isAlnum(r) && r != '-'
	})...)
}

// DotCase is a KeyFunc that converts names to dot.case (i.e., dot-separated). It does
// this by replacing any non-alphanumeric runes with dots. Hyphens are preserved
func DotCase(name string) string {
	var last rune = -1
	runeMap := func(r rune) (m rune) {
		defer func() { last = r }()
		if r == '-' || isAlnum(r) {
			return r
		}
		m = '.'
		if last == m {
			return -1
		}
		return m
	}
	return strings.Map(runeMap, name)
}

// KebabCase is a KeyFunc that converts names to kebab-case (i.e., hyphen-separated). It does this
// by replacing any non-alphanumeric runes with hyphens.
func KebabCase(name string) string {
	var last rune = -1
	runeMap := func(r rune) (m rune) {
		defer func() { last = r }()
		if isAlnum(r) {
			return r
		}
		m = '-'
		if last == m {
			return -1
		}
		return m
	}
	return strings.Map(runeMap, name)
}
