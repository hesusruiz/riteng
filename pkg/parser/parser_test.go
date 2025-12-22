package parser

import (
	"fmt"
	"testing"

	"github.com/hesusruiz/riteng/pkg/ast"
	"github.com/hesusruiz/riteng/pkg/lexer"
)

func TestParseDocument(t *testing.T) {
	input := `
<section>
  <p>Paragraph 1</p>
  <x-code>
    fmt.Println("Hello")
  </x-code>
`
	l := lexer.New(input)
	p := New(l)
	doc := p.ParseDocument()

	if len(doc.Children) != 1 {
		t.Fatalf("Expected 1 child (Section), got %d", len(doc.Children))
	}

	section, ok := doc.Children[0].(*ast.SectionNode)
	if !ok {
		t.Fatalf("Expected child 0 to be *ast.SectionNode, got %T", doc.Children[0])
	}
	if section.Identifier != "section" {
		t.Errorf("Expected section ID 'section', got %q", section.Identifier)
	}
	
	if len(section.Children) < 2 {
		for i, c := range section.Children {
			fmt.Printf("Child %d: %T\n", i, c)
		}
		t.Fatalf("Expected at least 2 children in section, got %d", len(section.Children))
	}
	
	// Check Paragraph
	para, ok := section.Children[0].(*ast.GenericBlock)
	if !ok {
		t.Fatalf("Expected section child 0 to be GenericBlock, got %T", section.Children[0])
	}
	if para.Tag != "p" {
		t.Errorf("Expected generic block tag 'p', got %q", para.Tag)
	}
	
	// Check x-code
	verbatim, ok := section.Children[1].(*ast.VerbatimNode)
	if !ok {
		t.Fatalf("Expected section child 1 to be VerbatimNode, got %T", section.Children[1])
	}
	if verbatim.Kind != "x-code" {
		t.Errorf("Expected kind 'x-code', got %q", verbatim.Kind)
	}
	
	expectedContent := "    fmt.Println(\"Hello\")\n"
	// Verify content
	if verbatim.Content != expectedContent {
		t.Errorf("Expected content:\n%q\nGot:\n%q", expectedContent, verbatim.Content)
	}
}
