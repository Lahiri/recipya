package language

import "strings"

// RecipeText contains the recipe text used for language detection.
type RecipeText struct {
	Name         string
	Description  string
	Ingredients  []string
	Instructions []string
}

// Detection is the result of local recipe language detection.
type Detection struct {
	Language   Code
	Confidence float64
}

var englishSignals = map[string]int{
	"a": 1, "add": 4, "and": 2, "bake": 4, "boil": 4, "bowl": 3, "chopped": 3,
	"cook": 4, "cream": 2, "cup": 2, "cups": 2, "diced": 3, "for": 1, "garlic": 3,
	"heat": 4, "in": 1, "ingredients": 2, "into": 2, "mix": 4, "minutes": 3, "oil": 2,
	"olive": 2, "onion": 3, "over": 2, "pepper": 3, "preheat": 5, "recipe": 2, "salt": 2,
	"sauce": 2, "serve": 4, "simmer": 4, "slice": 3, "stir": 4, "tablespoon": 3,
	"teaspoon": 3, "the": 2, "to": 1, "tomato": 3, "tomatoes": 3, "until": 4,
	"water": 2, "with": 2,
}

var italianSignals = map[string]int{
	"a": 1, "acqua": 2, "aggiungere": 5, "aggiungete": 5, "aglio": 3, "al": 2, "alla": 2,
	"bollente": 4, "cipolla": 3, "con": 2, "cottura": 4, "cuocere": 5, "cuocete": 5,
	"cucchiaio": 3, "cucchiaino": 3, "di": 2, "e": 1, "farina": 3, "fettine": 3, "fino": 2,
	"forno": 3, "gli": 2, "il": 2, "in": 1, "ingredienti": 2, "la": 2, "le": 2, "minuti": 4,
	"olio": 3, "padella": 4, "pepe": 3, "per": 2, "pomodori": 3, "preparare": 5,
	"prezzemolo": 3, "qb": 2, "q": 1, "ricetta": 3, "sale": 3, "scaldate": 5,
	"servite": 5, "spaghetti": 3, "tagliate": 5, "unite": 5, "uova": 3, "versate": 5,
}

var englishPhrases = map[string]int{
	"bring to a boil":  8,
	"cook until":       8,
	"heat the oil":     8,
	"preheat the oven": 9,
	"season with":      7,
}

var italianPhrases = map[string]int{
	"acqua bollente salata": 9,
	"cuocete per":           8,
	"olio extravergine":     7,
	"per preparare":         9,
	"scaldate una padella":  9,
	"tagliate a":            7,
}

// DetectRecipe detects whether recipe text is English or Italian.
func DetectRecipe(text RecipeText) Detection {
	englishScore, italianScore := scoreText(text.Name, 1)
	descriptionEnglish, descriptionItalian := scoreText(text.Description, 4)
	instructionsEnglish, instructionsItalian := scoreText(strings.Join(text.Instructions, " "), 4)
	if descriptionEnglish+descriptionItalian+instructionsEnglish+instructionsItalian == 0 {
		return Detection{Language: English, Confidence: 0}
	}
	addScores(&englishScore, &italianScore, descriptionEnglish, descriptionItalian)
	addScores(&englishScore, &italianScore, instructionsEnglish, instructionsItalian)
	englishDelta, italianDelta := scoreText(strings.Join(text.Ingredients, " "), 2)
	addScores(&englishScore, &italianScore, englishDelta, italianDelta)

	if italianScore > englishScore {
		return Detection{Language: Italian, Confidence: confidence(italianScore, englishScore)}
	}
	return Detection{Language: English, Confidence: confidence(englishScore, italianScore)}
}

func addScores(englishScore, italianScore *int, englishDelta, italianDelta int) {
	*englishScore += englishDelta
	*italianScore += italianDelta
}

func scoreText(text string, weight int) (int, int) {
	text = normalizeText(text)
	if text == "" {
		return 0, 0
	}

	englishScore, italianScore := 0, 0
	for phrase, score := range englishPhrases {
		if strings.Contains(text, phrase) {
			englishScore += score * weight
		}
	}
	for phrase, score := range italianPhrases {
		if strings.Contains(text, phrase) {
			italianScore += score * weight
		}
	}
	for _, word := range strings.Fields(text) {
		englishScore += englishSignals[word] * weight
		italianScore += italianSignals[word] * weight
	}
	return englishScore, italianScore
}

func confidence(winningScore, losingScore int) float64 {
	if winningScore == 0 {
		return 0
	}
	return float64(winningScore-losingScore) / float64(winningScore)
}
