package html2md

import (
	"strings"
	"testing"

	"slop/internal/import/htmlclean"
)

// Snippet from examples/birdhalf-baked.WordPress.2026-07-08.xml (chess-is-actually-fun).
const chessListSnippet = `<!-- wp:list {"ordered":true} -->
<ol class="wp-block-list"><!-- wp:list-item -->
<li>I have a distaste for any website that used .org in the modern age and expects to be taken seriously</li>
<!-- /wp:list-item -->

<!-- wp:list-item -->
<li>Social content matters and helps keep the game relevant. I feel this is lacking in lichess.org</li>
<!-- /wp:list-item --></ol>
<!-- /wp:list -->

<!-- wp:quote -->
<blockquote class="wp-block-quote"><!-- wp:paragraph -->
<p>I CAN take this pawn, therefore I MUST take this pawn</p>
<!-- /wp:paragraph --><cite>me</cite></blockquote>
<!-- /wp:quote -->`

func TestConvertFixtureSnippet(t *testing.T) {
	cleaned, err := htmlclean.Clean(chessListSnippet)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Convert(cleaned, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "1. I have a distaste") {
		t.Fatalf("expected ordered list from fixture: %s", out)
	}
	if !strings.Contains(out, "2. Social content matters") {
		t.Fatalf("expected second list item: %s", out)
	}
	if !strings.Contains(out, "> I CAN take this pawn") {
		t.Fatalf("expected blockquote from fixture: %s", out)
	}
	if !strings.Contains(out, "> — *me*") {
		t.Fatalf("expected cite from fixture: %s", out)
	}
}
