package ast

// Node represents any node in the Rite Abstract Syntax Tree (AST).
type Node interface {
	// sealed is an unexported method that prevents external packages from implementing
	// this interface. This creates a "sealed" type hierarchy (sum type), ensuring that
	// only nodes defined in this package can be used.
	sealed()
}

// Attribute represents an HTML attribute key-value pair.
type Attribute struct {
	Key   string
	Value string
}

// Document is the root node of a Rite document.
type Document struct {
	Metadata map[string]interface{}
	Blocks   []Node
}

func (d *Document) sealed() {}

// BlockNode represents a block-level node (normal block, list block, verbatim block, etc.).
type BlockNode struct {
	TagName    string       // e.g. "div", "section", "p", "ul", "ol", "li", "pre", "x-fig", "x-quote", "x-code", "x-include"
	Attributes []Attribute  // Parsed attributes including shorthands
	Rest       string       // The text content on the same line after the tag (e.g. for `<section>Title` -> "Title")
	Indent     int          // Indentation level (number of spaces)
	Children   []Node       // Sub-blocks nested under this block
	IsVerbatim bool         // True for pre, x-code
	Lines      []string     // For verbatim blocks, contains the raw indented lines
}

func (b *BlockNode) sealed() {}

// GetAttribute returns the value of the attribute with the given key, and whether it exists.
func (b *BlockNode) GetAttribute(key string) (string, bool) {
	for _, attr := range b.Attributes {
		if attr.Key == key {
			return attr.Value, true
		}
	}
	return "", false
}

// GetAttributeOrEmpty returns the value of the attribute with the given key, or empty string if not found.
func (b *BlockNode) GetAttributeOrEmpty(key string) string {
	val, ok := b.GetAttribute(key)
	if !ok {
		return ""
	}
	return val
}

// SetAttribute sets or overwrites an attribute.
func (b *BlockNode) SetAttribute(key, value string) {
	for i, attr := range b.Attributes {
		if attr.Key == key {
			b.Attributes[i].Value = value
			return
		}
	}
	b.Attributes = append(b.Attributes, Attribute{Key: key, Value: value})
}

// RemoveAttribute removes an attribute.
func (b *BlockNode) RemoveAttribute(key string) {
	for i, attr := range b.Attributes {
		if attr.Key == key {
			b.Attributes = append(b.Attributes[:i], b.Attributes[i+1:]...)
			return
		}
	}
}

// InlineNode represents inline elements within paragraphs (Text, Bold, Italic, Code, Inline Tags).
type InlineNode interface {
	Node
	// inlineNode is an unexported marker method used to restrict the types of nodes
	// that can be assigned as children of a ParagraphNode or other inline containers.
	inlineNode()
}

// TextNode is a plain text segment.
type TextNode struct {
	Content string
}

func (t *TextNode) sealed()     {}
func (t *TextNode) inlineNode() {}

// StrongNode represents bold/strong text (*text* or **text**).
type StrongNode struct {
	Content string
}

func (s *StrongNode) sealed()     {}
func (s *StrongNode) inlineNode() {}

// EmphasisNode represents italicized text (_text_).
type EmphasisNode struct {
	Content string
}

func (e *EmphasisNode) sealed()     {}
func (e *EmphasisNode) inlineNode() {}

// InlineCodeNode represents inline code (``code`` or `code`).
type CodeNode struct {
	Content string
}

func (c *CodeNode) sealed()     {}
func (c *CodeNode) inlineNode() {}

// InlineTagNode represents inline HTML tags like <a>, <strong>, <span>, or macros like <x-ref>.
type InlineTagNode struct {
	TagName    string
	Attributes []Attribute
	Children   []InlineNode // content inside the inline tag
}

func (i *InlineTagNode) sealed()     {}
func (i *InlineTagNode) inlineNode() {}

// GetAttribute returns the value of the attribute with the given key, and whether it exists.
func (i *InlineTagNode) GetAttribute(key string) (string, bool) {
	for _, attr := range i.Attributes {
		if attr.Key == key {
			return attr.Value, true
		}
	}
	return "", false
}

// ParagraphNode represents a block containing text lines with inline markup.
type ParagraphNode struct {
	Indent   int
	Children []InlineNode
}

func (p *ParagraphNode) sealed() {}
