# Rite Markup Language to HTML Processor Specifications

This document specifies the requirements and design for a text processor program that converts documents written in the **Rite** markup language into standard HTML5. The processor will be implemented in the **Go** programming language and will follow a two-phase architecture: **Parsing** the Rite source text into an Abstract Syntax Tree (AST), and Generation of the target HTML from the AST.

Rite is heavily based on standard HTML, but uses significant indentation (lite Python), macros and some simplifications to make it easier to edit and read Rite documents. A Rite processor program can generate HTML which can then be processed with any other standard tools.

When humans read code or text, we rely heavily on visual cues to understand structure, and indentation is one of the most important visual cues for structure. A well-formatted HTML file allows a human reader to scan the document and instantly see the relationships between document elements (parents, children, and siblings) without reading every single HTML tag.

Any well-formed, properly formatted, and indented HTML file is considered valid Rite. Rite adds some simplification rules and macros to facilitate writing documents while making reading the source text easier than standard HTML.

We first describe the expected formatting and indentation rules for standard HTML, and later we will describe the specifics of Rite.

# 1. Rules for standard HTML

### Rule 1: Indentation unit

Choose one indentation unit and stick to it strictly throughout the entire document, also for the included files.
It is recommended to use **2 spaces or 4 spaces** per indentation level. Choose one and stick to it
Every nested level increases the indentation by exactly one unit. Use spaces, not tabs.

### Rule 2: Block-level Tag Alignment

Block-level elements and their corresponding closing tags must have the same indentation level.

* The opening tag starts the line. Unless specified in this document, a line must have only the opening tag without text following it.
* The closing tag starts its own line at the same indentation. In Rite, closing tags can be omitted, but this will be described below.
* Everything inside (the children) is indented exactly one level deeper (or more when there are nested children).

```html
<!-- Correct -->
<div>
  <p>Nested content goes here.</p>
</div>

<!-- Incorrect -->
<div>
    <p>Nested content.</p>
  </div>

```

## 1.2. Element-Specific Formatting

### Rule 3: Block vs. Inline Element Treatment

How you indent depends entirely on the display nature of the tag:

* **Block-Level Elements (`<div>`, `<article>`, `<form>`, `<header>`):** Must always start on a new line, and their closing tags must start on a new line.
* **Inline Elements (`<a>`, `<strong>`, `<span>`, `<em>`):** Should remain inline with the text they wrap, unless the line becomes too long to read comfortably.

```html
<!-- Correct Treatment of Block and Inline -->
<section>
  <p>This is a paragraph with an <a href="#">inline link</a> inside it.</p>
</section>

```

### Rule 4: Exception for Simple Content Blocks

For block-level elements that contain *only* a short string of text and no child elements, you may keep the entire element on a single line to save vertical space.

```html
<!-- Allowed for brevity -->
<h1>Main Page Title</h1>
<td>$15.00</td>

```

## 1.3. Managing Long Lines and Attributes

### Rule 5: Multi-Attribute Wrapping

When a single HTML tag has too many attributes (e.g., an `<img>` or an `<input>` tag with classes, IDs, sources, and accessibility tags), it becomes hard to read horizontally.

* **The Rule:** If a tag exceeds 80–100 characters, wrap each attribute onto its own line.
* **Indentation:** Indent the attributes one level deeper than the opening tag. Place the closing bracket `>` on its own line, aligned with the opening tag.

```html
<!-- Correct multi-attribute wrapping -->
<img 
  src="assets/images/hero-banner.jpg" 
  alt="A scenic view of the mountains during sunrise" 
  class="ui-image-responsive heavy-load-optimized"
  loading="lazy"
>

```

## 1.4. Document Structure & Comments

### Rule 6: Sibling Alignment

Elements that share the same parent and exist at the same structural level (siblings) must share the same indentation level.

```html
<ul>
  <li>Item 1</li>
  <li>Item 2</li>
  <li>Item 3</li>
</ul>

```

### Rule 7: Comment Alignment

Comments should be treated exactly like the HTML elements they document. They must inherit the exact indentation level of the code block they are describing or sitting inside.

```html
<main>
  <!-- Main content area container -->
  <div class="content-wrapper">
    <p>Content goes here.</p>
  </div>
</main>

```

# 2 Rite Markup Language Specifications

## 2.1 Lines and indentation

**Line:** A line in Rite is a line of text ended by the usual EOL (end-of-line character, e.g., CR or CR/LF depending on the platform).

**Indentation:** A line has zero or more space characters at the beginning before the actual text of the line. The number of spaces at the beginning of the line determines the indentation of the line. Only space characters are allowed for indentation; tabs are not allowed. However, to allow for flexibility, the actual Go processor may have an option to reformat the input file according to the Rite rules (something like the gofmt program), and accept tabs and convert them to spaces.

**Empty lines**: a line with only white space is an empty line, and indentation is 0. Contiguous empty lines are normally coalesced into a single empty line when parsing, with the exception of lines in verbatim blocks, described later in the document. From now on, when we use just the word “line”, we are referring to lines which are non-empty. When we have to refer to empty lines, we will explicitly say so.

**Special empty lines:** a line which contains only indentation and an HTML block end tag (like `</p>`) is considered an empty line, that is, the tag is ignored. This is so because Rite uses significant indentation to determine when blocks of text start and end. However, it is expected that for block HTML tags, if a start tag is ended with an end tag, both start and end tags have the same indentation. A line which has text following a block end tag is an error. For other types of tags, like end tags, having text following it is not an error.

**File indentation:** The first block of text (as defined below) in any file (including included files) **must start at indentation 0**. The first block of text in the file which has indentation (the block can not be the first in the file) defines the `indentation multiplier` of the file, or in other words, the `indentation of the file`. 

**Consistency:** All lines in the file must have an indentation which is a multiple of the file's indentation multipler.

## 2.2 Paragraphs

**Paragraph:** A paragraph is a set of non-empty lines which are contiguous and have the same indentation.

**Start/End:** A non-empty line after an empty line starts a paragraph. A non-empty line that has an indentation greater or lower than the previous line also starts a paragraph.

**Indentation of Paragraph:** The indentation of a paragraph is the indentation of the first line that starts the paragraph. Obviously, it is also the same as all the lines composing the paragraph, per the definition of paragraph.

## 2.3 Blocks

There are three types of blocks of text: `normal`, `list` and `verbatim`. We describe first the processing for normal blocks and later the exceptional rules for the other two types.

**Normal Block (or just Block):** Normal blocks form the basic hierarchical structure for content in Rite, and we follow a recursive definition style.

The first paragraph in a file starts the first block in the file. The first paragraph of a block is called the `start paragraph`. The first paragraph of the block is special, as it serves to indicate the type of block via block-level HTML tags. The type of block is determined by the characters at the beginning of the text of the start paragraph. For example, a block can start with the block-level HTML tags `<p>`, `<div>`, `<section>` and we would classify each block with that tag name. A block that does not start with any block-level tag (that is, starts with no tag or with inline tags) is automatically converted to a paragraph block. This is equivalent to a block that starts with the paragraph tag `<p>`.

After the start paragraph of a block, a paragraph with the same indentation as the block creates a sibling block of the current block. A paragraph with more indentation than the current block (we only allow one indentation level more, indenting two or more levels in a single step is an error) creates a child block. A paragraph with less indentation creates a block which is a sibling of the immediately previous block with the same indentation (de-denting by more than one level is allowed, contrary to indenting). Obviously, de-denting is impossible when the current block has indentation zero.

**Indentation of the block:** The indentation of a block is established by the indentation of its start paragraph.

**End of Block:** As mentioned above, a block ends with a paragraph with an indentation equal or lower than the start paragraph of the block, or the end of the file. Empty lines do not end a block. Paragraphs with more indentation than the block are the contents of the block, in the form of child blocks in a recursive way.

## 2.4 List Blocks
A list block is a special type of block used to facilitate to the user the definition of lists, similar to Markdown.

Before defining a list block, we need to define a `list item marker`, which is how each item in a list block is defined. They can be unordered or ordered.
An unordered list item marker can be either `- ` (a hyphen followed with a space) or `* ` (an asterisk followed by a space). An ordered list item marker is `1. ` (or in general, any single-digit followed by a dot and a space). A list item is a paragraph which starts with a list item marker.

A new list item starts a new implicit list block, except when the new list item has a previous sibling and it is of the same type (ordered, unordered). The implicit list block created is of the same type as the list item that created it.

For the sake of clarity, we describe the following examples:

- If a `<p>` block is followed by a paragraph with the same indentation starting with `- `, an implicit unordered list block is created (`<ul>`). If the paragraph starts with `1. ` then the implicit list block is ordered (`<ol>`).
- If a `<p>` block is followed by an unordered list item at the same indentation, and then we have another unordered list item at the same indentation, then the unordered list continues, including the two unordered list items. But if the last paragraph is an ordered list item, the previous implicit list block is closed an a new implicit ordered list block is created to include the last paragraph.

As a further facility to the writer, a paragraph starting with the tag `<li>` is treated like an unordered list item, starting an implicit undordered list block if none is created.

Of course, explicit list blocks can be created if needed by using the standard HTML tags for that purpose (`<ol>` or `<ul>`), as long as the formatting/indentation rules in Rite are followed.

List items are standard Rite blocks, so they can contain child blocks themselves, following the indentation rules in Rite.

## 2.5 Verbatim block
A verbatim block is a special block used to preserve formatting of lines and line breaks in the verbatim block. A verbatim block starts with a verbatim tag which in Rite are `<pre>` and `<x-code>`. The tag name `x-code` is a Rite macro which is equivalent to `pre``code` (more on this later). `script` and `style` are forbidden, as Rite is for text processing and styles and behaviour are specified with a different mechanism.

A verbatim block includes the start paragraph and all subsequent paragraphs which have an indentation greater than the block. Contrary to normal blocks, verbatim blocks do not have child blocks, and all paragraphs of greater indentation are considered to form part of the verbatim block, so we say that the paragraphs are included in verbatim form. As with normal blocks, a paragraph with equal or lower indentation than the current block closes that block.

## 2.6 Regarding the HTML tags

In Rite, we use 'normal' HTML tags and additional tags specific to Rite with a tag name that starts with `x-` to implement macros. Note: we do not implement Web Components in the standard way. The Rite macros exist only in the source text and are processed and replaced by standard HTML by the Rite processor. In run-time, the special Rite tags do not exist.

**Normal tag:** A normal HTML tag is a standard HTML start tag, like `<section>` or `<a src="index.html">`. Normal tags are parsed according to the rules of HTML, parsing its name, attributes and values to store in the AST. The Parser must support:

  * key="value"  
  * key='value'  
  * key=value (no quotes)  
  * key (boolean)

Those normal tags can be block-level tags or inline tags. Block-level tags can ONLY appear at the start of a block in Rite. They are invalid in the middle of a paragraph.

**Macro tag:** A macro tag is a specific Rite start tag with a name that begins with `x-`, for example, `<x-include src="file_to_include">`. At parsing time, the macro tags have to be parsed the same as a normal HTML tag, parsing its name, attributes and values to store in the AST. There are some exceptions which will be described later.

**Markdown inline markers:** A paragraph can also include markdown markers for bold, italic, underscore etc. The meaning in Rite is exactly the same as in markdown. For example:

  * **Strong/Bold:** e.g., `*text*`  
  * **Emphasis/Italic:** e.g., `_text_`  
  * **Inline Code:** e.g., ``code snippet``


## 2.7 Specific processing for tags

## The `section` block

Section blocks are the main structuring mechaninsm in Rite, marking individual chapters or distinct sections in the document (e.g., Methodology).

A section block starts with a paragraph which starts with a `<section>` tag. The paragraph is composed of the start tag and the rest of the paragraph (which we will creatively call `rest`). When generating HTML, a `<section>` tag will generate the following:

* A start tag `<section>` with all the same attributes defined in the source text. Eg. `<section class="classname">`.  
* A header tag (`h1`, `h2`, `h3`, etc.) where the level of the header tag depends on the indentation of the section tag with respect to other section tags. For example, the outermost section tag in the document generates the `<h1>` tag. All sections immediately inside the outermost section tag receive the h2 tag, and so on.  
* The contents of the header tag are the `rest` of the paragraph (as defined above), prefixed with the indicator of indentation of the section. For example, if `rest` is "Example section":  
  * If the section is the outermost section of the document, the content of the h1 header is “1. Example section”.  
  * If the section is the first immediately inside the outermost section of the document, the h2 header content will be “1.1 Example section”.  
  * The numbers are incremented corresponding to the indentation of the section and the order of the section relative to its siblings in its parent section.

For example, the following Rite text with the first section at the topmost level

```html
<section id="demo">Demo
  This is a normal paragraph block.
  
  This is a normal paragraph at the same level as the previous one.

  <section id="demo2">Demo 2
    This is another normal paragraph.

```

Generates the following HTML:

```html
<section id="demo">
  <h1>1. Demo</h1>
  <p>This is a normal paragraph block.</p>

  <p>This is a normal paragraph at the same level as the previous one.</p>

  <section id="demo2">
    <h2>1.1. Demo 2</h2>
      <p>This is another normal paragraph.</p>
</section>
```

## The `x-include` macro

The `x-include` macro can only appear alone in a line. It requires an `href` attribute, pointing to a file which must be included during parsing. The file must be parsed as if the including file contained the text being included. The indentation level of the included text must be the same as the indentation level of the `x-include` macro. In other words, the text at indentation level 0 in the included file must appear at the indentation level of the macro tag in the including file, and so on with the other indentation levels in the included text.

The `href` attribute can refer to a local file or to a remote one (if the file name is a url starting with `https`). Remote files must be cached just in case it is included again.

For example:

```html
`<x-include src="chapters/chapter_5.rite">`
```

Would include a file named `chapter_5.rite` in a subdirectory named `chapters`. The subdirectory is relative to the location of the inluding file, not relative to where the Rite processing program is run. For security reasons, the path to the included file can not start with a `/` or include `..`, to avoid exiting from the directory where the main Rite text file exists.

##  The `x-ref` macro

The `x-ref` macro is a reference to another section of the document. The tag has a string value which must correspond to the `id` attribute of some other tag in the document. For example: `<x-ref another_section>`. When the id has spaces, it has to be quoted, as in `"another section"`.

When generating HTML, the x-ref tag will be replaced by an anchor tag (`<a>`) where:

* Its `href` attribute is the `id` of the referenced tag.  
* If the referenced tag is a `section` tag, the content of the anchor tag is the content of the associated header tag, prefixed with “Section “ (notice the blank space to separate it from the indentation indicator of sections).  
* If the reference is to a table or figure, the contents of the anchor tag is the figcaption (if there is one) of the table or figure.

It is an error if the referenced tag does not exist in the document (taking into account the text included by the `x-include` tags).

### The `x-fig` macro

The `x-fig` macro is a shorthand for the combination of `figure`, `img` and `figcaption`. For example, the Rite source:

```html
<x-fig src="elephant.jpg">An elephant at sunset
```

Is translated to the following HTML:

```html
<figure>  
  <img  
    src="elephant.jpg"  
    alt="An elephant at sunset"
  />
  <figcaption>Fig 4. An elephant at sunset</figcaption>  
</figure>
```

* The `x-fig` tag uses the `rest` text to build the `figcaption`.  
* The `figcaption` text is transformed by appending the text “Fig n. “, where the ‘n’ must be the sequential numbering of `x-fig` tags in the source document, taking into account the included files.

### The `x-quote` macro

The `x-quote` macro is similar to `x-fig`, but for quotes instead of images:

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

### The `x-code` macro

The `x-code` macro is a verbatim tag, which is translated to a `<pre><code>` combination. As described above, the verbatim block is processed line-by-line (versus paragraph-by-paragraph which is the normal Rite processing mode) and includes all lines of text with an indentation higher than the indentation of the verbatim tag. The verbatim block can include empty lines, even the initial line. Empty lines must be preserved and sent to the output.

To facilitate the life of the writer (and the reader), the indentation of the text inside the verbatim block is adjusted to the left: the line inside the verbatim block with the minimum indentation will have indentation zero in the generated HTML. The other lines in the verbatim block will have indentations in the generated HTML according to their relative indentation in the original Rite text.

For example:

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

I am a new paragraph, which makes the verbatim block to close.
```

Will be converted to the HTML:

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

<p>I am a new paragraph, which makes the verbatim block to close.</p>
```

Notice how the text lines inside the verbatim block have been 'shifted' to the left, so the HTML displayed to the user will be properly formatted.

