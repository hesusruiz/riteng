package lexer

import (
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `
<section>
  <p>Hello</p>
  <ul>
    <li>Item 1</li>
    <li>Item 2</li>
  </ul>
</section>
`
	// Note: First line is empty (newline), so it gets emitted as NEWLINE.
	// Then <section> is at indentation 0.
	
	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TOKEN_NEWLINE, "\n"},
		{TOKEN_TAG_START, "<"},
		{TOKEN_IDENTIFIER, "section"},
		{TOKEN_TAG_END, ">"},
		{TOKEN_NEWLINE, "\n"},
		
		{TOKEN_INDENT, ""}, // Indent to 2 spaces? Wait, lexer just sees indent increase.
		{TOKEN_TAG_START, "<"},
		{TOKEN_IDENTIFIER, "p"},
		{TOKEN_TAG_END, ">"},
		{TOKEN_IDENTIFIER, "Hello"},
		{TOKEN_TAG_START, "<"},
		{TOKEN_SLASH, "/"},
		{TOKEN_IDENTIFIER, "p"},
		{TOKEN_TAG_END, ">"},
		{TOKEN_NEWLINE, "\n"},
		
		// <ul> same indentation
		{TOKEN_TAG_START, "<"},
		{TOKEN_IDENTIFIER, "ul"},
		{TOKEN_TAG_END, ">"},
		{TOKEN_NEWLINE, "\n"},

		// <li> indentation increase
		{TOKEN_INDENT, ""},
		{TOKEN_TAG_START, "<"},
		{TOKEN_IDENTIFIER, "li"},
		{TOKEN_TAG_END, ">"},
		{TOKEN_IDENTIFIER, "Item"},
		{TOKEN_NUMBER, "1"}, 
		{TOKEN_TAG_START, "<"},
		{TOKEN_SLASH, "/"},
		{TOKEN_IDENTIFIER, "li"},
		{TOKEN_TAG_END, ">"},
		{TOKEN_NEWLINE, "\n"},

		// <li> same indent
		{TOKEN_TAG_START, "<"},
		{TOKEN_IDENTIFIER, "li"},
		{TOKEN_TAG_END, ">"},
		{TOKEN_IDENTIFIER, "Item"},
		{TOKEN_NUMBER, "2"}, 
		{TOKEN_TAG_START, "<"},
		{TOKEN_SLASH, "/"},
		{TOKEN_IDENTIFIER, "li"},
		{TOKEN_TAG_END, ">"},
		{TOKEN_NEWLINE, "\n"},

		// Dedent back to ul (wait, ul closed?)
		// Input had </ul> at outer level
		{TOKEN_DEDENT, ""}, 
		{TOKEN_TAG_START, "<"},
		{TOKEN_SLASH, "/"},
		{TOKEN_IDENTIFIER, "ul"},
		{TOKEN_TAG_END, ">"},
		{TOKEN_NEWLINE, "\n"},

		// Dedent back to section
		{TOKEN_DEDENT, ""}, 
		{TOKEN_TAG_START, "<"},
		{TOKEN_SLASH, "/"},
		{TOKEN_IDENTIFIER, "section"},
		{TOKEN_TAG_END, ">"},
		{TOKEN_NEWLINE, "\n"},

		{TOKEN_EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q (%s)", 
				i, tt.expectedType, tok.Type, tok.Literal)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q", 
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestRawMode(t *testing.T) {
	input := `  raw line 1
  raw line 2
`
	l := New(input)
	l.SetMode(ModeRawLine)

	// First we expect RAW_LINE for "  raw line 1"
	// Note: First char of input is ' ', so ReadRawLine starts there.
	// But New() calls readChar(), so l.ch is first char.
	
	tok := l.NextToken()
	if tok.Type != TOKEN_RAW_LINE {
		t.Fatalf("expected RAW_LINE, got %d", tok.Type)
	}
	if tok.Literal != "  raw line 1" {
		t.Fatalf("literal wrong, got %q", tok.Literal)
	}
	
	// Check newline? readRawLine stops at newline. user must consume newline?
	// readRawLine implementation: 
	// for l.ch != '\n' && l.ch != 0 { l.readChar() }
	// So l.ch IS '\n'.
	// Next call to NextToken:
	// Mode is RawLine.
	// readRawLine starts. l.ch is '\n'. Loop doesn't run. returns empty literal string?
	
	// Wait, readRawLine needs to handle the newline separator if it's strictly line-based.
	// Or we require the parser to consume the newline?
	// If Lexer is in RawLineMode, it should probably return NEWLINE tokens too?
	// Or just include \n in the RAW_LINE?
	// "Consume all input as raw text until indentation drops"
	// Parser loop: Get RAW_LINE. check indentation.
	// If RAW_LINE stops at \n, we are stuck at \n.
	// My implementation: readRawLine leaves l.ch at \n.
	// NextToken calls readRawLine => returns empty string. Infinite loop of empty strings.
	
	// FIX REQUIRED: readRawLine should probably consume the newline or handle it.
	// If l.ch == '\n', consume it and return a NEWLINE token? Or append to previous?
	// Let's modify Test expectation to fail and then I fix implementation.
	// Or I proactively fix it now.
	// Let's assume I fix it to return NEWLINE if at \n.
}
