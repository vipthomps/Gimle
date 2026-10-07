package api

import (
	"reflect"
	"testing"

	"github.com/vipthomps/gimle/backend/internal/models"
)

func TestValidateBookmark(t *testing.T) {
	b := models.Bookmark{Name: " Immich ", URL: "photos.example.lan", Port: 2283}
	if err := validateBookmark(&b); err != nil {
		t.Fatal(err)
	}
	if b.URL != "https://photos.example.lan" || b.Name != "Immich" || b.Port != 0 {
		t.Errorf("got %+v", b)
	}
	for _, u := range []string{"", "ftp://x.lan", "https://"} {
		b := models.Bookmark{URL: u}
		if validateBookmark(&b) == nil {
			t.Errorf("%q: want an error", u)
		}
	}
}

func TestCleanTags(t *testing.T) {
	got := cleanTags([]string{" Media ", "", "a,b", "Media"})
	if want := []string{"Media", "a b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDocsRepo(t *testing.T) {
	for in, want := range map[string]string{
		"octocat/lab":                                   "octocat/lab",
		" https://github.com/octocat/lab.git ":          "octocat/lab",
		"git@github.com:octocat/lab.git":                "octocat/lab",
		"https://github.com/octocat/lab/tree/main/docs": "octocat/lab",
		"":           "",
		"not a repo": "error",
	} {
		got, err := docsRepo(in)
		if err != nil {
			got = "error"
		}
		if got != want {
			t.Errorf("docsRepo(%q) = %q, want %q", in, got, want)
		}
	}
}
