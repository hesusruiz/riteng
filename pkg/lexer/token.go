package lexer

import "fmt"

type TokenType int

const (
	// Special
	TOKEN_EOF     TokenType = iota
	TOKEN_ERROR
	
	// Indentation
	TOKEN_INDENT  // Shift Right
	TOKEN_DEDENT  // Shift Left
	TOKEN_NEWLINE // Line end

	// Content
	TOKEN_TEXT    // Standard text content
	TOKEN_RAW_LINE // Full line content (for x-code generic processing)

	// Block Markers
	TOKEN_TAG_START // <
	TOKEN_TAG_END   // >
	TOKEN_SLASH     // /
	TOKEN_EQUALS    // =
	TOKEN_STRING    // "value" or 'value'
	TOKEN_IDENTIFIER // tag names, attribute keys

	// Markdown/Riteng Markers
	TOKEN_HASH       // #
	TOKEN_STAR       // *
	TOKEN_UNDERSCORE // _
	TOKEN_BACKTICK   // `
	TOKEN_DASH       // -
	TOKEN_DOT        // .
	TOKEN_NUMBER     // 1, 23 (for ordered lists)
)

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

func (t Token) String() string {
	return fmt.Sprintf("Type:%d Literal:%q Line:%d", t.Type, t.Literal, t.Line)
}

// LookupIdent checks if an identifier is a keyword (if we had any reserved keywords)
// For now, it just returns IDENTIFIER
func LookupIdent(ident string) TokenType {
	return TOKEN_IDENTIFIER
}
