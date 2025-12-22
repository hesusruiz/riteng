package ast

import (
	"fmt"
	"strings"
)

// Node is the base interface for all AST nodes
type Node interface {
	TokenLiteral() string
	String() string
	Position() (string, int) // Filename, Line
}

type BaseNode struct {
	TokenLat   string
	Line       int
	Filename   string
}

func (b *BaseNode) Position() (string, int) {
	return b.Filename, b.Line
}

// Statements (Block-level elements)
type Block interface {
	Node
	blockNode()
}

// Inline (Text-level elements)
type Inline interface {
	Node
	inlineNode()
}

// Document is the root node
type Document struct {
	Children []Block
	Filename string
}

func (d *Document) TokenLiteral() string { return "" }
func (d *Document) String() string {
	var out strings.Builder
	for _, s := range d.Children {
		out.WriteString(s.String())
	}
	return out.String()
}
func (d *Document) Position() (string, int) { return d.Filename, 1 }

// SectionNode represents <section> blocks
type SectionNode struct {
	BaseNode
	Level      int // h1, h2, etc. (calculated later or during parse?)
	Identifier string
	Title      []Inline
	Children   []Block
}

func (s *SectionNode) TokenLiteral() string { return s.TokenLat }
func (s *SectionNode) String() string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("<Section ID=%q>", s.Identifier))
	for _, c := range s.Children {
		out.WriteString(c.String())
	}
	out.WriteString("</Section>")
	return out.String()
}
func (s *SectionNode) blockNode() {}

// VerbatimNode represents pre, x-code
type VerbatimNode struct {
	BaseNode
	Kind     string // "pre", "x-code"
	Content  string // The raw text content
	Alt      string // Caption or alt text
	Children []Block // Captions can be blocks? Spec says "x-fig uses the rest text to build caption" which is parsed inline? 
	                 // x-code inner block is raw text. Caption is usually from attributes or adjacent? 
                     // Spec: "rest of paragraph" -> Title/Caption.
    // For x-code, the "rest" is processed? 
    // "The first line after the x-code tag must be indented... interior block"
    // The "rest" of the start line is caption/title? 
	Caption []Inline
}

func (v *VerbatimNode) TokenLiteral() string { return v.TokenLat }
func (v *VerbatimNode) String() string { return fmt.Sprintf("<Verbatim Kind=%q>%s</Verbatim>", v.Kind, v.Content) }
func (v *VerbatimNode) blockNode() {}

// GenericBlock represents standard HTML blocks (p, div, ul, li currently handled implicitly or explicitly)
type GenericBlock struct {
	BaseNode
	Tag        string
	Attributes map[string]string
	Children   []Block // For container blocks (div, blockquote)
	Content    []Inline // For leaf blocks (p, li) - wait, li can have children blocks? 
	// Rite spec 2.3: "Block includes start paragraph and all subsequent paragraphs > indent".
	// "Child block starts with a block tag".
	// So a 'p' block generally doesn't have child blocks, it ends when indent matches or drops.
	// But 'li' is a block. 
	// Let's use a generic structure where Content is the immediate text, and Children are sub-blocks.
}

func (b *GenericBlock) TokenLiteral() string { return b.TokenLat }
func (b *GenericBlock) String() string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("<%s>", b.Tag))
	// Content
	for _, c := range b.Content {
		out.WriteString(c.String())
	}
	// Children
	for _, c := range b.Children {
		out.WriteString(c.String())
	}
	out.WriteString(fmt.Sprintf("</%s>", b.Tag))
	return out.String()
}
func (b *GenericBlock) blockNode() {}

// TextNode holds plain text
type TextNode struct {
	BaseNode
	Value string
}

func (t *TextNode) TokenLiteral() string { return t.Value }
func (t *TextNode) String() string       { return t.Value }
func (t *TextNode) inlineNode()          {}
func (t *TextNode) blockNode()           {} // Sometimes text is treated as block content directly? No, usually wrapped in P.

// InlineTagNode (bold, italic, etc)
type InlineTagNode struct {
	BaseNode
	Tag      string
	Children []Inline
}

func (i *InlineTagNode) TokenLiteral() string { return i.TokenLat }
func (i *InlineTagNode) String() string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("<%s>", i.Tag))
	for _, c := range i.Children {
		out.WriteString(c.String())
	}
	out.WriteString(fmt.Sprintf("</%s>", i.Tag))
	return out.String()
}
func (i *InlineTagNode) inlineNode() {}
