package generator

import (
	"fmt"
	"strings"

	"github.com/hesusruiz/riteng/pkg/ast"
)

type Generator struct {
	output          strings.Builder
	sectionCounters []int
}

func New() *Generator {
	return &Generator{
		// Start with one counter for the top level (level 0)
		sectionCounters: []int{0},
	}
}

func (g *Generator) Generate(doc *ast.Document) (string, error) {
	g.output.Reset()
	// Reset counters for each generation
	g.sectionCounters = []int{0}
	
	// Document Children
	for _, child := range doc.Children {
		err := g.visitBlock(child)
		if err != nil {
			return "", err
		}
	}
	return g.output.String(), nil
}

func (g *Generator) visitBlock(node ast.Block) error {
	switch n := node.(type) {
	case *ast.SectionNode:
		return g.visitSection(n)
	case *ast.GenericBlock:
		return g.visitGenericBlock(n)
	case *ast.VerbatimNode:
		return g.visitVerbatim(n)
	case *ast.IncludeNode:
		// Includes should effectively be resolved by Parser/Loader?
		// Or if they remain in AST, we should warn or skip?
		// For now, ignore.
		return nil
	default:
		return fmt.Errorf("unknown block type: %T", node)
	}
}

func (g *Generator) visitSection(n *ast.SectionNode) error {
	// 1. Increment current level counter
	if len(g.sectionCounters) > 0 {
		g.sectionCounters[len(g.sectionCounters)-1]++
	} else {
		// Should not happen if initialized correctly
		g.sectionCounters = append(g.sectionCounters, 1)
	}
	
	// 2. Calculate Header Level (h1, h2, ...)
	// Level is determined by depth (stack size)
	level := len(g.sectionCounters)
	if level > 6 { level = 6 } // Cap at h6
	
	// 3. Construct Numbering String (e.g. "1.2")
	var numberParts []string
	for _, c := range g.sectionCounters {
		numberParts = append(numberParts, fmt.Sprintf("%d", c))
	}
	numberStr := strings.Join(numberParts, ".")
	
	// 4. Generate Output
	g.output.WriteString("<section id=\"")
	g.output.WriteString(n.Identifier)
	g.output.WriteString("\">\n")
	
	// Header Tag
	g.output.WriteString(fmt.Sprintf("<h%d>", level))
	g.output.WriteString(numberStr + " ")
	
	// Title Content (Inline)
	for _, inline := range n.Title {
		g.visitInline(inline)
	}
	g.output.WriteString(fmt.Sprintf("</h%d>\n", level))
	
	// 5. Prepare for children (Push new counter 0)
	g.sectionCounters = append(g.sectionCounters, 0)
	
	for _, child := range n.Children {
		if err := g.visitBlock(child); err != nil {
			return err
		}
	}
	
	// 6. Pop counter
	g.sectionCounters = g.sectionCounters[:len(g.sectionCounters)-1]
	
	g.output.WriteString("</section>\n")
	return nil
}

func (g *Generator) visitGenericBlock(n *ast.GenericBlock) error {
	g.output.WriteString("<" + n.Tag)
	// TODO: attributes
	g.output.WriteString(">")
	
	// Inline Content (e.g. text inside p)
	for _, inline := range n.Content {
		g.visitInline(inline)
	}
	
	// Child Blocks (e.g. nested lists or divs)
	// If there are child blocks, usually we might want a newline before them?
	// But inline content + block content is mixed?
	// Spec: "Block includes start paragraph and all subsequent paragraphs > indent"
	// HTML5: <p> cannot contain blocks (div, ul). Browsers will close <p>.
	// Rite: implicit paragraphs are <p>.
	// If Rite <p> has indented children, they MUST be allowed inside?
	// If child is `x-code` inside `p` -> `<p>...<pre>...</pre></p>` invalid?
	// Yes, <pre> is flow content. <p> allows phrasing content.
	// So if <p> has block children, we might need to close <p>?
	// Or maybe Rite <p> is just a logical block, but output HTML should adjust?
	// For MVP, we just dump it. Browser or proper HTML generator rules can handle details.
	if len(n.Children) > 0 {
		g.output.WriteString("\n")
		for _, child := range n.Children {
			if err := g.visitBlock(child); err != nil {
				return err
			}
		}
	}
	
	g.output.WriteString("</" + n.Tag + ">\n")
	return nil
}

func (g *Generator) visitVerbatim(n *ast.VerbatimNode) error {
	// e.g. <figure><pre><code>...</code></pre><figcaption>...</figcaption></figure>
	// or simple <pre> based on Kind.
	// Spec: x-code -> <pre><code>
	
	g.output.WriteString("<pre>")
	if n.Kind == "x-code" {
		g.output.WriteString("<code>")
	}
	
	g.output.WriteString(htmlEscape(n.Content))
	
	if n.Kind == "x-code" {
		g.output.WriteString("</code>")
	}
	g.output.WriteString("</pre>\n")
	return nil
}

func (g *Generator) visitInline(node ast.Inline) {
	switch n := node.(type) {
	case *ast.TextNode:
		g.output.WriteString(htmlEscape(n.Value))
	case *ast.InlineTagNode:
		// TODO: Recursion for nested inlines?
		// Currently InlineTagNode has just string content?
		// Need to check AST definition.
	default:
		g.output.WriteString(fmt.Sprintf("<!-- unknown inline %T -->", node))
	}
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
