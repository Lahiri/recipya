package language

import (
	"context"
	_ "embed"
	"encoding/csv"
	"fmt"
	"strings"
)

//go:embed dictionaries/ingredients.csv
var ingredientDictionaryCSV string

// IngredientProvider normalizes ingredient text for nutrition lookup.
type IngredientProvider interface {
	NormalizeIngredient(ctx context.Context, text string, sourceLanguage, targetLanguage Code) (string, error)
}

// Normalizer normalizes ingredient text to English for FDC lookup.
type Normalizer struct {
	Fallback IngredientProvider
}

// NewNormalizer returns a dictionary-first ingredient normalizer.
func NewNormalizer(fallback IngredientProvider) Normalizer {
	return Normalizer{Fallback: fallback}
}

// NormalizeIngredient normalizes an ingredient phrase for nutrition lookup.
func (n Normalizer) NormalizeIngredient(ctx context.Context, text string, sourceLanguage, targetLanguage Code) (string, error) {
	if strings.TrimSpace(text) == "" || targetLanguage != English || sourceLanguage == English {
		return text, nil
	}

	if sourceLanguage == Italian {
		normalized := normalizeItalianIngredient(text)
		if normalized != normalizeText(text) {
			return normalized, nil
		}
	}

	if n.Fallback == nil {
		return text, nil
	}
	translated, err := n.Fallback.NormalizeIngredient(ctx, text, sourceLanguage, targetLanguage)
	if err != nil {
		return text, err
	}
	if strings.TrimSpace(translated) == "" {
		return text, nil
	}
	return translated, nil
}

var italianIngredients = mustLoadItalianIngredients(ingredientDictionaryCSV)

var italianIngredientPhrases = sortedPhraseKeys(italianIngredients)

var italianIngredientAliases = map[string]string{
	"aceto":                       "vinegar",
	"aceto balsamico":             "balsamic vinegar",
	"acqua":                       "water",
	"aglio":                       "garlic",
	"basilico":                    "basil",
	"burro":                       "butter",
	"carota":                      "carrot",
	"carote":                      "carrots",
	"cipolla":                     "onion",
	"cipolle":                     "onions",
	"farina":                      "flour",
	"latte":                       "milk",
	"lievito":                     "yeast",
	"melanzana":                   "eggplant",
	"melanzane":                   "eggplants",
	"olio":                        "oil",
	"olio d oliva":                "olive oil",
	"olio extravergine d oliva":   "extra virgin olive oil",
	"olio extra vergine d oliva":  "extra virgin olive oil",
	"olio extravergine di oliva":  "extra virgin olive oil",
	"olio extra vergine di oliva": "extra virgin olive oil",
	"panna":                       "cream",
	"parmigiano":                  "parmesan",
	"parmigiano reggiano":         "parmesan cheese",
	"pasta":                       "pasta",
	"pepe":                        "pepper",
	"peperoncino":                 "chili pepper",
	"peperoncino fresco":          "fresh chili pepper",
	"pomodori":                    "tomatoes",
	"pomodoro":                    "tomato",
	"prezzemolo":                  "parsley",
	"riso":                        "rice",
	"sale":                        "salt",
	"sale fino":                   "fine salt",
	"spaghetti":                   "spaghetti",
	"uova":                        "eggs",
	"uovo":                        "egg",
	"zucchina":                    "zucchini",
	"zucchine":                    "zucchini",
	"zucchero":                    "sugar",
}

func mustLoadItalianIngredients(data string) map[string]string {
	dictionary, err := loadIngredientDictionary(data)
	if err != nil {
		panic(err)
	}
	return dictionary
}

func loadIngredientDictionary(data string) (map[string]string, error) {
	dictionary := make(map[string]string, len(italianIngredientAliases))
	for key, value := range italianIngredientAliases {
		dictionary[normalizeText(key)] = value
	}

	reader := csv.NewReader(strings.NewReader(data))
	reader.FieldsPerRecord = 2
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 || records[0][0] != "italiano" || records[0][1] != "english" {
		return nil, fmt.Errorf("ingredient dictionary has invalid header")
	}

	for i, record := range records[1:] {
		italian := normalizeText(record[0])
		english := strings.TrimSpace(record[1])
		if italian == "" || english == "" {
			return nil, fmt.Errorf("ingredient dictionary row %d has an empty field", i+2)
		}
		dictionary[italian] = english
	}
	return dictionary, nil
}

func normalizeItalianIngredient(text string) string {
	words := strings.Fields(normalizeText(text))
	if len(words) == 0 {
		return ""
	}

	var out []string
	for i := 0; i < len(words); {
		matched := false
		for _, phrase := range italianIngredientPhrases {
			phraseWords := strings.Fields(phrase)
			if wordsMatch(words, i, phraseWords) {
				out = append(out, italianIngredients[phrase])
				i += len(phraseWords)
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		out = append(out, words[i])
		i++
	}
	return strings.Join(out, " ")
}

func wordsMatch(words []string, start int, phraseWords []string) bool {
	if start+len(phraseWords) > len(words) {
		return false
	}
	for i, word := range phraseWords {
		if words[start+i] != word {
			return false
		}
	}
	return true
}

func sortedPhraseKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && phraseLess(keys[j-1], keys[j]); j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	return keys
}

func phraseLess(left, right string) bool {
	leftWords := len(strings.Fields(left))
	rightWords := len(strings.Fields(right))
	if leftWords != rightWords {
		return leftWords < rightWords
	}
	return len(left) < len(right)
}
