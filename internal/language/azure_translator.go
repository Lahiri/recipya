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

const defaultAzureTranslatorAPIURL = "https://api.cognitive.microsofttranslator.com/translate"

// AzureTranslatorConfig configures an Azure AI Translator ingredient translation fallback.
type AzureTranslatorConfig struct {
	APIURL          string
	APIKey          string
	Region          string
	Timeout         time.Duration
	SourceLanguages []Code
	HTTPClient      *http.Client
}

// AzureTranslatorProvider translates ingredient phrases with Azure AI Translator when the local dictionary misses.
type AzureTranslatorProvider struct {
	apiURL          string
	apiKey          string
	region          string
	timeout         time.Duration
	sourceLanguages map[Code]struct{}
	httpClient      *http.Client
}

// NewAzureTranslatorProvider returns an Azure AI Translator ingredient provider.
func NewAzureTranslatorProvider(config AzureTranslatorConfig) *AzureTranslatorProvider {
	apiURL := strings.TrimSpace(config.APIURL)
	if apiURL == "" {
		apiURL = defaultAzureTranslatorAPIURL
	} else if !strings.HasSuffix(apiURL, "/translate") {
		apiURL = strings.TrimRight(apiURL, "/") + "/translate"
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

	return &AzureTranslatorProvider{
		apiURL:          apiURL,
		apiKey:          strings.TrimSpace(config.APIKey),
		region:          strings.TrimSpace(config.Region),
		timeout:         timeout,
		sourceLanguages: supported,
		httpClient:      client,
	}
}

// NormalizeIngredient translates text to the target language with Azure AI Translator.
func (p *AzureTranslatorProvider) NormalizeIngredient(ctx context.Context, text string, sourceLanguage, targetLanguage Code) (string, error) {
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
	params.Set("api-version", "3.0")
	params.Set("from", azureTranslatorLanguageCode(sourceLanguage))
	params.Set("to", azureTranslatorLanguageCode(targetLanguage))

	body := fmt.Sprintf(`[{"text":"%s"}]`, strings.ReplaceAll(text, `"`, `\"`))
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, p.apiURL+"?"+params.Encode(), strings.NewReader(body))
	if err != nil {
		return text, err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	if p.region != "" {
		req.Header.Set("Ocp-Apim-Subscription-Region", p.region)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return text, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return text, fmt.Errorf("azure translator failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	var decoded []struct {
		Translations []struct {
			Text string `json:"text"`
		} `json:"translations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return text, err
	}
	if len(decoded) == 0 || len(decoded[0].Translations) == 0 || strings.TrimSpace(decoded[0].Translations[0].Text) == "" {
		return text, nil
	}
	return decoded[0].Translations[0].Text, nil
}

func azureTranslatorLanguageCode(code Code) string {
	switch code {
	case Italian:
		return "it"
	case English:
		return "en"
	default:
		return strings.ToLower(string(code))
	}
}
