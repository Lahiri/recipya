package services

import (
	"context"
	"testing"

	"github.com/reaper47/recipya/internal/app"
	"github.com/reaper47/recipya/internal/language"
	"github.com/reaper47/recipya/internal/models"
)

func TestRecipeLanguage(t *testing.T) {
	italianRecipe := models.Recipe{
		Name:         "Spaghetti al pomodoro",
		Description:  "Per preparare gli spaghetti scaldate una padella con olio extravergine.",
		Ingredients:  []string{"200 g spaghetti", "2 cucchiai olio extravergine", "sale qb"},
		Instructions: []string{"Cuocete per 10 minuti in acqua bollente salata.", "Unite i pomodori e servite."},
	}
	englishRecipe := models.Recipe{
		Name:         "Tomato pasta",
		Description:  "Preheat the oven and season with salt.",
		Ingredients:  []string{"200 g pasta", "1 tablespoon olive oil", "salt"},
		Instructions: []string{"Cook until tender.", "Serve with tomato sauce."},
	}

	tests := []struct {
		name     string
		recipe   models.Recipe
		setting  string
		wantLang string
	}{
		{name: "auto detects Italian", recipe: italianRecipe, setting: "auto", wantLang: "it"},
		{name: "auto detects English", recipe: englishRecipe, setting: "auto", wantLang: "en"},
		{name: "explicit English setting wins", recipe: italianRecipe, setting: "en", wantLang: "en"},
		{name: "explicit Italian setting wins", recipe: englishRecipe, setting: "it", wantLang: "it"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recipeLanguage(tt.recipe, models.UserSettings{RecipeLanguage: tt.setting})
			if got != tt.wantLang {
				t.Fatalf("recipeLanguage() = %q, want %q", got, tt.wantLang)
			}
		})
	}
}

func TestShouldRecalculateNutrition(t *testing.T) {
	tests := []struct {
		name                 string
		isIngredientsUpdated bool
		isLanguageUpdated    bool
		want                 bool
	}{
		{name: "unchanged"},
		{name: "ingredients changed", isIngredientsUpdated: true, want: true},
		{name: "language changed", isLanguageUpdated: true, want: true},
		{name: "ingredients and language changed", isIngredientsUpdated: true, isLanguageUpdated: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRecalculateNutrition(tt.isIngredientsUpdated, tt.isLanguageUpdated)
			if got != tt.want {
				t.Fatalf("shouldRecalculateNutrition(%t, %t) = %t, want %t", tt.isIngredientsUpdated, tt.isLanguageUpdated, got, tt.want)
			}
		})
	}
}

func TestNormalizeIngredientsForNutrition(t *testing.T) {
	tests := []struct {
		name        string
		ingredients []string
		language    language.Code
		want        []string
	}{
		{
			name:        "italian ingredients are normalized to english",
			ingredients: []string{"1 cucchiaino olio evo", "300 g pomodori"},
			language:    language.Italian,
			want:        []string{"1 teaspoon extra virgin olive oil", "300 g tomatoes"},
		},
		{
			name:        "english ingredients pass through",
			ingredients: []string{"1 cucchiaino olio evo"},
			language:    language.English,
			want:        []string{"1 cucchiaino olio evo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeIngredientsForNutrition(context.Background(), tt.ingredients, tt.language, nil)
			if len(got) != len(tt.want) {
				t.Fatalf("normalizeIngredientsForNutrition() returned %d items, want %d", len(got), len(tt.want))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("normalizeIngredientsForNutrition()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestNormalizeIngredientsForNutritionUsesFallbackForDictionaryMiss(t *testing.T) {
	provider := &testIngredientProvider{translated: "chicory"}

	got := normalizeIngredientsForNutrition(context.Background(), []string{"100 g puntarelle", "300 g pomodori"}, language.Italian, provider)

	want := []string{"chicory", "300 g tomatoes"}
	if len(got) != len(want) {
		t.Fatalf("normalizeIngredientsForNutrition() returned %d items, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("normalizeIngredientsForNutrition()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if provider.calls != 1 {
		t.Fatalf("provider was called %d times, want 1 dictionary miss", provider.calls)
	}
}

func TestNewIngredientProviderUsesConfiguredTranslationProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		wantType string
	}{
		{name: "deepl", provider: "deepl", wantType: "deepl"},
		{name: "google", provider: "google", wantType: "google"},
		{name: "azure", provider: "azure", wantType: "azure"},
		{name: "unknown", provider: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := app.ConfigFile{Integrations: app.ConfigIntegrations{Translation: app.ConfigTranslation{Enabled: true, Provider: tt.provider, APIKey: "test-key"}}}
			provider := newIngredientProvider(config)
			if tt.wantType == "" {
				if provider != nil {
					t.Fatalf("newIngredientProvider() = %T, want nil", provider)
				}
				return
			}
			if provider == nil {
				t.Fatal("newIngredientProvider() returned nil, want a provider")
			}
			switch tt.provider {
			case "deepl":
				if _, ok := provider.(*language.DeepLProvider); !ok {
					t.Fatalf("newIngredientProvider() = %T, want *language.DeepLProvider", provider)
				}
			case "google":
				if _, ok := provider.(*language.GoogleTranslateProvider); !ok {
					t.Fatalf("newIngredientProvider() = %T, want *language.GoogleTranslateProvider", provider)
				}
			case "azure":
				if _, ok := provider.(*language.AzureTranslatorProvider); !ok {
					t.Fatalf("newIngredientProvider() = %T, want *language.AzureTranslatorProvider", provider)
				}
			}
		})
	}
}

type testIngredientProvider struct {
	translated string
	calls      int
}

func (p *testIngredientProvider) NormalizeIngredient(context.Context, string, language.Code, language.Code) (string, error) {
	p.calls++
	return p.translated, nil
}

func TestBulkRecipeLanguageSelection(t *testing.T) {
	recipes := models.Recipes{
		{ID: 1, Language: "en"},
		{ID: 2, Language: "en"},
		{ID: 3, Language: "it"},
	}

	tests := []struct {
		name      string
		recipeIDs []int64
		want      []int64
	}{
		{name: "empty selection means all recipes", want: []int64{1, 2, 3}},
		{name: "selected recipes only", recipeIDs: []int64{2, 99}, want: []int64{2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recipeIDsForBulkLanguageUpdate(recipes, tt.recipeIDs)
			if len(got) != len(tt.want) {
				t.Fatalf("recipeIDsForBulkLanguageUpdate() returned %d ids, want %d", len(got), len(tt.want))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("recipeIDsForBulkLanguageUpdate()[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}
