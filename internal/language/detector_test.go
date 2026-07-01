package language

import "testing"

func TestDetectRecipe(t *testing.T) {
	tests := []struct {
		name string
		text RecipeText
		want Code
	}{
		{
			name: "italian instructions outweigh ambiguous title",
			text: RecipeText{
				Name:        "Spaghetti",
				Description: "Spaghetti aglio olio e peperoncino perfetti con la ricetta tradizionale.",
				Ingredients: []string{"Spaghetti 320 g", "Aglio 3 spicchi", "Olio extravergine d'oliva 70 g"},
				Instructions: []string{
					"Per preparare gli spaghetti, sbucciate gli spicchi d'aglio e tagliate il peperoncino.",
					"Scaldate una padella, versate l'olio e cuocete per 10 minuti.",
				},
			},
			want: Italian,
		},
		{
			name: "english instructions",
			text: RecipeText{
				Name:        "Tomato pasta",
				Description: "A simple weeknight pasta recipe with tomato sauce.",
				Ingredients: []string{"320 g spaghetti", "2 cloves garlic", "olive oil"},
				Instructions: []string{
					"Heat the oil in a pan, add the garlic, and cook until fragrant.",
					"Boil the pasta, stir into the sauce, and serve with parsley.",
				},
			},
			want: English,
		},
		{
			name: "ambiguous ingredients default English",
			text: RecipeText{Ingredients: []string{"spaghetti", "sale", "pasta"}},
			want: English,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectRecipe(tt.text)
			if got.Language != tt.want {
				t.Fatalf("DetectRecipe(%s) language = %q, want %q", tt.name, got.Language, tt.want)
			}
		})
	}
}
