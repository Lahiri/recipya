package main

import (
	"context"
	"fmt"
	"net/http/httptest"

	"github.com/reaper47/recipya/internal/models"
	"github.com/reaper47/recipya/internal/templates"
	"github.com/reaper47/recipya/web/components"
)

func main() {
	data := templates.Data{
		IsAuthenticated: false,
		IsHxRequest:     true,
		About:           templates.NewAboutData(),
		Title:           "Lovely Ukraine",
		Functions:       templates.NewFunctionsData[int64](),
		CookbookFeature: templates.CookbookFeature{
			Cookbook:  templates.MakeCookbookView(models.Cookbook{ID: 2, Title: "Lovely Ukraine"}, 1, 1),
			ShareData: templates.ShareData{IsFromHost: false, IsShared: true},
		},
		Pagination: templates.Pagination{IsHidden: true},
	}
	rr := httptest.NewRecorder()
	ctx := context.Background()
	err := components.CookbookIndex(data).Render(ctx, rr)
	fmt.Println(err)
	fmt.Println(rr.Body.String())
}
