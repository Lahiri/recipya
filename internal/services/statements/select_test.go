package statements

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/reaper47/recipya/internal/models"
	_ "modernc.org/sqlite"
)

func BenchmarkBuildPaginatedResultsQuery(b *testing.B) {
	for i := 0; i < b.N; i++ {
		got := buildSelectPaginatedResultsQuery(models.SearchOptionsRecipes{Query: "one two three four"})
		_ = got
	}
}

func BenchmarkBuildSelectNutrientFDC(b *testing.B) {
	for i := 0; i < b.N; i++ {
		got := BuildSelectNutrientFDC([]string{"one", "two", "three", "four", "five"})
		_ = got
	}
}

func TestBuildBaseSelectRecipe(t *testing.T) {
	testcases := []struct {
		name string
		in   models.Sort
		want string
	}{
		{
			name: "A-Z",
			in:   models.Sort{IsAToZ: true},
			want: "ROW_NUMBER() OVER (ORDER BY CASE WHEN recipes.highlighted = 1 THEN 0 ELSE 1 END, recipes.name ASC) AS row_num",
		},
		{
			name: "Z-A",
			in:   models.Sort{IsZToA: true},
			want: "ROW_NUMBER() OVER (ORDER BY CASE WHEN recipes.highlighted = 1 THEN 0 ELSE 1 END, recipes.name DESC) AS row_num",
		},
		{
			name: "new to old",
			in:   models.Sort{IsNewestToOldest: true},
			want: "ROW_NUMBER() OVER (ORDER BY CASE WHEN recipes.highlighted = 1 THEN 0 ELSE 1 END, recipes.created_at DESC) AS row_num",
		},
		{
			name: "old to new",
			in:   models.Sort{IsOldestToNewest: true},
			want: "ROW_NUMBER() OVER (ORDER BY CASE WHEN recipes.highlighted = 1 THEN 0 ELSE 1 END, recipes.created_at ASC) AS row_num",
		},
		{
			name: "default",
			in:   models.Sort{IsDefault: true},
			want: "ROW_NUMBER() OVER (ORDER BY CASE WHEN recipes.highlighted = 1 THEN 0 ELSE 1 END, recipes.id) AS row_num",
		},
		{
			name: "random",
			in:   models.Sort{IsRandom: true},
			want: "ROW_NUMBER() OVER (ORDER BY CASE WHEN recipes.highlighted = 1 THEN 0 ELSE 1 END, RANDOM()) AS row_num",
		},
		{
			name: "no options",
			in:   models.Sort{},
			want: "ROW_NUMBER() OVER (ORDER BY CASE WHEN recipes.highlighted = 1 THEN 0 ELSE 1 END, recipes.id) AS row_num",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildBaseSelectRecipe(tc.in)

			before, after1, _ := strings.Cut(got, "FROM recipes")
			if !strings.Contains(before, tc.want) {
				t.Fatalf("expected %q in SELECT of query", tc.want)
			}

			_, after2, _ := strings.Cut(baseSelectSearchRecipe, "FROM recipes")
			if after1 != after2 {
				t.Fatal("FROM recipes bit from baseRecipes variable not equal")
			}
		})
	}
}

func TestBuildBaseSelectRecipeHighlightsFirst(t *testing.T) {
	got := BuildBaseSelectRecipe(models.Sort{})
	if !strings.Contains(got, "CASE WHEN recipes.highlighted = 1 THEN 0 ELSE 1 END") {
		t.Fatalf("expected highlighted-first ordering in query, got %q", got)
	}
}

func TestBuildBaseSelectRecipeExecutesWithHighlightsFirst(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE recipes (id INTEGER PRIMARY KEY, name TEXT, description TEXT, highlighted BOOLEAN, image TEXT, created_at TIMESTAMP);
		CREATE TABLE category_recipe (recipe_id INTEGER, category_id INTEGER);
		CREATE TABLE categories (id INTEGER, name TEXT);
		CREATE TABLE keyword_recipe (recipe_id INTEGER, keyword_id INTEGER);
		CREATE TABLE keywords (id INTEGER, name TEXT);
		CREATE TABLE user_recipe (recipe_id INTEGER, user_id INTEGER);
		INSERT INTO recipes (id, name, highlighted) VALUES (1, 'First by ID', 0), (2, 'Highlighted', 1);
		INSERT INTO user_recipe (recipe_id, user_id) VALUES (1, 7), (2, 7);
	`); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	for _, tc := range []struct {
		name string
		sort models.Sort
	}{
		{name: "default"},
		{name: "random", sort: models.Sort{IsRandom: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			query := "SELECT recipe_id FROM (" + BuildBaseSelectRecipe(tc.sort) + " WHERE user_recipe.user_id = ? GROUP BY recipes.id) ORDER BY row_num"
			rows, err := db.Query(query, 7)
			if err != nil {
				t.Fatalf("query recipes: %v", err)
			}
			defer rows.Close()

			var ids []int64
			for rows.Next() {
				var id int64
				if err := rows.Scan(&id); err != nil {
					t.Fatalf("scan recipe ID: %v", err)
				}
				ids = append(ids, id)
			}
			if err := rows.Err(); err != nil {
				t.Fatalf("iterate recipes: %v", err)
			}
			if len(ids) != 2 || ids[0] != 2 || ids[1] != 1 {
				t.Fatalf("recipe order = %v, want [2 1]", ids)
			}
		})
	}
}

func TestBuildSelectNutrientFDC(t *testing.T) {
	testcases := []struct {
		name        string
		ingredients []string
		want        string
	}{
		{
			name:        "one ingredient",
			ingredients: []string{"one"},
			want:        "WHERE description LIKE '%one%'",
		},
		{
			name:        "multiple ingredients",
			ingredients: []string{"one", "two"},
			want:        "WHERE description LIKE '%one%' AND description LIKE '%two%'",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildSelectNutrientFDC(tc.ingredients)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("expected %q in query", tc.want)
			}
		})
	}
}

func expectedSearchRecipeQuery(hasMatch bool, includeCookbookFilter bool) string {
	var sb strings.Builder
	sb.WriteString("SELECT recipe_id, name, description, image, created_at, category, keywords, highlighted, row_num FROM ( SELECT recipes.id AS recipe_id, recipes.name AS name, recipes.description AS description, recipes.highlighted AS highlighted, recipes.image AS image, recipes.created_at AS created_at, categories.name AS category, GROUP_CONCAT(DISTINCT keywords.name) AS keywords, user_id, ROW_NUMBER() OVER (ORDER BY CASE WHEN recipes.highlighted = 1 THEN 0 ELSE 1 END, recipes.id) AS row_num FROM recipes LEFT JOIN category_recipe ON recipes.id = category_recipe.recipe_id LEFT JOIN categories ON category_recipe.category_id = categories.id LEFT JOIN keyword_recipe ON recipes.id = keyword_recipe.recipe_id LEFT JOIN keywords ON keyword_recipe.keyword_id = keywords.id LEFT JOIN user_recipe ON recipes.id = user_recipe.recipe_id WHERE recipes.id IN (SELECT id FROM recipes_fts WHERE user_id = ?")
	if hasMatch {
		sb.WriteString(" AND recipes_fts MATCH ?")
	}
	sb.WriteString(" ORDER BY rank)")
	if includeCookbookFilter {
		sb.WriteString(" AND recipes.id NOT IN (SELECT recipe_id FROM cookbook_recipes WHERE cookbook_id = ?)")
	}
	sb.WriteString(" GROUP BY recipes.id)")
	return sb.String()
}

func TestSelectSearchRecipe(t *testing.T) {
	testcases := []struct {
		name    string
		options models.SearchOptionsRecipes
		want    string
	}{
		{
			name: "no queries",
			want: expectedSearchRecipeQuery(false, false),
		},
		{
			name: "advanced category only",
			options: models.SearchOptionsRecipes{
				Advanced: models.AdvancedSearch{Category: "breakfast"},
			},
			want: expectedSearchRecipeQuery(true, false),
		},
		{
			name: "advanced multiple categories",
			options: models.SearchOptionsRecipes{
				Advanced: models.AdvancedSearch{Category: "breakfast,dinner"},
			},
			want: expectedSearchRecipeQuery(true, false),
		},
		{
			name:    "one query",
			options: models.SearchOptionsRecipes{Query: "one two three four"},
			want:    expectedSearchRecipeQuery(true, false),
		},
		{
			name: "one query with advanced search",
			options: models.SearchOptionsRecipes{
				Advanced: models.AdvancedSearch{Category: "breakfast"},
				Query:    "one two three four",
			},
			want: expectedSearchRecipeQuery(true, false),
		},
		{
			name:    "cookbook search",
			options: models.SearchOptionsRecipes{Query: "choco", CookbookID: 1},
			want:    expectedSearchRecipeQuery(true, true),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildSearchRecipeQuery(tc.options)
			compareSQL(t, got, tc.want)
		})
	}
}

func TestBuildSelectPaginatedResults(t *testing.T) {
	testcases := []struct {
		name    string
		options models.SearchOptionsRecipes
		want    string
	}{
		{
			name:    "empty query",
			options: models.SearchOptionsRecipes{Page: 1},
			want:    "WITH results AS (" + expectedSearchRecipeQuery(false, false) + ") SELECT * FROM results WHERE row_num BETWEEN 1 AND 15",
		},
		{
			name:    "full search one query",
			options: models.SearchOptionsRecipes{Query: "one two three four", Page: 2},
			want:    "WITH results AS (" + expectedSearchRecipeQuery(true, false) + ") SELECT * FROM results WHERE row_num BETWEEN 16 AND 30",
		},
		{
			name:    "with advanced",
			options: models.SearchOptionsRecipes{Query: "one two", Page: 1, Advanced: models.AdvancedSearch{Category: "breakfast"}},
			want:    "WITH results AS (" + expectedSearchRecipeQuery(true, false) + ") SELECT * FROM results WHERE row_num BETWEEN 1 AND 15",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildSelectPaginatedResults(tc.options)
			compareSQL(t, got, tc.want)
		})
	}
}

func TestBuildSelectSearchResultsCount(t *testing.T) {
	testcases := []struct {
		name    string
		queries []string
		options models.SearchOptionsRecipes
		want    string
	}{
		{
			name:    "empty query",
			options: models.SearchOptionsRecipes{Page: 1},
			want:    "WITH results AS (" + expectedSearchRecipeQuery(false, false) + ")SELECT COUNT(*) FROM results",
		},
		{
			name:    "full search one query",
			options: models.SearchOptionsRecipes{Query: "one two three four", Page: 3},
			want:    "WITH results AS (" + expectedSearchRecipeQuery(true, false) + ")SELECT COUNT(*) FROM results",
		},
		{
			name:    "with advanced",
			options: models.SearchOptionsRecipes{Query: "one two three four", Page: 3, Advanced: models.AdvancedSearch{Category: "breakfast", Text: "one two three four"}},
			want:    "WITH results AS (" + expectedSearchRecipeQuery(true, false) + ")SELECT COUNT(*) FROM results",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildSelectSearchResultsCount(tc.options)
			compareSQL(t, got, tc.want)
		})
	}
}

func compareSQL(tb testing.TB, got, want string) {
	tb.Helper()
	got = strings.Join(strings.Fields(strings.TrimSpace(got)), " ")
	want = strings.Join(strings.Fields(strings.TrimSpace(want)), " ")
	if got != want {
		tb.Fatalf("got:\n%q\nbut want:\n%q", got, want)
	}
}
