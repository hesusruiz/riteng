package lexer

type LexerMode int

const (
	ModeStandard LexerMode = iota
	ModeRawLine
)

type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	
	// Tracking
	line   int
	column int

	// Indentation
	indentStack   []int // Stack of indentation levels. Always starts with [0]
	pendingDedents int   // Number of DEDENT tokens queued to be emitted
	atStartOfLine bool  // True if we are at the start of a line (processing indent)
	lastIndentLen int   // Length of the last processed indentation (spaces count)

	// Mode
	mode LexerMode
}

func New(input string) *Lexer {
	l := &Lexer{
		input:         input,
		line:          1,
		column:        0,
		indentStack:   []int{0},
		atStartOfLine: true,
		mode:          ModeStandard,
	}
	l.readChar()
	return l
}

func (l *Lexer) SetMode(mode LexerMode) {
	l.mode = mode
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.column++
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) NextToken() Token {
	// 1. Handle Pending Dedents (priority)
	if l.pendingDedents > 0 {
		l.pendingDedents--
		return l.newToken(TOKEN_DEDENT, "")
	}

	// 2. Handle specific modes
	if l.mode == ModeRawLine {
		if l.ch == '\n' {
			tok := l.newToken(TOKEN_NEWLINE, "\n")
			l.readChar()
			l.line++
			l.column = 0
			// In Raw Mode, we might want to track indentation for the caller?
			// But for now, just return newline. The caller (Parser) controls when to switch back.
			l.atStartOfLine = true // Still track this?
			return tok
		}
		return l.readRawLine()
	}

	// 3. Handle Indentation (if at start of line)
	if l.atStartOfLine {
		return l.processIndentation()
	}

	// 4. Standard Tokenization
	l.skipWhitespace() // Skips spaces/tabs, stops at newline or char

	switch l.ch {
	case 0:
		// EOF: Close all open blocks with DEDENTs
		if len(l.indentStack) > 1 {
			l.pendingDedents = len(l.indentStack) - 1
			l.indentStack = []int{0} // Reset
			return l.NextToken()
		}
		return l.newToken(TOKEN_EOF, "")
	case '\n':
		l.atStartOfLine = true
		tok := l.newToken(TOKEN_NEWLINE, "\n")
		l.readChar()
		l.line++
		l.column = 0
		return tok
	case '<':
		tok := l.newToken(TOKEN_TAG_START, "<")
		l.readChar()
		return tok
	case '>':
		tok := l.newToken(TOKEN_TAG_END, ">")
		l.readChar()
		return tok
	case '/':
		tok := l.newToken(TOKEN_SLASH, "/")
		l.readChar()
		return tok
	case '=':
		tok := l.newToken(TOKEN_EQUALS, "=")
		l.readChar()
		return tok
	case '*':
		tok := l.newToken(TOKEN_STAR, "*")
		l.readChar()
		return tok
	case '-':
		tok := l.newToken(TOKEN_DASH, "-")
		l.readChar()
		return tok
	case '_':
		tok := l.newToken(TOKEN_UNDERSCORE, "_")
		l.readChar()
		return tok
	case '`':
		tok := l.newToken(TOKEN_BACKTICK, "`")
		l.readChar()
		return tok
	case '.':
		tok := l.newToken(TOKEN_DOT, ".")
		l.readChar()
		return tok
	case '"', '\'':
		return l.readString()
	default:
		if isLetter(l.ch) {
			return l.readIdentifier()
		} else if isDigit(l.ch) {
			return l.readNumber()
		} else {
			// Just return it as text/error? For now, let's assume it's part of text or illegal
			// But for Rite/HTML, many chars are valid text.
			// Ideally, we have a generic 'TEXT' token for content, but we are tokenizing strictly here.
			// Let's return Illegal or treat as Ident for now? 
			// BETTER: Treat unknown chars as part of an Identifier/Text if we are not inside a tag?
			// For this MVP, let's treat it as Illegal if not captured, implying we need to expand supported chars.
			tok := l.newToken(TOKEN_ERROR, string(l.ch))
			l.readChar()
			return tok
		}
	}
}

// processIndentation calculates indentation level and manages the stack
func (l *Lexer) processIndentation() Token {
	indentLen := 0
	
	// Peeking ahead to count spaces without consuming if strictly needed, 
	// but here we are at 'atStartOfLine', so next chars are spaces or content.
	// Note: 'l.ch' is currently the first char of the line (or space)
	
	// Special Case: Empty lines (just newline) -> Emit newline, ignore indentation change
	// Or we can peek to see if it's just spaces then newline.
	// But specification says: "A line with only white space is an empty line... coalesced"
	// Let's consume spaces first.
	
	// We need to preserve l.readChar() logic.
	// Backup position to count? simpler: consume spaces.
	
	// startPos := l.position // Unused
	for l.ch == ' ' {
		l.readChar()
		indentLen++
	}
	l.lastIndentLen = indentLen
	
	if l.ch == '\n' {
		// Empty line (spaces then newline)
		// Remain atStartOfLine = true for next line?
		// No, processIndentation returns a token. 
		// If we return NEWLINE here, the caller loop will handle it.
		// The caller will see \n, set atStartOfLine=true again.
		// But wait, we consumed the newline? No, check loop condition.
		// If l.ch == \n, we return NEWLINE?
		// Actually, spec says "Empty lines do not end a block".
		// We should probably Ignore this line's indentation effects, consume the newline, and continue.
		// Recurse?
		l.readChar() // Consume \n
		l.line++
		l.column = 0
		l.atStartOfLine = true
		return l.newToken(TOKEN_NEWLINE, "\n")
	}

	// We hit non-space content.
	l.atStartOfLine = false // Handled.

	currentIndent := indentLen
	topIndent := l.indentStack[len(l.indentStack)-1]

	if currentIndent > topIndent {
		l.indentStack = append(l.indentStack, currentIndent)
		return l.newToken(TOKEN_INDENT, "")
	} else if currentIndent < topIndent {
		// Detect how many levels to pop
		dedents := 0
		found := false
		for i := len(l.indentStack) - 1; i >= 0; i-- {
			if l.indentStack[i] == currentIndent {
				found = true
				break
			}
			dedents++
		}
		
		if !found {
			return l.newToken(TOKEN_ERROR, "Inconsistent indentation")
		}

		// Pop the stack
		l.indentStack = l.indentStack[:len(l.indentStack)-dedents]
		
		// Return first DEDENT, queue the rest
		if dedents > 0 {
			l.pendingDedents = dedents - 1
			return l.newToken(TOKEN_DEDENT, "")
		}
	}

	// currentIndent == topIndent -> No token needed, proceed to consume content (which will be handled by NextToken loop recursion/fallthrough?)
	// NextToken calls processIndentation. If we return nothing? 
	// Make processIndentation recursive call NextToken?
	return l.NextToken()
}

func (l *Lexer) readRawLine() Token {
	// Consumes full line until newline
	pos := l.position
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	// Note: We do NOT consume the newline here?
	// If we don't, next NextToken call sees newline.
	// But we want the RAW_LINE to include everything?
	// Usually Raw Line implies content. Newline is separate?
	// Let's include everything up to newline. Newline separate.
	lit := l.input[pos:l.position]
	return l.newToken(TOKEN_RAW_LINE, lit)
}

func (l *Lexer) readString() Token {
	quote := l.ch
	l.readChar()
	pos := l.position
	for l.ch != quote && l.ch != 0 {
		l.readChar()
	}
	lit := l.input[pos:l.position]
	if l.ch == quote {
		l.readChar()
	}
	return l.newToken(TOKEN_STRING, lit)
}

func (l *Lexer) readIdentifier() Token {
	pos := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '-' || l.ch == '_' {
		l.readChar()
	}
	lit := l.input[pos:l.position]
	return l.newToken(LookupIdent(lit), lit)
}

func (l *Lexer) readNumber() Token {
	pos := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.newToken(TOKEN_NUMBER, l.input[pos:l.position])
}

func (l *Lexer) skipWhitespace() {
	// Skip ' ' and '\t' (though tabs forbidden)
	// Do NOT skip '\n' as it is significant
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) newToken(tokenType TokenType, literal string) Token {
	return Token{
		Type:    tokenType,
		Literal: literal,
		Line:    l.line,
		Column:  l.column,
	}
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// Rewind moves the lexer back by n characters.
// This is used by the Parser when it over-consumes a Raw Line that turns out to be a Dedent.
func (l *Lexer) Rewind(n int) {
	l.readPosition -= n
	l.position -= n
	// We might need to adjust line/column, but for now we assume we just rewind to start of line
	// and NextToken will set atStartOfLine logic again?
	// If we rewind a full line (including \n?), we go back to previous line.
	// But readRawLine consumes until \n.
	// If we rewind the content, l.ch needs to be reset.
	l.ch = l.input[l.position]
	l.atStartOfLine = true // Crucial: We rewound to start of line
	
	// If we rewind the content, l.ch needs to be reset.
	l.ch = l.input[l.position]
	l.atStartOfLine = false // We assume we are NOT at start of line unless we rewind full line?
	// Actually typical Rewind use is for content. NextToken will handle newline if we hit it.
}

func (l *Lexer) RewindToLastIndent() {
	// Rewinds the indentation spaces consumed by the last processIndentation call
	l.Rewind(l.lastIndentLen)
	l.atStartOfLine = true // We are definitely back at start logic
}

func (l *Lexer) CurrentIndent() int {
	if len(l.indentStack) == 0 {
		return 0
	}
	return l.indentStack[len(l.indentStack)-1]
}
