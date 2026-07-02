package server_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/reaper47/recipya/internal/models"
)

func TestDebugCookbookEmptyBody(t *testing.T) {
	srv, ts, c := createWSServer()
	defer c.CloseNow()

	_, repo, revertFunc := prepareCookbook(srv)
	defer revertFunc()
	id := int64(len(repo.CookbooksRegistered[1]) + 1)
	repo.CookbooksRegistered[1] = append(repo.CookbooksRegistered[1], models.Cookbook{ID: id, Title: "Ensiferum"})
	srv.Repository = repo

	rr := sendHxRequestAsLoggedInNoBody(srv, http.MethodGet, ts.URL+"/cookbooks/"+fmt.Sprint(id)+"?page=1")

	fmt.Println("STATUS", rr.Code)
	fmt.Println(getBodyHTML(rr))
}
