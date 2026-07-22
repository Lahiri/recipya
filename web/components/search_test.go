package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/reaper47/recipya/internal/templates"
)

func TestSearchbarRendersTagInputBeforeTagChips(t *testing.T) {
	var buf bytes.Buffer
	if err := searchbar(templates.SearchbarData{}).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render searchbar: %v", err)
	}

	got := buf.String()
	inputIndex := strings.Index(got, `id="search_tag_input"`)
	chipsIndex := strings.Index(got, `id="search_tags"`)
	if inputIndex == -1 || chipsIndex == -1 {
		t.Fatalf("expected both tag input and tag chips in rendered HTML, got input=%d chips=%d", inputIndex, chipsIndex)
	}
	if chipsIndex < inputIndex {
		t.Fatalf("expected tag input to render before tag chips, got input at %d and chips at %d", inputIndex, chipsIndex)
	}
}

func TestSearchbarSubmitsWhenTagsChange(t *testing.T) {
	var buf bytes.Buffer
	if err := searchbar(templates.SearchbarData{}).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render searchbar: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "requestSubmit()") && !strings.Contains(got, "dispatchEvent(new Event('submit'") {
		t.Fatalf("expected tag changes to trigger form submission, got HTML: %s", got)
	}
}

func TestSearchbarComposesIntoHiddenQueryInput(t *testing.T) {
	var buf bytes.Buffer
	if err := searchbar(templates.SearchbarData{}).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render searchbar: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, `id="search_query" type="hidden" name="q"`) {
		t.Fatalf("expected a hidden composed query input, got HTML: %s", got)
	}
	if strings.Contains(got, "MutationObserver") {
		t.Fatalf("expected searchbar initialization without a global mutation observer, got HTML: %s", got)
	}
}

func TestListRecipesDoesNotReplaceSearchForm(t *testing.T) {
	var buf bytes.Buffer
	data := templates.Data{IsHxRequest: true}
	if err := ListRecipes(data).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render recipe list: %v", err)
	}

	got := buf.String()
	if strings.Contains(got, `id="search-form-shell"`) || strings.Contains(got, `hx-swap-oob`) {
		t.Fatalf("expected recipe result swaps to leave the search form mounted, got HTML: %s", got)
	}
}
