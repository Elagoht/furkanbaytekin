package main

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Elagoht/collage/pkg/collagetest"
)

// elagoht/ogimage: every page carries a share card drawn by this site, at a URL
// made from what is on it.

var ogImage = regexp.MustCompile(`<meta property="og:image" content="([^"]+)"`)

// shareCard returns the og:image of the page at target, after checking the page
// names exactly one, on this site, as a large Twitter card.
func shareCard(t *testing.T, c *collagetest.Client, target string) string {
	t.Helper()
	body := c.Get(target).WantStatus(http.StatusOK).Body
	found := ogImage.FindAllStringSubmatch(body, -1)
	if len(found) != 1 {
		t.Fatalf("GET %s has %d og:image tags, want 1", target, len(found))
	}
	card := found[0][1]
	if !strings.HasPrefix(card, "https://furkanbaytekin.dev/_og/") || !strings.HasSuffix(card, ".png") {
		t.Errorf("GET %s: og:image = %q, want a card under /_og/", target, card)
	}
	for _, want := range []string{
		`<meta name="twitter:card" content="summary_large_image">`,
		`<meta property="og:image:width" content="1200">`,
		`<meta property="og:image:height" content="630">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("GET %s does not contain %s", target, want)
		}
	}
	return card
}

// wantCard fetches a card, checks it is a 1200×630 PNG cached for good, and
// returns it.
func wantCard(t *testing.T, c *collagetest.Client, card string) image.Image {
	t.Helper()
	u, err := url.Parse(card)
	if err != nil {
		t.Fatal(err)
	}
	res := c.Get(u.Path).WantStatus(http.StatusOK)
	if got := res.Header.Get("Content-Type"); got != "image/png" {
		t.Errorf("GET %s: Content-Type = %q", u.Path, got)
	}
	if got := res.Header.Get("Cache-Control"); !strings.Contains(got, "immutable") {
		t.Errorf("GET %s: Cache-Control = %q, want immutable", u.Path, got)
	}
	wantSize(t, u.Path, []byte(res.Body))
	img, err := png.Decode(strings.NewReader(res.Body))
	if err != nil {
		t.Fatalf("%s: %v", u.Path, err)
	}
	return img
}

func wantSize(t *testing.T, name string, data []byte) {
	t.Helper()
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("%s is not a PNG: %v", name, err)
	}
	if cfg.Width != 1200 || cfg.Height != 630 {
		t.Errorf("%s is %d×%d, want 1200×630", name, cfg.Width, cfg.Height)
	}
}

// A post's card is its own: its title and cover, not the site's.
func TestPostHasItsOwnShareCard(t *testing.T) {
	c := client(t)
	post := shareCard(t, c, "/blogs/hello-world")
	img := wantCard(t, c, post)
	// The cover, fetched from the CMS, sits right of the title.
	if r, g, b, _ := img.At(936, 315).RGBA(); r>>8 < 200 || g>>8 > 60 || b>>8 > 60 {
		t.Errorf("the card's cover is not drawn: rgb(%d, %d, %d) where it sits", r>>8, g>>8, b>>8)
	}
	if about := shareCard(t, c, "/about"); about == post {
		t.Error("the post and /about carry the same card")
	}
}

// A post renamed in the CMS gets a new card at a new URL once the webhook
// drops its page: nothing is invalidated, because nothing at the old URL changes.
func TestRetitledPostGetsANewCard(t *testing.T) {
	cms := fakeCMS(t)
	c := clientOf(t, newTestApp(t))
	before := shareCard(t, c, "/blogs/hello-world")
	cms.retitle("Hello Again")
	webhook(c, testWebhookSecret, `{"event":"post.updated","entity":"post","slug":"hello-world","action":"updated"}`).WantStatus(http.StatusOK)
	after := shareCard(t, c, "/blogs/hello-world")
	if before == after {
		t.Error("a retitled post kept its card's URL")
	}
	wantCard(t, c, after)
	wantCard(t, c, before) // a share made before the change still shows
}

// Every page without a card of its own gets the default one, from its title.
func TestPagesGetTheDefaultShareCard(t *testing.T) {
	c := client(t)
	for _, target := range []string{"/", "/about", "/blogs"} {
		wantCard(t, c, shareCard(t, c, target))
	}
}

// An export holds every card its pages name, since there is no server to draw
// them later.
func TestExportWritesShareCards(t *testing.T) {
	fakeCMS(t)
	out := t.TempDir()
	if err := staticBuild(newTestApp(t), out, false); err != nil {
		t.Fatalf("staticBuild() = %v", err)
	}
	cards := 0
	err := filepath.WalkDir(out, func(file string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(file) != ".html" {
			return err
		}
		page, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		for _, m := range ogImage.FindAllSubmatch(page, -1) {
			u, err := url.Parse(string(m[1]))
			if err != nil {
				return err
			}
			data, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(u.Path)))
			if err != nil {
				t.Errorf("%s names %s, which the export does not hold", file, u.Path)
				continue
			}
			wantSize(t, u.Path, data)
			cards++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if cards == 0 {
		t.Error("no exported page names a card")
	}
}
