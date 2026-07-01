// Package language provides recipe language detection and ingredient normalization helpers.
package language

import "strings"

// Code identifies a supported recipe language or setting behavior.
type Code string

const (
	Auto    Code = "auto"
	English Code = "en"
	Italian Code = "it"
)

// Option describes a language choice for settings or recipe forms.
type Option struct {
	Code  Code
	Label string
}

// SettingOptions returns all supported user setting values.
func SettingOptions() []Option {
	return []Option{
		{Code: Auto, Label: "Auto"},
		{Code: English, Label: "English"},
		{Code: Italian, Label: "Italian"},
	}
}

// RecipeOptions returns the concrete languages that can be stored on recipes.
func RecipeOptions() []Option {
	return []Option{
		{Code: English, Label: "English"},
		{Code: Italian, Label: "Italian"},
	}
}

// Parse returns a supported language code and whether the input was valid.
func Parse(value string) (Code, bool) {
	switch Code(strings.ToLower(strings.TrimSpace(value))) {
	case Auto:
		return Auto, true
	case English:
		return English, true
	case Italian:
		return Italian, true
	default:
		return English, false
	}
}

// NormalizeSetting returns a valid user setting language code.
func NormalizeSetting(value string) Code {
	code, ok := Parse(value)
	if !ok {
		return Auto
	}
	return code
}

// NormalizeRecipe returns a concrete recipe language code.
func NormalizeRecipe(value string) Code {
	code, ok := Parse(value)
	if !ok || code == Auto {
		return English
	}
	return code
}

// Label returns the display label for code.
func Label(code Code) string {
	switch code {
	case Auto:
		return "Auto"
	case Italian:
		return "Italian"
	default:
		return "English"
	}
}

// IsRecipeLanguage reports whether code can be stored on a recipe.
func IsRecipeLanguage(code Code) bool {
	return code == English || code == Italian
}
