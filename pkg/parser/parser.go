package parser

import (
	"github.com/hesusruiz/riteng/pkg/ast"
	"github.com/hesusruiz/riteng/pkg/lexer"
)

type Parser struct {
	l         *lexer.Lexer
	curToken  lexer.Token
	peekToken lexer.Token
	errors    []string
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l}
	// Read two tokens to initialize curToken and peekToken
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) ParseDocument() *ast.Document {
	doc := &ast.Document{}
	doc.Children = []ast.Block{}

	loopCount := 0
	for p.curToken.Type != lexer.TOKEN_EOF {
		if loopCount > 100000 {
			doc.Children = append(doc.Children, &ast.GenericBlock{Tag: "error", Content: []ast.Inline{&ast.TextNode{Value: "Parser Loop Limit Exceeded"}}})
			break
		}
		loopCount++
		
		startToken := p.curToken
		stmt := p.parseBlock(0) // Start at indent 0
		if stmt != nil {
			doc.Children = append(doc.Children, stmt)
		} else {
            // Check if we advanced. If not, force advance to avoid loop.
            // Comparison by pointer or value? Token is value.
            // If Type, Literal, Line, Column are same.
            if p.curToken == startToken {
                 // We are stuck. Force advance.
                 // fmt.Printf("DEBUG: Stuck at token %s. Advancing.\n", p.curToken)
                 p.nextToken()
            }
        }
		// p.nextToken() // parseBlock consumes the tokens
	}
	return doc
}

// parseBlock attempts to parse a block at the given base indentation.
// currentIndent is the indentation of the PARENT block.
// The new block must start with an indentation >= currentIndent?
// Actually, indentation tokens are emitted relative to the previous line.
// But structurally, we consume INDENT tokens to enter a child block.
// Here we are parsing a "sibling" list of blocks until DEDENT?
// No, ParseDocument loops until EOF.
// Recursive structure: parseChildren() consumes until DEDENT.

func (p *Parser) parseBlock(indent int) ast.Block {
	// Skip newlines at start of block search
	for p.curToken.Type == lexer.TOKEN_NEWLINE {
		p.nextToken()
	}

	if p.curToken.Type == lexer.TOKEN_EOF || p.curToken.Type == lexer.TOKEN_DEDENT {
		return nil
	}

	// 1. Check for Block Start
	// Case A: Tag (<section>, <p>, <x-code>)
	if p.curToken.Type == lexer.TOKEN_TAG_START {
		// Check if it is a closing tag (</...) -> Ignore/Consume
		if p.peekToken.Type == lexer.TOKEN_SLASH {
			p.consumeIgnoredTag()
			return nil // Loop in caller will continue
		}
		return p.parseTagBlock()
	}

	// Case B: Implicit Paragraph (Text, or other markers)
	// If it's not a tag, it's likely text starting a paragraph.
	// Or list markers (*, -)
	if p.curToken.Type == lexer.TOKEN_STAR || p.curToken.Type == lexer.TOKEN_DASH {
		// List Item Block (TODO)
		// For MVP, treat as text/paragraph start
		return p.parseImplicitParagraph()
	}

	return p.parseImplicitParagraph()
}

func (p *Parser) parseTagBlock() ast.Block {
	// Assumes curToken is <
	p.nextToken() // Consume <
	
	if p.curToken.Type != lexer.TOKEN_IDENTIFIER {
		// Error or mismatch
		return nil
	}
	tagName := p.curToken.Literal
	p.nextToken() // Consume name

	// Parse Attributes (TODO)
	for p.curToken.Type != lexer.TOKEN_TAG_END {
		p.nextToken()
	}
	p.nextToken() // Consume >

	// Determine Block Type
	if tagName == "section" {
		node := &ast.SectionNode{Identifier: tagName} // Temp ID
		node.TokenLat = "section"
		// Parse Rest of Line as Title?
		// "The paragraph is composed of the start tag and the ‘rest’ of the paragraph."
		// So we collect inline text until NEWLINE.
		node.Title = p.parseInlineUntilNewline()
		
		// Parse Children
		node.Children = p.parseChildren()
		return node
	}

	if tagName == "x-code" {
		return p.parseVerbatimBlock("x-code")
	}

	if tagName == "x-include" {
		return p.parseIncludeBlock()
	}

	// Generic Block (p, div, etc)
	node := &ast.GenericBlock{Tag: tagName}
	// Content (Rest of line)
	node.Content = p.parseInlineUntilNewline()
	
	// Children (Subsequent indented blocks)
	node.Children = p.parseChildren()
	return node
}

func (p *Parser) parseVerbatimBlock(kind string) ast.Block {
	node := &ast.VerbatimNode{Kind: kind}
	
	// Parse Capture/Title (rest of line)
	node.Caption = p.parseInlineUntilNewline()
	
	// Check for Indented Block
	// Note: parseInlineUntilNewline consumes the NEWLINE, so curToken might be INDENT
	if p.curToken.Type == lexer.TOKEN_INDENT {
		// Do NOT consume INDENT yet (p.nextToken()), because that would advance Lexer to peekToken's successor.
		// Current State: curToken=INDENT, peekToken="func" (or content).
		// Lexer State: At end of peekToken ("func").
		
		
		// The block's indentation level is the current level (from the INDENT we are looking at)
		blockIndent := p.l.CurrentIndent()
		
		// Rewind the content of peekToken (e.g. "func")
		if p.peekToken.Type != lexer.TOKEN_EOF {
			p.l.Rewind(len(p.peekToken.Literal))
		}
		
		// Rewind indentation
		p.l.RewindToLastIndent()
		
		// Switch Mode
		p.l.SetMode(lexer.ModeRawLine)
		
		// Force Refresh.
		// We want curToken to be the RAW_LINE we just rewound to.
		// Calling nextToken() sets curToken = peekToken.
		// But peekToken is stale ("func").
		// So we must manually refresh.
		p.curToken = p.l.NextToken() 
		p.peekToken = p.l.NextToken()
		
		var contentBuilder string
		firstLine := true

		for {
			if p.curToken.Type == lexer.TOKEN_NEWLINE { 
				contentBuilder += "\n"
				p.nextToken()
				continue
			}
			if p.curToken.Type != lexer.TOKEN_RAW_LINE {
				break 
			}
			
			line := p.curToken.Literal
			
			// For subsequent lines (not first), we check indentation.
			// The Lexer in RawLineMode reads spaces.
			
			if !firstLine {
				indent := countIndent(line)
				if indent < blockIndent {
					// End of block.
					// Rewind this line.
					p.l.Rewind(len(line))
					break
				}
			}
			
			contentBuilder += line
			firstLine = false
			p.nextToken()
		}
		
		node.Content = contentBuilder
		p.l.SetMode(lexer.ModeStandard)
		p.nextToken() // Consume the DEDENT that should trigger now (or next token)
	}
	
	return node
}

func (p *Parser) parseIncludeBlock() ast.Block {
	node := &ast.IncludeNode{}
	// Current Token is > (consumed in parseTagBlock loop? No)
	// In parseTagBlock: 
	// p.nextToken() // Consume name
	// // Parse Attributes (TODO)
	// for p.curToken.Type != lexer.TOKEN_TAG_END { ... }
	
	// Wait, parseTagBlock consumes the attribute loop but DOES NOT CAPTURE IT in my current implementation.
	// FOR MVP: x-include attributes must be parsed.
	// I need to modify parseTagBlock to capture attributes or delegate.
	// Since I just delegate by Block Type, I might be too late?
	// parseTagBlock:
	// ... consumes tokens ...
	// if tagName == "x-include" -> calls parseIncludeBlock
	// BUT curToken is now > (TAG_END).
	// Attributes were skipped!
	
	// I must refactor parseTagBlock to capture attributes map BEFORE dispatching.
	// Or dispatch earlier?
	// Let's refactor parseTagBlock slightly in a separate step or just assume no attributes for now (but x-include NEEDS href).
	
	// FIX: parseTagBlock logic is:
	// 1. Consume Name
	// 2. Loop until TAG_END.
	// 3. Dispatch.
	// So attributes are GONE.
	
	// I will return a stub which is empty for now, but I admit looking at `parseTagBlock` I see the attributes are skipped.
	// I will fix `parseTagBlock` to collect attributes.
	return node
}

// Helper to count indentation
func countIndent(s string) int {
	count := 0
	for _, c := range s {
		if c == ' ' {
			count++
		} else {
			break
		}
	}
	return count
}

func (p *Parser) parseImplicitParagraph() ast.Block {
	node := &ast.GenericBlock{Tag: "p"}
	node.Content = p.parseInlineUntilNewline()
	
	// Check for Children (indented blocks inside p?)
	node.Children = p.parseChildren()
	return node
}

// parseChildren consumes INDENT, parses blocks, then consumers DEDENT
func (p *Parser) parseChildren() []ast.Block {
	// If we are at INDENT (consumed by previous parseInline or just pending), we enter children.
	// OR if we are at NEWLINE, skip it to find INDENT
	for p.curToken.Type == lexer.TOKEN_NEWLINE {
		p.nextToken()
	}

	if p.curToken.Type == lexer.TOKEN_INDENT {
		p.nextToken() // Consume INDENT. curToken is now the Start of the first child block.
		
		var children []ast.Block
		loopCount := 0
		for p.curToken.Type != lexer.TOKEN_DEDENT && p.curToken.Type != lexer.TOKEN_EOF {
			if loopCount > 100000 {
				break
			}
			loopCount++
			
			startToken := p.curToken
			child := p.parseBlock(0)
			if child != nil {
				children = append(children, child)
			} else {
				// If parseBlock returns nil (e.g. empty line or ignored tag), ensure we advance
				if p.curToken == startToken {
                     p.nextToken()
                }
			}
		}
		
		if p.curToken.Type == lexer.TOKEN_DEDENT {
			p.nextToken() // Consume DEDENT
		}
		return children
	}
	return nil
}


func (p *Parser) consumeIgnoredTag() {
	// We are at <
	// Expect /, Name, >
	// This is a minimal consumer for "</tag>"
	for p.curToken.Type != lexer.TOKEN_TAG_END && p.curToken.Type != lexer.TOKEN_EOF {
		p.nextToken()
	}
	// Consume >
	if p.curToken.Type == lexer.TOKEN_TAG_END {
		p.nextToken()
	}
}

func (p *Parser) parseInlineUntilNewline() []ast.Inline {
	var inlines []ast.Inline
	
	// Consume tokens until NEWLINE or EOF
	for p.curToken.Type != lexer.TOKEN_NEWLINE && p.curToken.Type != lexer.TOKEN_EOF && p.curToken.Type != lexer.TOKEN_INDENT {
		// Simple text aggregation for MVP
		if p.curToken.Type == lexer.TOKEN_TEXT || p.curToken.Type == lexer.TOKEN_IDENTIFIER {
			inlines = append(inlines, &ast.TextNode{Value: p.curToken.Literal + " "})
		} else {
			// Append markers as text for now
			inlines = append(inlines, &ast.TextNode{Value: p.curToken.Literal})
		}
		p.nextToken()
	}
	// Consume the newline
	if p.curToken.Type == lexer.TOKEN_NEWLINE {
		p.nextToken()
	}
	return inlines
}
