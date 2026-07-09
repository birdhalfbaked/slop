package htmlclean

import (
	"strings"
	"testing"
)

const gutenbergSnippet = `<!-- wp:paragraph -->
<p>Hello <strong>world</strong>.</p>
<!-- /wp:paragraph -->

<!-- wp:image {"id":48} -->
<figure class="wp-block-image size-full"><img src="https://example.com/img.png" alt="test alt" class="wp-image-48"/><figcaption class="wp-element-caption">A caption</figcaption></figure>
<!-- /wp:image -->

<!-- wp:heading {"level":3} -->
<h3 class="wp-block-heading">Chess.com</h3>
<!-- /wp:heading -->`

func TestCleanGutenberg(t *testing.T) {
	out, err := Clean(gutenbergSnippet)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "wp-block") {
		t.Fatalf("wp classes should be stripped: %s", out)
	}
	if strings.Contains(out, "<!--") {
		t.Fatalf("gutenberg comments should be removed: %s", out)
	}
	if !strings.Contains(out, "<strong>") {
		t.Fatal("expected strong tag")
	}
	if !strings.Contains(out, "<figure>") {
		t.Fatal("expected figure")
	}
	if !strings.Contains(out, `alt="test alt"`) {
		t.Fatal("expected alt preserved")
	}
	if !strings.Contains(out, "<figcaption>") {
		t.Fatal("expected figcaption")
	}
	if !strings.Contains(out, "<h3>") {
		t.Fatal("expected h3")
	}
}

func TestCleanPreservesList(t *testing.T) {
	in := `<ol class="wp-block-list"><li>First item</li><li>Second item</li></ol>`
	out, err := Clean(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "<ol>") || !strings.Contains(out, "<li>") {
		t.Fatalf("lists should be preserved: %s", out)
	}
	if !strings.Contains(out, "First item") || !strings.Contains(out, "Second item") {
		t.Fatalf("list text should survive: %s", out)
	}
}

func TestCleanPreservesBlockquote(t *testing.T) {
	in := `<blockquote class="wp-block-quote"><p>I MUST take this pawn</p><cite>me</cite></blockquote>`
	out, err := Clean(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "<blockquote>") {
		t.Fatalf("expected blockquote: %s", out)
	}
	if !strings.Contains(out, "<cite>") {
		t.Fatalf("expected cite: %s", out)
	}
}

func TestCleanPreservesHR(t *testing.T) {
	in := `<hr class="wp-block-separator" style="margin-top:1rem"/>`
	out, err := Clean(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "<hr") {
		t.Fatalf("expected hr: %s", out)
	}
	if strings.Contains(out, "style=") {
		t.Fatalf("hr attributes should be stripped: %s", out)
	}
}
