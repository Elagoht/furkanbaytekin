package blog

import (
	"bytes"
	"html/template"
	"slices"
	"strconv"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

// Heading is one entry of a post's table of contents.
type Heading struct {
	ID    string
	Text  string
	Level int
}

type Rendered struct {
	HTML     template.HTML
	Headings []Heading
}

// Renderer turns a post's Markdown into HTML.
//
// Raw HTML in the Markdown is passed through: posts embed videos and live
// demos, and they come from this site's own CMS, which is trusted to the same
// degree as the templates.
type Renderer struct {
	md     goldmark.Markdown
	client *Client
}

func NewRenderer(client *Client) *Renderer {
	return &Renderer{
		md: goldmark.New(
			goldmark.WithExtensions(
				extension.GFM,
				highlighting.NewHighlighting(
					highlighting.WithStyle("dracula"),
					highlighting.WithFormatOptions(chromahtml.TabWidth(4)),
				),
			),
			goldmark.WithParserOptions(parser.WithAutoHeadingID()),
			goldmark.WithRendererOptions(html.WithUnsafe()),
		),
		client: client,
	}
}

// Render converts source, collecting its level 2 and 3 headings.
func (r *Renderer) Render(source string) (Rendered, error) {
	src := []byte(source)
	doc := r.md.Parser().Parse(text.NewReader(src))

	var headings []Heading
	err := ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := node.(type) {
		case *ast.Heading:
			if n.Level < 2 || n.Level > 3 {
				return ast.WalkContinue, nil
			}
			id, _ := n.AttributeString("id")
			if value, ok := id.([]byte); ok {
				headings = append(headings, Heading{ID: string(value), Text: plainText(n, src), Level: n.Level})
			}
		case *ast.Image:
			n.Destination = []byte(r.client.Asset(string(n.Destination)))
			// Declared, so opti-image serves it from this site.
			if r.client.IsAsset(string(n.Destination)) {
				n.SetAttributeString("width", []byte(strconv.Itoa(ImageWidth)))
				n.SetAttributeString("height", []byte(strconv.Itoa(ImageHeight)))
				n.SetAttributeString("loading", []byte("lazy"))
			}
		case *ast.Link:
			if strings.HasPrefix(string(n.Destination), "/uploads/") {
				n.Destination = []byte(r.client.Asset(string(n.Destination)))
			}
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return Rendered{}, err
	}

	var out bytes.Buffer
	if err := r.md.Renderer().Render(&out, src, doc); err != nil {
		return Rendered{}, err
	}
	return Rendered{HTML: template.HTML(out.String()), Headings: headings}, nil
}

// plainText is a heading's text without its markup: "never trust `alg`" reads
// "never trust alg".
func plainText(node ast.Node, src []byte) string {
	var b strings.Builder
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := n.(type) {
		case *ast.Text:
			b.Write(t.Segment.Value(src))
		case *ast.String:
			b.Write(t.Value)
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

// Source returns a post's Markdown as it should be read on its own: every image,
// and every link to an upload, made absolute against the CMS, as Render makes
// them, since a relative "/uploads/..." means nothing away from it.
//
// The destinations are found by parsing and replaced where the source spells
// them — "](dest" in an inline link, "]: dest" in a reference definition — outside
// code, which shows Markdown rather than being it; the rest of the text is
// unchanged.
func (r *Renderer) Source(source string) (string, error) {
	src := []byte(source)
	doc := r.md.Parser().Parse(text.NewReader(src))
	rewrite := map[string]string{}
	var code []text.Segment
	err := ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var dest string
		switch n := node.(type) {
		case *ast.FencedCodeBlock, *ast.CodeBlock:
			lines := n.Lines()
			for i := range lines.Len() {
				code = append(code, lines.At(i))
			}
			return ast.WalkSkipChildren, nil
		case *ast.CodeSpan:
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					code = append(code, t.Segment)
				}
			}
			return ast.WalkSkipChildren, nil
		case *ast.Image:
			dest = string(n.Destination)
		case *ast.Link:
			if !strings.HasPrefix(string(n.Destination), "/uploads/") {
				return ast.WalkContinue, nil
			}
			dest = string(n.Destination)
		default:
			return ast.WalkContinue, nil
		}
		if absolute := r.client.Asset(dest); absolute != dest {
			rewrite[dest] = absolute
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return "", err
	}
	if len(rewrite) == 0 {
		return source, nil
	}
	pairs := make([]string, 0, len(rewrite)*6)
	for dest, absolute := range rewrite {
		pairs = append(pairs,
			"]("+dest, "]("+absolute,
			"](<"+dest, "](<"+absolute,
			"]: "+dest, "]: "+absolute,
		)
	}
	replacer := strings.NewReplacer(pairs...)

	// The text between the code, replaced; the code, as it is.
	slices.SortFunc(code, func(a, b text.Segment) int { return a.Start - b.Start })
	var out strings.Builder
	at := 0
	for _, seg := range code {
		if seg.Start < at {
			continue
		}
		out.WriteString(replacer.Replace(source[at:seg.Start]))
		out.WriteString(source[seg.Start:seg.Stop])
		at = seg.Stop
	}
	out.WriteString(replacer.Replace(source[at:]))
	return out.String(), nil
}
