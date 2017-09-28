package blog

import (
	"net/http"
	"testing"
)

var page *Page

func TestPageMarkdown(t *testing.T) {
	p, err := NewPage("./blog-test/content.md")
	if err != nil {
		t.Fatal(err)
	}

	page = p

	if page.ID != "blog-test" {
		t.Errorf("page.ID should be `blog-test`, got `%s`", page.ID)
	}

	if page.Title != "Hello, world" {
		t.Errorf("page.Title should be `Hello, world`, got `%s`", page.Title)
	}

	if page.Desc != "A test blog" {
		t.Errorf("page.Desc should be `A test blog`, got `%s`", page.Desc)
	}

	if len(page.Tags) != 3 {
		t.Fatalf("expected 3 tags, got %d", len(page.Tags))
	}

	if page.Tags[0] != "hello" {
		t.Errorf("first tag should be `hello`, got `%s`", page.Tags[0])
	}

	if page.Tags[1] != "world" {
		t.Errorf("second tag should be `world`, got `%s`", page.Tags[1])
	}

	if page.Tags[2] != "more-tags" {
		t.Errorf("third tag should be `more-tags`, got `%s`", page.Tags[2])
	}

	// Images can be repeated, so assigned shortIDs must be consistent
	if len(page.Images) != 2 {
		t.Fatalf("expected 2 page.Images, got %d", len(page.Images))
	}
}

func TestPageSave(t *testing.T) {
	if err := page.Save(); err != nil {
		t.Fatal(err)
	}

	for k, v := range page.Images {
		imageURL := cdnURL + "/" + v
		resp, err := http.Get(imageURL)
		if err != nil {
			t.Errorf("got error requesting `%s` (%s): %v", v, k, err)
		} else if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 for `%s` (%s), got `%s`", v, k, resp.Status)
		}
	}

	if _, err := GetPage(page.ID); err != nil {
		t.Fatal(err)
	}

	if results, err := SearchPages("hello"); err != nil {
		t.Error(err)
	} else if len(results) != 1 {
		t.Errorf("expected 1 search result for `hello`, got %d", len(results))
	}

	if tags, err := AllTags(); err != nil {
		t.Fatal(err)
	} else if len(tags) != len(page.Tags) {
		t.Errorf("expected %d tags, got %d", len(page.Tags), len(tags))
	} else if num, exists := tags["hello"]; !exists {
		t.Errorf("expected tag `hello` to exist")
	} else if num != 1 {
		t.Errorf("expected tag `hello` to have 1 occurence, has %d", num)
	}

	if results, err := GetPagesByTag("hello"); err != nil {
		t.Fatal(err)
	} else if len(results) != 1 {
		t.Errorf("expected 1 result for pages with tag `hello`, got %d", len(results))
	}
}

func TestPageDelete(t *testing.T) {
	if err := page.Delete(); err != nil {
		t.Fatal(err)
	}

	for k, v := range page.Images {
		imageURL := cdnURL + "/" + v
		resp, err := http.Get(imageURL)
		if err != nil {
			t.Errorf("got error requesting `%s` (%s): %v", v, k, err)
		} else if resp.StatusCode == http.StatusOK {
			t.Errorf("expected `%s` (%s) to be unavailable, got 200 OK", v, k)
		}
	}

	if _, err := GetPage(page.ID); err != ErrPageNotFound {
		t.Errorf("expected id `%s` to not be found, got %v", page.ID, err)
	}

	if results, err := SearchPages("hello"); err != nil {
		t.Fatal(err)
	} else if len(results) != 0 {
		t.Errorf("expected no search results for `hello`, got %d", len(results))
	}

	if tags, err := AllTags(); err != nil {
		t.Fatal(err)
	} else if len(tags) != 0 {
		t.Errorf("expected 0 tags, got %d (%v)", len(tags), tags)
	} else if num, exists := tags["hello"]; exists {
		t.Errorf("expected tag `hello` to not exist")
	} else if num != 0 {
		t.Errorf("expected tag `hello` to have 0 occurences, has %d", num)
	}

	if results, err := GetPagesByTag("hello"); err != nil {
		t.Fatal(err)
	} else if len(results) != 0 {
		t.Errorf("expected 0 results for pages with tag `hello`, got %d", len(results))
	}
}
