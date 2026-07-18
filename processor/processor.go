package processor

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hesusruiz/riteng/ast"
	"github.com/hesusruiz/riteng/parser"
)

// Global cache for remote inclusions
var (
	remoteCache = make(map[string]string)
	cacheMu     sync.RWMutex
)

// ProcessOptions holds config options for compilation.
type ProcessOptions struct {
	BaseDir string // directory of the main file
}

// Processor coordinates inclusion, numbering, reference resolution, and HTML rendering.
type Processor struct {
	opts       ProcessOptions
	sectionCtr []int
	figCtr     int
	// ID maps for resolving refs
	sections map[string]string // id -> section number + title (e.g., "1.1. Demo 2")
	figures  map[string]string // id -> figure caption (e.g., "Fig 4. An elephant at sunset")
}

func NewProcessor(opts ProcessOptions) *Processor {
	return &Processor{
		opts:     opts,
		sections: make(map[string]string),
		figures:  make(map[string]string),
	}
}

// Process runs the full compilation pipeline on a Document.
func (p *Processor) Process(doc *ast.Document) (string, error) {
	// 1. Resolve includes recursively
	resolvedBlocks, err := p.resolveIncludes(doc.Blocks, p.opts.BaseDir)
	if err != nil {
		return "", err
	}
	doc.Blocks = resolvedBlocks

	// 2. First pass: compute section/figure numbering, collect IDs for x-ref
	p.sectionCtr = make([]int, 10) // support nested sections up to 10 levels deep
	p.figCtr = 0
	if err := p.collectMetadataAndNumber(doc.Blocks, 0); err != nil {
		return "", err
	}

	// 3. Render the processed AST to HTML
	var sb strings.Builder
	if err := p.renderNodes(doc.Blocks, &sb, 0); err != nil {
		return "", err
	}

	return sb.String(), nil
}

// resolveIncludes processes all x-include macros recursively.
func (p *Processor) resolveIncludes(nodes []ast.Node, currentDir string) ([]ast.Node, error) {
	var result []ast.Node

	for _, node := range nodes {
		bNode, ok := node.(*ast.BlockNode)
		if ok && bNode.TagName == "x-include" {
			src, ok := bNode.GetAttribute("src")
			if !ok || src == "" {
				return nil, fmt.Errorf("x-include macro missing 'src' attribute")
			}

			var content string
			var nextDir string

			if strings.HasPrefix(src, "https://") {
				// Remote include
				cacheMu.RLock()
				cached, found := remoteCache[src]
				cacheMu.RUnlock()

				if found {
					content = cached
				} else {
					resp, err := http.Get(src)
					if err != nil {
						return nil, fmt.Errorf("error fetching remote include %s: %w", src, err)
					}
					defer resp.Body.Close()
					if resp.StatusCode != http.StatusOK {
						return nil, fmt.Errorf("remote include %s returned status %s", src, resp.Status)
					}
					bodyBytes, err := io.ReadAll(resp.Body)
					if err != nil {
						return nil, fmt.Errorf("error reading remote include %s: %w", src, err)
					}
					content = string(bodyBytes)
					cacheMu.Lock()
					remoteCache[src] = content
					cacheMu.Unlock()
				}
				nextDir = currentDir // keep same base directory for remote includes
			} else {
				// Local include
				// Security check: cannot start with '/' or '\', cannot contain '..'
				if strings.HasPrefix(src, "/") || strings.HasPrefix(src, "\\") || strings.Contains(src, "..") {
					return nil, fmt.Errorf("security violation: local include path '%s' cannot start with '/' or contain '..'", src)
				}

				fullPath := filepath.Join(currentDir, src)
				fileBytes, err := os.ReadFile(fullPath)
				if err != nil {
					return nil, fmt.Errorf("error reading local include %s: %w", src, err)
				}
				content = string(fileBytes)
				nextDir = filepath.Dir(fullPath)
			}

			// Parse the included content
			incDoc, err := parser.Parse(strings.NewReader(content))
			if err != nil {
				return nil, fmt.Errorf("error parsing included file %s: %w", src, err)
			}

			// Recursively resolve includes inside the included document
			incBlocks, err := p.resolveIncludes(incDoc.Blocks, nextDir)
			if err != nil {
				return nil, err
			}

			// Push indentation level of the macro into included blocks
			adjustIndentation(incBlocks, bNode.Indent)
			result = append(result, incBlocks...)
		} else {
			if ok && len(bNode.Children) > 0 {
				resChildren, err := p.resolveIncludes(bNode.Children, currentDir)
				if err != nil {
					return nil, err
				}
				bNode.Children = resChildren
			}
			result = append(result, node)
		}
	}

	return result, nil
}

func adjustIndentation(nodes []ast.Node, amount int) {
	for _, node := range nodes {
		if bNode, ok := node.(*ast.BlockNode); ok {
			bNode.Indent += amount
			adjustIndentation(bNode.Children, amount)
		}
	}
}

// collectMetadataAndNumber does a first pass to assign section/figure numbers and populate the ID maps.
func (p *Processor) collectMetadataAndNumber(nodes []ast.Node, sectionDepth int) error {
	for _, node := range nodes {
		bNode, ok := node.(*ast.BlockNode)
		if !ok {
			continue
		}

		id, hasId := bNode.GetAttribute("id")

		if bNode.TagName == "section" {
			// Determine if it is unnumbered
			_, isUnnumbered := bNode.GetAttribute("unnumbered")
			var secNum string

			if !isUnnumbered {
				// Increment the counter for this level and reset sub-levels
				if sectionDepth < len(p.sectionCtr) {
					p.sectionCtr[sectionDepth]++
					for i := sectionDepth + 1; i < len(p.sectionCtr); i++ {
						p.sectionCtr[i] = 0
					}
				}
				// Build section number string
				var parts []string
				for i := 0; i <= sectionDepth; i++ {
					parts = append(parts, fmt.Sprintf("%d", p.sectionCtr[i]))
				}
				secNum = strings.Join(parts, ".") + "."
				bNode.SetAttribute("__sec_prefix", secNum)
			}

			// Store metadata if ID is present
			if hasId && id != "" {
				title := strings.TrimSpace(bNode.Rest)
				if secNum != "" {
					p.sections[id] = secNum + " " + title
				} else {
					p.sections[id] = title
				}
			}

			// Recurse into children, incrementing sectionDepth
			if err := p.collectMetadataAndNumber(bNode.Children, sectionDepth+1); err != nil {
				return err
			}
		} else if bNode.TagName == "x-fig" {
			p.figCtr++
			caption := strings.TrimSpace(bNode.Rest)
			figLabel := fmt.Sprintf("Fig %d. %s", p.figCtr, caption)
			if hasId && id != "" {
				p.figures[id] = figLabel
			}
			// Save computed fig label on the node for rendering phase
			bNode.SetAttribute("__fig_label", figLabel)

			if err := p.collectMetadataAndNumber(bNode.Children, sectionDepth); err != nil {
				return err
			}
		} else {
			if hasId && id != "" {
				// Default map to rest/text if target exists (e.g. x-code figcaption or generic element)
				p.figures[id] = strings.TrimSpace(bNode.Rest)
			}
			if err := p.collectMetadataAndNumber(bNode.Children, sectionDepth); err != nil {
				return err
			}
		}
	}
	return nil
}

// renderNodes converts AST nodes to HTML strings.
func (p *Processor) renderNodes(nodes []ast.Node, sb *strings.Builder, sectionDepth int) error {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.BlockNode:
			if err := p.renderBlock(n, sb, sectionDepth); err != nil {
				return err
			}
		case *ast.ParagraphNode:
			// Generally ParagraphNodes are inline children, but render them just in case
			sb.WriteString(strings.Repeat(" ", n.Indent))
			sb.WriteString("<p>")
			if err := p.renderInlineNodes(n.Children, sb); err != nil {
				return err
			}
			sb.WriteString("</p>\n")
		}
	}
	return nil
}

func (p *Processor) renderBlock(b *ast.BlockNode, sb *strings.Builder, sectionDepth int) error {
	indentStr := strings.Repeat(" ", b.Indent)

	switch b.TagName {
	case "section":
		headerLevel := 2 + sectionDepth // starts at h2
		if headerLevel > 6 {
			headerLevel = 6
		}

		prefix := b.GetAttributeOrEmpty("__sec_prefix")
		if prefix != "" {
			prefix = prefix + " "
		}

		// Render starting tag with attributes
		sb.WriteString(indentStr)
		sb.WriteString("<section")
		p.renderAttributes(b.Attributes, sb, "unnumbered")
		sb.WriteString(">\n")

		// Render header inside section
		sb.WriteString(indentStr)
		fmt.Fprintf(sb, "  <h%d>", headerLevel)
		sb.WriteString(prefix)
		sb.WriteString(html.EscapeString(b.Rest))
		fmt.Fprintf(sb, "</h%d>\n", headerLevel)

		// Render children
		if err := p.renderNodes(b.Children, sb, sectionDepth+1); err != nil {
			return err
		}

		sb.WriteString(indentStr)
		sb.WriteString("</section>\n")

	case "x-fig":
		figLabel := b.GetAttributeOrEmpty("__fig_label")
		src, _ := b.GetAttribute("src")

		sb.WriteString(indentStr)
		sb.WriteString("<figure>\n")

		sb.WriteString(indentStr)
		sb.WriteString("  <img src=\"")
		sb.WriteString(html.EscapeString(src))
		sb.WriteString("\" alt=\"")
		sb.WriteString(html.EscapeString(b.Rest))
		sb.WriteString("\" />\n")

		sb.WriteString(indentStr)
		sb.WriteString("  <figcaption>")
		sb.WriteString(html.EscapeString(figLabel))
		sb.WriteString("</figcaption>\n")

		sb.WriteString(indentStr)
		sb.WriteString("</figure>\n")

	case "x-quote":
		author := strings.TrimSpace(b.Rest)
		sb.WriteString(indentStr)
		sb.WriteString("<figure>\n")

		if author != "" {
			sb.WriteString(indentStr)
			sb.WriteString("  <figcaption><b>")
			sb.WriteString(html.EscapeString(author))
			sb.WriteString("</b></figcaption>\n")
		}

		sb.WriteString(indentStr)
		sb.WriteString("  <blockquote>\n")

		// Quote contents are child blocks
		for _, child := range b.Children {
			if childBlock, ok := child.(*ast.BlockNode); ok {
				// Render block directly inside blockquote
				if err := p.renderBlock(childBlock, sb, sectionDepth); err != nil {
					return err
				}
			}
		}

		sb.WriteString(indentStr)
		sb.WriteString("  </blockquote>\n")

		sb.WriteString(indentStr)
		sb.WriteString("</figure>\n")

	case "x-code":
		idAttr, hasId := b.GetAttribute("id")

		sb.WriteString(indentStr)
		sb.WriteString("<figure>\n")

		sb.WriteString(indentStr)
		sb.WriteString("  <pre><code>")

		// Write verbatim lines
		for idx, line := range b.Lines {
			sb.WriteString(html.EscapeString(line))
			if idx < len(b.Lines)-1 {
				sb.WriteString("\n")
			}
		}

		sb.WriteString("</code></pre>\n")

		if b.Rest != "" {
			sb.WriteString(indentStr)
			sb.WriteString("  <figcaption")
			if hasId {
				fmt.Fprintf(sb, " id=\"%s\"", html.EscapeString(idAttr))
			}
			sb.WriteString(">")
			sb.WriteString(html.EscapeString(b.Rest))
			sb.WriteString("</figcaption>\n")
		}

		sb.WriteString(indentStr)
		sb.WriteString("</figure>\n")

	case "p", "div", "article", "header", "footer", "aside", "nav", "blockquote", "figure", "figcaption", "table", "thead", "tbody", "tr", "th", "td", "form", "main", "ul", "ol", "li":
		sb.WriteString(indentStr)
		fmt.Fprintf(sb, "<%s", b.TagName)
		p.renderAttributes(b.Attributes, sb)
		sb.WriteString(">")

		if b.IsVerbatim {
			for _, line := range b.Lines {
				sb.WriteString(html.EscapeString(line))
				sb.WriteString("\n")
			}
		} else {
			// Parse Rest as inline nodes
			if b.Rest != "" {
				inlines := parser.ParseInline(b.Rest)
				if err := p.renderInlineNodes(inlines, sb); err != nil {
					return err
				}
			}

			if len(b.Children) > 0 {
				sb.WriteString("\n")
				if err := p.renderNodes(b.Children, sb, sectionDepth); err != nil {
					return err
				}
				sb.WriteString(indentStr)
			}
		}

		fmt.Fprintf(sb, "</%s>\n", b.TagName)

	default:
		// Unknown tag: render as standard tag
		sb.WriteString(indentStr)
		fmt.Fprintf(sb, "<%s", b.TagName)
		p.renderAttributes(b.Attributes, sb)
		sb.WriteString(">")
		if b.Rest != "" {
			inlines := parser.ParseInline(b.Rest)
			if err := p.renderInlineNodes(inlines, sb); err != nil {
				return err
			}
		}
		if len(b.Children) > 0 {
			sb.WriteString("\n")
			if err := p.renderNodes(b.Children, sb, sectionDepth); err != nil {
				return err
			}
			sb.WriteString(indentStr)
		}
		fmt.Fprintf(sb, "</%s>\n", b.TagName)
	}

	return nil
}

func (p *Processor) renderInlineNodes(nodes []ast.InlineNode, sb *strings.Builder) error {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.TextNode:
			sb.WriteString(html.EscapeString(n.Content))
		case *ast.StrongNode:
			sb.WriteString("<strong>")
			sb.WriteString(html.EscapeString(n.Content))
			sb.WriteString("</strong>")
		case *ast.EmphasisNode:
			sb.WriteString("<em>")
			sb.WriteString(html.EscapeString(n.Content))
			sb.WriteString("</em>")
		case *ast.CodeNode:
			sb.WriteString("<code>")
			sb.WriteString(html.EscapeString(n.Content))
			sb.WriteString("</code>")
		case *ast.InlineTagNode:
			if err := p.renderInlineTag(n, sb); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Processor) renderInlineTag(t *ast.InlineTagNode, sb *strings.Builder) error {
	if t.TagName == "x-ref" {
		href, ok := t.GetAttribute("href")
		if !ok || href == "" {
			return fmt.Errorf("x-ref tag missing 'href' attribute")
		}

		// Look up ID in sections or figures/tables
		var linkText string
		if secText, ok := p.sections[href]; ok {
			linkText = "Section " + secText
		} else if figText, ok := p.figures[href]; ok {
			linkText = figText
		} else {
			return fmt.Errorf("referenced ID '%s' not found in document", href)
		}

		fmt.Fprintf(sb, "<a href=\"#%s\">%s</a>", html.EscapeString(href), html.EscapeString(linkText))
		return nil
	}

	// Normal inline tag
	fmt.Fprintf(sb, "<%s", t.TagName)
	p.renderAttributes(t.Attributes, sb)
	sb.WriteString(">")

	if len(t.Children) > 0 {
		if err := p.renderInlineNodes(t.Children, sb); err != nil {
			return err
		}
	}
	fmt.Fprintf(sb, "</%s>", t.TagName)
	return nil
}

func (p *Processor) renderAttributes(attrs []ast.Attribute, sb *strings.Builder, ignoreKeys ...string) {
	for _, attr := range attrs {
		// skip internal/ignore attributes
		if strings.HasPrefix(attr.Key, "__") {
			continue
		}
		ignored := false
		for _, ignoreKey := range ignoreKeys {
			if attr.Key == ignoreKey {
				ignored = true
				break
			}
		}
		if ignored {
			continue
		}

		if attr.Value == "" {
			// Boolean attribute
			fmt.Fprintf(sb, " %s", html.EscapeString(attr.Key))
		} else {
			fmt.Fprintf(sb, " %s=\"%s\"", html.EscapeString(attr.Key), html.EscapeString(attr.Value))
		}
	}
}
