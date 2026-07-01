package language

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultGoogleTranslateAPIURL = "https://translation.googleapis.com/language/translate/v2"

// GoogleTranslateConfig configures a Google Translate ingredient translation fallback.
// It uses the same provider seam as the previous DeepL implementation so the service layer stays unchanged.
type GoogleTranslateConfig struct {
	APIURL          string
	APIKey          string
	Timeout         time.Duration
	SourceLanguages []Code
	HTTPClient      *http.Client
}

// GoogleTranslateProvider translates ingredient phrases with Google Translate when the local dictionary misses.
type GoogleTranslateProvider struct {
	apiURL          string
	apiKey          string
	timeout         time.Duration
	sourceLanguages map[Code]struct{}
	httpClient      *http.Client
}

// NewGoogleTranslateProvider returns a Google Translate ingredient provider.
func NewGoogleTranslateProvider(config GoogleTranslateConfig) *GoogleTranslateProvider {
	apiURL := strings.TrimSpace(config.APIURL)
	if apiURL == "" {
		apiURL = defaultGoogleTranslateAPIURL
	} else if !strings.HasSuffix(apiURL, "/language/translate/v2") {
		apiURL = strings.TrimRight(apiURL, "/") + "/language/translate/v2"
	}

	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	sourceLanguages := config.SourceLanguages
	if len(sourceLanguages) == 0 {
		sourceLanguages = []Code{Italian}
	}
	supported := make(map[Code]struct{}, len(sourceLanguages))
	for _, code := range sourceLanguages {
		if IsRecipeLanguage(code) && code != English {
			supported[code] = struct{}{}
		}
	}

	client := config.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	return &GoogleTranslateProvider{
		apiURL:          apiURL,
		apiKey:          strings.TrimSpace(config.APIKey),
		timeout:         timeout,
		sourceLanguages: supported,
		httpClient:      client,
	}
}

// NormalizeIngredient translates text to the target language with Google Translate.
func (p *GoogleTranslateProvider) NormalizeIngredient(ctx context.Context, text string, sourceLanguage, targetLanguage Code) (string, error) {
	if p == nil || p.apiKey == "" || strings.TrimSpace(text) == "" {
		return text, nil
	}
	if targetLanguage != English || sourceLanguage == English {
		return text, nil
	}
	if _, ok := p.sourceLanguages[sourceLanguage]; !ok {
		return text, nil
	}

	requestCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	params := url.Values{}
	params.Set("key", p.apiKey)
	params.Set("q", text)
	params.Set("source", googleTranslateLanguageCode(sourceLanguage))
	params.Set("target", googleTranslateLanguageCode(targetLanguage))

	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, p.apiURL+"?"+params.Encode(), nil)
	if err != nil {
		return text, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return text, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return text, fmt.Errorf("google translate failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var decoded struct {
		Data struct {
			Translations []struct {
				TranslatedText string `json:"translatedText"`
			} `json:"translations"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return text, err
	}
	if len(decoded.Data.Translations) == 0 || strings.TrimSpace(decoded.Data.Translations[0].TranslatedText) == "" {
		return text, nil
	}
	return decoded.Data.Translations[0].TranslatedText, nil
}

func googleTranslateLanguageCode(code Code) string {
	switch code {
	case Italian:
		return "it"
	case English:
		return "en"
	default:
		return strings.ToLower(string(code))
	}
}
