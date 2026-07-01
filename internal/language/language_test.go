package language

import "testing"

func TestNormalizeSetting(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  Code
	}{
		{name: "auto", value: "auto", want: Auto},
		{name: "english", value: "en", want: English},
		{name: "italian", value: "it", want: Italian},
		{name: "empty defaults to auto", value: "", want: Auto},
		{name: "invalid defaults to auto", value: "de", want: Auto},
		{name: "trims case", value: " IT ", want: Italian},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeSetting(tt.value); got != tt.want {
				t.Fatalf("NormalizeSetting(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestNormalizeRecipe(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  Code
	}{
		{name: "english", value: "en", want: English},
		{name: "italian", value: "it", want: Italian},
		{name: "auto becomes concrete default", value: "auto", want: English},
		{name: "invalid becomes concrete default", value: "fr", want: English},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeRecipe(tt.value); got != tt.want {
				t.Fatalf("NormalizeRecipe(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestLabel(t *testing.T) {
	tests := []struct {
		code Code
		want string
	}{
		{code: Auto, want: "Auto"},
		{code: English, want: "English"},
		{code: Italian, want: "Italian"},
		{code: Code("unknown"), want: "English"},
	}

	for _, tt := range tests {
		if got := Label(tt.code); got != tt.want {
			t.Fatalf("Label(%q) = %q, want %q", tt.code, got, tt.want)
		}
	}
}
