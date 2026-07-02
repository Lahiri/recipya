package server_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/reaper47/recipya/internal/models"
)

func TestDebugAddRecipeRequest(t *testing.T) {
	srv := newServerTest()
	recipes := models.Recipes{{ID: 1, Name: "Cheese Toasts"}, {ID: 2, Name: "Maple Syrup Waffles"}, {ID: 3, Name: "Chicken Jerky"}}
	repo := &mockRepository{
		CookbooksRegistered: map[int64][]models.Cookbook{
			1: {
				{ID: 1, Recipes: []models.Recipe{recipes[0]}},
				{ID: 2, Recipes: []models.Recipe{recipes[1]}},
			},
		},
		RecipesRegistered: map[int64]models.Recipes{1: append(models.Recipes(nil), recipes...)},
	}
	srv.Repository = repo

	rr := sendHxRequestAsLoggedIn(srv, http.MethodPost, "/cookbooks/2", formHeader, strings.NewReader("recipeId=3"))
	fmt.Println("code", rr.Code)
	for _, c := range repo.CookbooksRegistered[1] {
		fmt.Printf("c=%d ids=%v\n", c.ID, ids(c.Recipes))
	}
}

func ids(recipes []models.Recipe) []int64 {
	out := make([]int64, len(recipes))
	for i, r := range recipes {
		out[i] = r.ID
	}
	return out
}
