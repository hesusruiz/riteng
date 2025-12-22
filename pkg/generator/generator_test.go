package generator

import (
	"strings"
	"testing"

	"github.com/hesusruiz/riteng/pkg/ast"
)

func TestGenerateBasic(t *testing.T) {
	// Manually construct mostly valid AST
	doc := &ast.Document{
		Children: []ast.Block{
			&ast.SectionNode{
				Identifier: "sec1",
				Children: []ast.Block{
					&ast.GenericBlock{
						Tag: "p",
						Content: []ast.Inline{
							&ast.TextNode{Value: "Hello World "},
						},
					},
					&ast.VerbatimNode{
						Kind: "x-code",
						Content: "fmt.Println(\"Hi\")\n",
					},
				},
			},
		},
	}
	
	gen := New()
	html, err := gen.Generate(doc)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	
	expected := `<section id="sec1">
<h1>1 </h1>
<p>Hello World </p>
<pre><code>fmt.Println("Hi")
</code></pre>
</section>
`
	if strings.TrimSpace(html) != strings.TrimSpace(expected) {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, html)
	}
}
