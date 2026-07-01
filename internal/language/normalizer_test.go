package language

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNormalizerNormalizeIngredient(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		source Code
		want   string
	}{
		{name: "english passthrough", text: "300 g tomatoes", source: English, want: "300 g tomatoes"},
		{name: "simple italian plural", text: "300 g pomodori", source: Italian, want: "300 g tomatoes"},
		{name: "phrase before word", text: "Olio extravergine d'oliva 70 g", source: Italian, want: "extra virgin olive oil 70 g"},
		{name: "fresh chili pepper phrase", text: "Peperoncino fresco 1", source: Italian, want: "fresh chili pepper 1"},
		{name: "unknown italian falls back unchanged without provider", text: "q.b. puntarelle", source: Italian, want: "q.b. puntarelle"},
	}

	normalizer := NewNormalizer(nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizer.NormalizeIngredient(context.Background(), tt.text, tt.source, English)
			if err != nil {
				t.Fatalf("NormalizeIngredient(%q) returned error: %v", tt.text, err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeIngredient(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestLoadIngredientDictionary(t *testing.T) {
	dictionary, err := loadIngredientDictionary(ingredientDictionaryCSV)
	if err != nil {
		t.Fatalf("loadIngredientDictionary returned error: %v", err)
	}

	tests := []struct {
		italian string
		want    string
	}{
		{italian: "olio evo", want: "extra virgin olive oil"},
		{italian: "cucchiaino olio evo", want: "teaspoon extra virgin olive oil"},
		{italian: "branzino", want: "sea bass"},
		{italian: "zucchina (piccola)", want: "small zucchini"},
		{italian: "pomodori", want: "tomatoes"},
	}

	for _, tt := range tests {
		t.Run(tt.italian, func(t *testing.T) {
			got := dictionary[normalizeText(tt.italian)]
			if got != tt.want {
				t.Fatalf("dictionary[%q] = %q, want %q", tt.italian, got, tt.want)
			}
		})
	}
}

func TestNormalizerUsesFallbackForDictionaryMiss(t *testing.T) {
	normalizer := NewNormalizer(fakeProvider{translated: "chicory"})

	got, err := normalizer.NormalizeIngredient(context.Background(), "puntarelle", Italian, English)
	if err != nil {
		t.Fatalf("NormalizeIngredient with fallback returned error: %v", err)
	}
	if got != "chicory" {
		t.Fatalf("NormalizeIngredient fallback = %q, want %q", got, "chicory")
	}
}

func TestNormalizerReturnsOriginalOnFallbackError(t *testing.T) {
	wantErr := errors.New("provider unavailable")
	normalizer := NewNormalizer(fakeProvider{err: wantErr})

	got, err := normalizer.NormalizeIngredient(context.Background(), "puntarelle", Italian, English)
	if !errors.Is(err, wantErr) {
		t.Fatalf("NormalizeIngredient error = %v, want %v", err, wantErr)
	}
	if got != "puntarelle" {
		t.Fatalf("NormalizeIngredient on fallback error = %q, want original", got)
	}
}

type fakeProvider struct {
	translated string
	err        error
}

func (p fakeProvider) NormalizeIngredient(context.Context, string, Code, Code) (string, error) {
	return p.translated, p.err
}

func TestGoogleTranslateProviderNormalizeIngredient(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/language/translate/v2" {
			t.Fatalf("request path = %q, want /language/translate/v2", r.URL.Path)
		}
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"translations":[{"translatedText":"chicory"}]}}`))
	}))
	defer server.Close()

	provider := NewGoogleTranslateProvider(GoogleTranslateConfig{APIURL: server.URL, APIKey: "test-key", SourceLanguages: []Code{Italian}})

	got, err := provider.NormalizeIngredient(context.Background(), "100 g puntarelle", Italian, English)
	if err != nil {
		t.Fatalf("NormalizeIngredient returned error: %v", err)
	}
	if got != "chicory" {
		t.Fatalf("NormalizeIngredient = %q, want chicory", got)
	}
	if gotQuery.Get("key") != "test-key" || gotQuery.Get("q") != "100 g puntarelle" || gotQuery.Get("source") != "it" || gotQuery.Get("target") != "en" {
		t.Fatalf("request query = %#v", gotQuery)
	}
}

func TestGoogleTranslateProviderDisabledWithoutAPIKey(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()

	provider := NewGoogleTranslateProvider(GoogleTranslateConfig{APIURL: server.URL, SourceLanguages: []Code{Italian}})

	got, err := provider.NormalizeIngredient(context.Background(), "puntarelle", Italian, English)
	if err != nil {
		t.Fatalf("NormalizeIngredient returned error: %v", err)
	}
	if got != "puntarelle" {
		t.Fatalf("NormalizeIngredient = %q, want original", got)
	}
	if called {
		t.Fatal("Google Translate provider called HTTP server without API key")
	}
}

func TestGoogleTranslateProviderReturnsOriginalOnHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusTooManyRequests)
	}))
	defer server.Close()

	provider := NewGoogleTranslateProvider(GoogleTranslateConfig{APIURL: server.URL, APIKey: "test-key", SourceLanguages: []Code{Italian}})

	got, err := provider.NormalizeIngredient(context.Background(), "puntarelle", Italian, English)
	if err == nil {
		t.Fatal("NormalizeIngredient returned nil error, want HTTP error")
	}
	if got != "puntarelle" {
		t.Fatalf("NormalizeIngredient = %q, want original", got)
	}
}
