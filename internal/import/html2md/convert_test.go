package html2md

import (
	"strings"
	"testing"
)

func TestConvertFigureAndHeading(t *testing.T) {
	in := `<figure><img src="https://example.com/a.png" alt="alt text"><figcaption>Cap</figcaption></figure>
<h3>Section</h3>
<p>Text with <a href="https://example.com">link</a> and <em>emphasis</em>.</p>`

	out, err := Convert(in, func(u string) string {
		if u == "https://example.com/a.png" {
			return "/static/uploads/a.png"
		}
		return u
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `class="post-figure"`) {
		t.Fatalf("expected post-figure: %s", out)
	}
	if !strings.Contains(out, `/static/uploads/a.png`) {
		t.Fatalf("expected rewritten src: %s", out)
	}
	if !strings.Contains(out, "<figcaption>Cap</figcaption>") {
		t.Fatalf("expected caption: %s", out)
	}
	if !strings.Contains(out, "### Section") {
		t.Fatalf("expected heading: %s", out)
	}
	if !strings.Contains(out, "[link](https://example.com)") {
		t.Fatalf("expected markdown link: %s", out)
	}
	if !strings.Contains(out, "*emphasis*") {
		t.Fatalf("expected emphasis: %s", out)
	}
}

func TestConvertCodeBlock(t *testing.T) {
	in := `<pre><code>fmt.Println("hi")</code></pre>`
	out, err := Convert(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "```") || !strings.Contains(out, `fmt.Println("hi")`) {
		t.Fatalf("expected fenced code: %s", out)
	}
}

func TestConvertList(t *testing.T) {
	in := `<ul><li>Functional</li><li>Decorative</li></ul>
<ol><li>First</li><li>Second</li></ol>`
	out, err := Convert(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "- Functional") {
		t.Fatalf("expected bullet list: %s", out)
	}
	if !strings.Contains(out, "1. First") {
		t.Fatalf("expected ordered list: %s", out)
	}
}

func TestConvertBlockquote(t *testing.T) {
	in := `<blockquote><p>I MUST take this pawn</p><cite>me</cite></blockquote>`
	out, err := Convert(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "> I MUST take this pawn") {
		t.Fatalf("expected quote line: %s", out)
	}
	if !strings.Contains(out, "> — *me*") {
		t.Fatalf("expected cite line: %s", out)
	}
}

func TestConvertHR(t *testing.T) {
	in := `<p>Above</p><hr><p>Below</p>`
	out, err := Convert(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "---") {
		t.Fatalf("expected hr as ---: %s", out)
	}
}

func TestConvertInlineSubSup(t *testing.T) {
	in := `<p>10<sup>18</sup> bytes</p>`
	out, err := Convert(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "<sup>18</sup>") {
		t.Fatalf("expected sup preserved: %s", out)
	}
}
