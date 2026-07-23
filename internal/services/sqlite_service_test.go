package services

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/reaper47/recipya/internal/app"
	"github.com/reaper47/recipya/internal/language"
	"github.com/reaper47/recipya/internal/models"
	_ "modernc.org/sqlite"
)

func TestEnsureRecipeHighlightColumnAddsItWhenMissing(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE recipes (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			image TEXT DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' ||
									substr(lower(hex(randomblob(2))), 2) || '-a' ||
									substr(lower(hex(randomblob(2))), 2) || '-%' ||
									substr(lower(hex(randomblob(6))), 2)),
			yield INTEGER,
			url TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("create recipes table: %v", err)
	}

	if err := ensureRecipeHighlightColumn(context.Background(), db); err != nil {
		t.Fatalf("ensureRecipeHighlightColumn: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('recipes') WHERE name = 'highlighted'`).Scan(&count); err != nil {
		t.Fatalf("query pragma_table_info: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected highlighted column to exist, got %d rows", count)
	}
}

func TestRecipeHighlightedMigrationIsIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE recipes (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			image TEXT DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' ||
								 substr(lower(hex(randomblob(2))), 2) || '-a' ||
								 substr(lower(hex(randomblob(2))), 2) || '-%' ||
								 substr(lower(hex(randomblob(6))), 2)),
			yield INTEGER,
			url TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("create recipes table: %v", err)
	}

	if _, err := db.Exec(`ALTER TABLE recipes ADD COLUMN highlighted BOOLEAN NOT NULL DEFAULT 0`); err != nil {
		t.Fatalf("pre-seed highlighted column: %v", err)
	}

	migrationBytes, err := os.ReadFile(filepath.Join("migrations", "20260723120000_add_recipe_highlighted.sql"))
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}
	migrationSQL := strings.ReplaceAll(string(migrationBytes), "-- +goose Up\n", "")
	migrationSQL = strings.ReplaceAll(migrationSQL, "-- +goose Down\n-- SQLite does not support dropping a column directly, so the rollback is left as a no-op.\n", "")

	if _, err := db.Exec(migrationSQL); err != nil {
		t.Fatalf("expected migration to be idempotent, got %v", err)
	}
}

func TestToggleRecipeHighlightPersistsForOwner(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE recipes (id INTEGER PRIMARY KEY, highlighted BOOLEAN NOT NULL DEFAULT 0);
		CREATE TABLE user_recipe (user_id INTEGER NOT NULL, recipe_id INTEGER NOT NULL);
		INSERT INTO recipes (id) VALUES (1);
		INSERT INTO user_recipe (user_id, recipe_id) VALUES (7, 1);
	`); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	service := &SQLiteService{DB: db, Mutex: &sync.Mutex{}}
	highlighted, err := service.ToggleRecipeHighlight(1, 7)
	if err != nil {
		t.Fatalf("toggle highlight: %v", err)
	}
	if !highlighted {
		t.Fatal("expected recipe to be highlighted")
	}

	if _, err := service.ToggleRecipeHighlight(1, 8); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected non-owner toggle to fail with sql.ErrNoRows, got %v", err)
	}

	var persisted bool
	if err := db.QueryRow(`SELECT highlighted FROM recipes WHERE id = 1`).Scan(&persisted); err != nil {
		t.Fatalf("read persisted highlight: %v", err)
	}
	if !persisted {
		t.Fatal("expected highlighted state to persist")
	}

	highlighted, err = service.ToggleRecipeHighlight(1, 7)
	if err != nil {
		t.Fatalf("toggle highlight off: %v", err)
	}
	if highlighted {
		t.Fatal("expected recipe to be unhighlighted")
	}
	if err := db.QueryRow(`SELECT highlighted FROM recipes WHERE id = 1`).Scan(&persisted); err != nil {
		t.Fatalf("read unhighlighted state: %v", err)
	}
	if persisted {
		t.Fatal("expected unhighlighted state to persist")
	}
}

func TestAddRecipeTxPersistsHighlighted(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("sqlite"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	if err := ensureRecipeHighlightColumn(context.Background(), db); err != nil {
		t.Fatalf("ensure highlighted column: %v", err)
	}

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	service := &SQLiteService{DB: db, Mutex: &sync.Mutex{}}
	recipeID, err := service.addRecipeTx(context.Background(), tx, models.Recipe{
		Name:         "Highlighted import",
		Highlighted:  true,
		Ingredients:  []string{"1 cup water"},
		Instructions: []string{"Serve."},
	}, 1)
	if err != nil {
		t.Fatalf("add recipe: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit transaction: %v", err)
	}

	var highlighted bool
	if err := db.QueryRow(`SELECT highlighted FROM recipes WHERE id = ?`, recipeID).Scan(&highlighted); err != nil {
		t.Fatalf("read highlighted state: %v", err)
	}
	if !highlighted {
		t.Fatal("expected imported highlighted state to persist")
	}
}

func TestScanRecipeConsumesHighlightedColumn(t *testing.T) {
	tests := []struct {
		name             string
		isSearch         bool
		wantDestinations int
		highlightedIndex int
	}{
		{name: "compact", isSearch: true, wantDestinations: 9, highlightedIndex: 7},
		{name: "full", wantDestinations: 34, highlightedIndex: 4},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recipe, err := scanRecipe(highlightScanner{
				t:                t,
				wantDestinations: tc.wantDestinations,
				highlightedIndex: tc.highlightedIndex,
			}, tc.isSearch)
			if err != nil {
				t.Fatalf("scan recipe: %v", err)
			}
			if !recipe.Highlighted {
				t.Fatal("expected highlighted state from scanned row")
			}
		})
	}
}

type highlightScanner struct {
	t                *testing.T
	wantDestinations int
	highlightedIndex int
}

func (s highlightScanner) Scan(dest ...any) error {
	s.t.Helper()
	if len(dest) != s.wantDestinations {
		s.t.Fatalf("Scan received %d destinations, want %d", len(dest), s.wantDestinations)
	}
	highlighted, ok := dest[s.highlightedIndex].(*bool)
	if !ok {
		s.t.Fatalf("destination %d has type %T, want *bool", s.highlightedIndex, dest[s.highlightedIndex])
	}
	*highlighted = true
	return nil
}

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

func TestScaleNutritionForYield(t *testing.T) {
	nutrition := models.Nutrition{
		Calories:           "800 kcal",
		TotalCarbohydrates: "40 g",
		Protein:            "20 g",
	}

	got := scaleNutritionForYield(nutrition, 4)
	want := models.Nutrition{
		Calories:           "200 kcal",
		TotalCarbohydrates: "10 g",
		Protein:            "5 g",
		IsPerServing:       true,
	}

	if got != want {
		t.Fatalf("scaleNutritionForYield() = %+v, want %+v", got, want)
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
