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

const defaultDeepLAPIURL = "https://api-free.deepl.com/v2/translate"

// DeepLConfig configures a DeepL ingredient translation fallback.
type DeepLConfig struct {
	APIURL          string
	APIKey          string
	Timeout         time.Duration
	SourceLanguages []Code
	HTTPClient      *http.Client
}

// DeepLProvider translates ingredient phrases with DeepL when the local dictionary misses.
type DeepLProvider struct {
	apiURL          string
	apiKey          string
	timeout         time.Duration
	sourceLanguages map[Code]struct{}
	httpClient      *http.Client
}

// NewDeepLProvider returns a DeepL ingredient provider.
func NewDeepLProvider(config DeepLConfig) *DeepLProvider {
	apiURL := strings.TrimSpace(config.APIURL)
	if apiURL == "" {
		apiURL = defaultDeepLAPIURL
	} else if !strings.HasSuffix(apiURL, "/v2/translate") {
		apiURL = strings.TrimRight(apiURL, "/") + "/v2/translate"
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

	return &DeepLProvider{
		apiURL:          apiURL,
		apiKey:          strings.TrimSpace(config.APIKey),
		timeout:         timeout,
		sourceLanguages: supported,
		httpClient:      client,
	}
}

// NormalizeIngredient translates text to the target language with DeepL.
func (p *DeepLProvider) NormalizeIngredient(ctx context.Context, text string, sourceLanguage, targetLanguage Code) (string, error) {
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

	form := url.Values{}
	form.Set("text", text)
	form.Set("source_lang", deepLLanguageCode(sourceLanguage))
	form.Set("target_lang", deepLLanguageCode(targetLanguage))

	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, p.apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return text, err
	}
	req.Header.Set("Authorization", "DeepL-Auth-Key "+p.apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return text, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return text, fmt.Errorf("deepl translate failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var decoded struct {
		Translations []struct {
			Text string `json:"text"`
		} `json:"translations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return text, err
	}
	if len(decoded.Translations) == 0 || strings.TrimSpace(decoded.Translations[0].Text) == "" {
		return text, nil
	}
	return decoded.Translations[0].Text, nil
}

func deepLLanguageCode(code Code) string {
	switch code {
	case Italian:
		return "IT"
	case English:
		return "EN-US"
	default:
		return strings.ToUpper(string(code))
	}
}
