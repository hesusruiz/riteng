# Rite Markup Language to HTML Processor Specifications

# 1 Introduction

This document specifies the requirements and design for a text processor program that converts documents written in the **Rite** markup language into standard HTML5. The processor will be implemented in the **Go** programming language and will follow a two-phase architecture: **Parsing** the Rite source text into an Abstract Syntax Tree (AST), and Generation of the target HTML from the AST.

## 1.1 Goal

The primary goal of this specification is to serve as the guide for the development of the Rite-to-HTML processor, particularly acting as a detailed reference for both human developers and AI code generation tools (e.g., Gemini Code Assist).

## 1.2 Program Architecture

The processor (`rite-html-go`) will consist of two distinct and sequential phases:

1. **Parser (`pkg/parser`):** Reads the Rite source text and constructs an in-memory Abstract Syntax Tree (AST). The parser will be implemented with a recursive descent parser.

2. **Generator (`pkg/generator`):** Traverses the AST and outputs the corresponding HTML5 code.

# 2 Rite Markup Language Specifications (Indentation-Based)

Any well-formed, properly formatted, and indented HTML file is considered valid Rite. In this context, the words “properly formatted” means that start and end tags for block-level tags have to be in different lines and with the same indentation, and that the content inside such start and end tags must be indented with respect to the tags. 

 The following conditions govern the Rite syntax, where white space at the beginning of each line is significant, similar to Python.

## 2.1 Indentation and Lines

* **Line Definition:** A line in Rite is a text line ended by the usual EOL (end-of-line character, e.g., CR or CR/LF).

* **Indentation:** A line has 0 or more space characters at the beginning before the actual text of the line. The number of spaces at the beginning of the line determines the indentation of the line. Only space characters are allowed for indentation; tabs are not allowed.

* **Empty lines**: a line with only white space is an empty line, and indentation is 0\. Contiguous empty lines are normally coalesced into a single empty line when parsing, with the exception of lines in verbatim blocks, described later in the document. From now on, when we use just the word “line”, we are referring to lines which are non-empty. When we have to refer to empty lines, we will explicitly say so.

* **Special empty lines:** a line which contains only indentation and an HTML end tag (like `</p>`) is considered an empty line, that is, the tag is ignored.

* **File indentation:** The first block in any file (including included files) **must start at indentation 0**. This modification to the original spec simplifies parsing and inclusion logic.

* **Consistency:** All lines in the file must have an indentation which is a multiple of the file's determined indentation.

## 2.2 Paragraphs

* **Definition:** A paragraph is a set of non-empty lines which are contiguous and have the same indentation.

* **Start/End:** A non-empty line after an empty line starts a paragraph. A non-empty line that has an indentation greater or lower than the previous line also starts a paragraph..

* **Indentation of Paragraph:** The indentation of a paragraph is the indentation of the first line that starts the paragraph. Obviously, it is also the same as all the lines composing the paragraph, per the definition of paragraph.

## 2.3 Blocks

* **Definition:** A block starts when a paragraph starts with a block tag (defined below). The block includes the start paragraph and all subsequent paragraphs which have an indentation greater than the first paragraph in the block.

* The first paragraph of the block is special, as it serves to indicate the type of block via block tags.

* **Indentation of the block:** The indentation of a block is established by the first child line that has an indentation greater than the block's start line. If a block has no children (empty), it is treated as a leaf node and indentation depth is irrelevant.

* **End of Block:** A block ends with a paragraph with an indentation equal or lower than the start paragraph of the block. Empty lines do not end a block.

* **Child blocks:** A block can contain child blocks. A child block starts with a block tag, like any block.

* **List blocks:** List blocks are a special type of block which has different rules for when they start and end. A list block starts when a paragraph starts with a list marker, and the immediately preceding paragraph was not a list item of the same type/indentation. The block can be of two types: ordered and unordered. List markers are defined in a section below. The list block includes all paragraphs with the same indentation and that start with the same type of list marker, and all child blocks. List blocks can be of two types (ordered and unordered), depending on the type of list marker which the first paragraph of the block has.

* **Verbatim blocks:** Verbatim blocks are a special type of block which has different rules of processing. A verbatim block starts with a paragraph which starts with a verbatim marker (defined in a section below) and includes the start paragraph and all subsequent paragraphs which have an indentation  greater than the first paragraph in the block. Contrary to normal blocks, verbatim blocks do not have child blocks, and all paragraphs of equal or greater indentation are considered to form part of the verbatim block, so we say that the paragraphs are included in verbatim form. 

## 2.4 Normal Tags and Macros

* **Normal tag:** A normal tag is a standard HTML start tag, like `<section>` or `<a src="index.html">`. Normal tags are parsed according to the rules of HTML, parsing its name, attributes and values to store in the AST. The Parser must support:

  * key="value"  
  * key='value'  
  * key=value (no quotes)  
  * key (boolean)

* **Macro tag:** A macro tag is a start HTML tag with a name that begins with x-, for example, `<x-include src="file_to_include">`. At parsing time, the macro tags have to be parsed the same as a normal HTML tag, parsing its name, attributes and values to store in the AST. There are some exceptions which will be described later.

* **Block tags and inline tags:** Rite paragraphs can include any normal tag and a set of macro tags which are defined in a section of this document. Both normal and macro tags can be of two types: block tags or inline tags. The meaning of block versus inline is essentially the same as in standard HTML, with some special processing rules defined in this document. Examples of block tags are `<section>`, `<p>` and  `<div>`. Examples of inline tags are `<b>` and `<a>`.

* **Markdown inline markers:** A paragraph can also include markdown markers for bold, italic, underscore etc. The meaning in Rite is exactly the same as in markdown. For example:

  * **Strong/Bold:** e.g., `*text*`  
  * **Emphasis/Italic:** e.g., `_text_`  
  * **Inline Code:** e.g., ``code snippet``

## 2.5 Block Starters and Implicit Tag Mapping

Blocks are marked with a type, depending on how the block starts.

If the block starts with a block tag, the block is marked as being of the same type as the tag. The tag is parsed using the usual rules of HTML tags and is included as part of the block. That is, the attributes and values in the tag are processed and included in the block node for the AST.

If the block starts with an inline tag or with no tag, the block is marked as if it starts with `p` (assumes an implicit `p` block tag).

There are some tags which are special, which are associated with list markers and verbatim markers.

### List markers

If a block starts with a list marker, the block is marked as a list block, with its subtype (ordered or unordered). All paragraphs of the block with the same indentation and list marker than the first paragraph of the block are considered elements of the list. The list markers are the following:

* `li`

* A hyphen (`-`) followed by white space and the rest of the text.

* An asterisk (`*`) followed by white space and the rest of the text.

* An integer number (e.g., 1 or 15) followed by a dot (.), white space, and the rest of the text.

Lists can be ordered or unordered. Ordered lists use the integer number marker, unordered lists use `li`, hyphen or asterisk markers.

### Verbatim markers

Verbatim markers are block tags used to preserve formatting of lines and line breaks in the verbatim block. The verbatim markers in Rite are `pre` and `x-code`. `x-code` is a macro which is equivalent to `pre``code`. `script` and `style` are forbidden, as Rite is for text processing and styles and behaviour are specified with a different mechanism, not in the scope of this document.

## 2.7 Inline Markup and Nesting

Each of the paragraphs of a block undergo a separate, recursive inline parsing phase. Inline elements are non-block-starting content.

* **HTML Inline Tags:** Any standard HTML inline tag (e.g., `<b>`, `<i>`, `<a>`, `<img>`) is supported.

* **Inline Macro Tags:** Macro tags (starting with x-, e.g., `<x-tooltip>`) are fully supported in an inline context, following the same parsing and nesting rules as standard HTML inline tags.

* **Markdown Inline Markers:** The standard Markdown markers for formatting are supported:

  * Strong/Bold (e.g., `*text*` or `**text**`)

  * Emphasis/Italic (e.g., `_text_` or `*text*`)

  * Inline Code (e.g., `code snippet`)
º
* **Nesting:** Inline elements, including Normal and Macro tags, may be nested within each other (e.g., bold text within a link).

# Tags and processing specific to Rite

## Section block

A section block starts with a paragraph which starts with a ‘`section`’ tag. The paragraph is composed of the start tag and the ‘rest’ of the paragraph. When generating HTML, a section tag will generate the following:

* A start tag `<section>` with all the same attributes defined in the source text. Eg. `<section class="classname">`.  
* A header hag (`h1`, `h2`, `h3`, …) where the level of the header tag depends on the indentation of the section tag with respect to other section tags. For example, the outermost section tag in the document generates the h1 tag. All sections immediately inside the outermost section tag receive the h2 tag, and so on.  
* The contents of the header tag are the ‘rest’ of the paragraph, prefixed with the indicator of indentation of the section. For example, if ‘rest’ is “Example section”:  
  * If the section is the outermost section of the document, the content of the h1 header is “1. Example section”.  
  * If the section is the first immediately inside the outermost section of the document, the h2 header content will be “1.1 Example section”.  
  * The numbers are incremented corresponding to the indentation of the section and the order of the section relative to its siblings in its parent section.

The Parser needs a State Stack to track counters.

## Macro tags

This is the list of macro tags. They are all prefixed by ‘x-’ to differentiate from the normal HTML tags.

### The x-include tag

The ‘x-include’ tag requires a ‘href’ attribute, pointing to a file which must be included during parsing. The file must be parsed as if the including file contained the text being included.

The ‘href’ attribute can refer to a local file or to a remote one (if the file name is a url starting with ‘https’). Remote files must be cached just in case it is included again.

###  The x-ref tag

The x-ref tag is a reference to another section of the document. The tag has a string value which must correspond to the ‘id’ attribute of some other tag in the document. For example: `<x-ref another_section>`. When the id has spaces, it has to be quoted.

The x-ref tag will be resolved in the HTML generation phase, but the parser must generate a dictionary with all the ids found in the document and included documents, so x-ref tags will be able to be processed later.

When generating HTML, the x-ref tag will generate an anchor tag (`<a>`):

* Its ‘href’ attribute is the ‘id’ of the referenced tag.  
* If the referenced tag is a section tag, the content of the anchor tag is the content of the associated header tag, prefixed with “Section “ (notice the blank space to separate it from the indentation indicator of sections).  
* If the reference is to a table or figure, the contents of the anchor tag is the figcaption (if there is one) of the table or figure.

### The x-fig tag

The x-fig tag is a shorthand for the combination of figure, img and figcaption. For example, the Rite source:

```html
<x-fig src="elephant.jpg">An elephant at sunset
```

Is translated to the following HTML:

```html
<figure>  
  <img  
    src="elephant.jpg"  
    alt="An elephant at sunset"/>
  <figcaption>Fig 4. An elephant at sunset</figcaption>  
</figure>
```

* The x-fig tag uses the ‘rest’ text to build the figcaption.  
* The fig caption text is transformed by appending the text “Fig n. “, where the ‘n’ must be the sequential numbering of x-fig tags in the source document. This means that the parsing phase must keep a count of the x-figs found until that moment, and set the number in the AST node corresponding to the x-fig element.

### The x-quote tag

The x-quote tag is similar to x-fig, but for quotes instead of images:

```html
<x-quote>Edsger Dijkstra  
  This is a quote
```

Is translated to the following:

```html
<figure>  
  <figcaption><b>Edsger Dijkstra</b></figcaption>  
  <blockquote>  
    This is a quote.  
  </blockquote>  
 </figure>
```

### The x-code tag

The x-code tag is a verbatim tag, which is translated to a `<pre><code>` combination. When entering an x-code block, the lexer enters a "Raw Text Mode" where it consumes all input as raw text until the indentation level drops, bypassing standard tokenization.

```html
<x-code id="cow-caption">A cow saying, "I'm an expert in my field".  
  ___________________________  
  &lt; I'm an expert in my field. &gt;  
  ---------------------------  
        \   ^__^  
         \  (oo)\______  
            (__)\       )\/\  
                ||----w |  
                ||     ||
```

```html
<figure>  
  <pre role="img" aria-label="ASCII COW"><code>  
___________________________  
&lt; I'm an expert in my field. &gt;  
---------------------------  
      \   ^__^  
        \  (oo)\______  
          (__)\       )\/\  
              ||----w |  
              ||     ||
</code></pre>  
  <figcaption id="cow-caption">  
    A cow saying, "I'm an expert in my field".  
  </figcaption>  
</figure>
```

The first line after the x-code tag must be indented to be part of the block defined by the x-code tag, as per the Rite rules. But the x-code tag is a verbatim tag which maintains the formatting of the text in the block.

That means that the processing of the interior block of an x-tag is special: it must be done line by line instead of paragraph by paragraph, which is the normal Rite processing mode. In addition, empty lines must be preserved and sent to the output.

To facilitate the life of the writers, the parser must generate a `<pre><code>` combination where all lines of the interior block of the x-code tag are shifted-left in the same amount of spaces as the indentation of the first non-blank line of the interior block. To make the Lexer simpler, this processing (shifting left) must be done in the parser (where the indentation of the x-code block is already known) or in the generation phase, where semantics are already known.

# Go Implementation Plan


The architecture is now confirmed as:

1. **Lexer:** Context-aware (Standard vs. Raw Text), generating indentation tokens.
2. **Parser:** Recursive descent, building a rich AST.
3. **Generator:** Traversal-based, handling state (numbering) and resolution (cross-references).

Below is the **Go Implementation Plan** designed to meet these specific requirements.

### **Phase 1: The Lexer (`pkg/lexer`)**

The Lexer is the foundation. It must handle the significant whitespace and the "Raw Text Mode" required for `x-code`.

**1.1 Token Definitions**
We will define tokens to abstract the raw text into meaningful chunks for the parser.

```go
type TokenType int

const (
    // Structural
    TOKEN_EOF TokenType = iota
    TOKEN_ERROR
    TOKEN_NEWLINE

    // Indentation (Crucial for Rite)
    TOKEN_INDENT  // Generated when indentation increases
    TOKEN_DEDENT  // Generated when indentation decreases

    // Content
    TOKEN_TEXT    // Standard text content
    [cite_start]TOKEN_RAW_TEXT // For inside x-code blocks [cite: 156]

    // Block Markers
    TOKEN_TAG_START // <
    TOKEN_TAG_END   // >
    TOKEN_SLASH     // / (for closing tags)
    TOKEN_EQUALS    // =
    TOKEN_STRING    // "value" or 'value'
    TOKEN_IDENTIFIER // tag names, attribute keys

    // Markdown/Rite Markers
    TOKEN_HASH      // # (reserved, though sections use tags)
    TOKEN_STAR      // * (bold or list)
    TOKEN_UNDERSCORE // _ (italic)
    TOKEN_BACKTICK  // ` (inline code)
    TOKEN_DASH      // - (list)
    TOKEN_DOT       // . (ordered list)
    TOKEN_NUMBER    // 1 (ordered list)
)

```

**1.2 The Lexer Struct & State Machine**
The Lexer needs a stack to track indentation levels to emit the correct number of `DEDENT` tokens when the indentation drops multiple levels at once.

```go
type Lexer struct {
    input       string
    position    int
    readPosition int
    ch          byte
    
    // Indentation State
    indentStack []int // Starts with [0]

    // Context State
    [cite_start]isInRawMode bool // For x-code [cite: 156]
}

// NextToken switches logic based on state
func (l *Lexer) NextToken() Token {
    // 1. Handle Raw Mode (x-code) specifically
    if l.isInRawMode {
        return l.readRawToken()
    }
    
    // 2. Handle Indentation at start of line
    if l.atStartOfLine {
        return l.readIndentation() 
    }

    // 3. Standard Tokenization (<, >, *, text, etc.)
}

```

**1.3 Handling `x-code` Raw Mode**
As specified in , when the Parser detects `<x-code>`, it signals the Lexer to enter `RawMode`.

* **Logic:** The Lexer consumes lines strictly checking indentation.
* 
**Shift-Left:** As per 

, the parser (or lexer helper) calculates the indent of the first non-empty line and strips that prefix from subsequent lines.

---

### **Phase 2: The Parser (`pkg/parser`)**

The Parser transforms tokens into an Abstract Syntax Tree (AST). It uses "Recursive Descent" .

**2.1 AST Nodes**
We need specific nodes to carry the metadata described in the spec (IDs, numbering, references).

```go
// Base Interface
type Node interface {
    TokenLiteral() string
}

// Block Nodes
type Document struct {
    Children []Node
    Meta     map[string]string // File-level metadata
}

[cite_start]type SectionNode struct { // [cite: 100]
    Level      int    // Calculated depth (h1, h2...)
    Identifier string // Extracted or generated ID
    Title      []Node // Inline nodes (bold, text)
    Children   []Node // Sub-sections and content
}

type BlockNode struct { // Generic paragraphs, divs
    Tag      string // "p", "div", "blockquote"
    Children []Node
}

[cite_start]type VerbatimNode struct { // [cite: 87]
    Type    string // "pre", "x-code"
    Content string // The raw text
    Alt     string // For x-code caption/aria
}

[cite_start]type FigureNode struct { // [cite: 130]
    Src      string
    Caption  []Node
    [cite_start]Number   int // For "Fig n." generation [cite: 141]
}

// Inline Nodes
type TextNode struct { Value string }
type InlineTagNode struct { ... } // <b>, <span>
[cite_start]type RefNode struct { TargetID string } // <x-ref> [cite: 121]

```

**2.2 Parsing Logic & State**
The Parser needs to maintain the "Dictionary" of IDs mentioned in  and track `x-fig` counts .

```go
type Parser struct {
    l *lexer.Lexer
    
    // State Tracking
    [cite_start]figCount int // Tracks x-figs found [cite: 142]
    [cite_start]idRegistry map[string]bool // For x-ref validation [cite: 125]
    
    // Errors
    errors []string
}

// Main Loop
func (p *Parser) ParseDocument() *ast.Document {
    // Loop until EOF
    // Detect Block Type (List, Section, Verbatim, Paragraph)
    // Recursively parse children
}

```

* 
**List Detection:** Check for `TOKEN_DASH`, `TOKEN_STAR`, or `TOKEN_NUMBER` + `TOKEN_DOT` at the start of a block 

.
* 
**Implicit P:** If no block tag is found, create a `BlockNode{Tag: "p"}` 

.

---

### **Phase 3: The Generator (`pkg/generator`)**

The Generator walks the AST and outputs HTML5. It handles the dynamic numbering rules.

**3.1 The Generator Struct**
It needs a "State Stack" for section numbering .

```go
type Generator struct {
    // Section Numbering Stack
    // e.g. [1, 2] means we are inside Section 1, Subsection 2
    sectionCounters []int 
}

func (g *Generator) VisitSection(node *ast.SectionNode) {
    // 1. Increment current level counter
    g.incrementCounter(node.Level)
    
    // 2. Generate Number String (e.g., "1.2")
    numberStr := g.getCurrentNumberString()
    
    // 3. Output HTML
    // <section class="...">
    //   <h2 id="...">1.2 Title...</h2>
    //   ... children ...
    // </section>
}

```

**3.2 Reference Resolution (`x-ref`)**

* **Runtime Resolution:** When visiting an `x-ref` node, the generator looks up the target ID.
* **Formatting:** If the target is a Section, it prefixes "Section ". If a Figure, it uses the caption text .



---

### **Phase 4: Helper Utilities**

1. **Loader (`pkg/loader`)**: Handles `x-include` logic.
* Must support Local files and HTTP/HTTPS URLs .


* 
**Caching:** Must cache remote files 

.
* **Safety:** Implement circular dependency checks.



### **Next Steps for Development**

1. **Initialize Module:** `go mod init rite-processor`
2. **Build Lexer First:** Implement the indentation stack and `x-code` raw mode switching. Write unit tests for edge cases (dedenting 3 levels at once).
3. **Build AST:** Define the structs.
4. **Connect Parser:** Implement the recursive parsing for Blocks first, then Inline elements.

Would you like me to generate the **code for the Lexer struct and its core loop** to get you started?

# 3 Go Implementation Details (Maintenance Guide)

This section documents the specific architectural decisions and logic used in the Go implementation (`riteng`).

## 3.1 Lexer (`pkg/lexer`)
*   **Indentation Management**: Uses an `indentStack` (slice of ints) to track levels. Emits `TOKEN_INDENT` (push) and `TOKEN_DEDENT` (pop).
*   **Raw Line Mode**: Critical for `x-code` blocks.
    *   Triggered by `SetMode(ModeRawLine)`.
    *   In this mode, the Lexer emits `TOKEN_RAW_LINE` containing the *entire* line content (including leading whitespace).
    *   **Rewind Capability**: Implements `Rewind(n)` and `RewindToLastIndent()` to allow the Parser to "peek" at lines and push them back if they terminate a block.
    *   **Clipping Fix**: The initial line of content in `x-code` requires careful rewinding to avoid clipping the first word. The implementation rewinds based on `len(peekToken.Literal)` *before* the `INDENT` token is fully consumed in the state machine.

## 3.2 Parser (`pkg/parser`)
*   **Recursive Descent**: `ParseDocument` -> `parseBlock` -> `parseChildren`.
*   **Infinite Loop Protection**: `parseBlock` logic includes safety checks to force token stream advancement if no match is found, preventing infinite loops on invalid input or EOF.
*   **`x-code` Handling**:
    1.  Parser detects `x-code` tag.
    2.  Sets Lexer to `ModeRawLine`.
    3.  Consumes `TOKEN_RAW_LINE`s.
    4.  Checks indentation of each raw line against the `x-code` block's base indent.
    5.  If a line is less indented, it `Rewinds` that line and switches Lexer back to `ModeStandard`.
*   **`x-include`**: Implemented via AST Splicing. The parser recursively invokes `ParseDocument` on the included file text and appends the resulting children to the current AST.
*   **Ghost Block**: (Known Issue) The parser currently emits an empty `GenericBlock` ("p") at EOF due to `consumeIgnoredTag` logic. The Generator handles this gracefully (renders empty block or harmless p).

## 3.3 Generator (`pkg/generator`)
*   **Visitor Pattern**: A simple `Generate(doc)` function walks the AST.
*   **Output**: Writes to `strings.Builder`.
*   **Entity Escaping**: Basic HTML escaping implemented for text nodes and verbatim content.

## 3.4 CLI (`cmd/rite`)
*   Standard Unix-style tool.
*   Usage: `rite -i input.rite -o output.html`
*   Defaults to `stdin`/`stdout` if flags are omitted.