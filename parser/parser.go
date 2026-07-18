package parser

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/hesusruiz/riteng/ast"
	"gopkg.in/yaml.v3"
)

// Parse parses a Rite document from the given reader.
func Parse(r io.Reader) (*ast.Document, error) {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	doc := &ast.Document{
		Metadata: make(map[string]interface{}),
	}

	// 1. Parse YAML Metadata at the absolute top of the file
	startIdx := 0
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "---") {
		yamlLines := []string{}
		endIdx := -1
		for i := 1; i < len(lines); i++ {
			if strings.HasPrefix(strings.TrimSpace(lines[i]), "---") {
				endIdx = i
				break
			}
			yamlLines = append(yamlLines, lines[i])
		}
		if endIdx != -1 {
			yamlStr := strings.Join(yamlLines, "\n")
			err := yaml.Unmarshal([]byte(yamlStr), &doc.Metadata)
			if err != nil {
				return nil, fmt.Errorf("error parsing metadata: %w", err)
			}
			startIdx = endIdx + 1
		}
	}

	remainingLines := lines[startIdx:]

	// 2. Determine indentation multiplier and validate
	multiplier, err := detectMultiplier(remainingLines)
	if err != nil {
		return nil, err
	}

	// 3. Parse blocks
	blocks, err := parseBlocks(remainingLines, multiplier, 0)
	if err != nil {
		return nil, err
	}
	doc.Blocks = blocks

	return doc, nil
}

// detectMultiplier finds the indentation multiplier (first non-zero indentation)
func detectMultiplier(lines []string) (int, error) {
	firstBlockFound := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		indent := countIndent(line)
		if !firstBlockFound {
			if indent != 0 {
				return 0, fmt.Errorf("first block of text must start at indentation 0")
			}
			firstBlockFound = true
		} else {
			if indent > 0 {
				return indent, nil
			}
		}
	}
	// Default to 2 if no indented blocks found
	return 2, nil
}

// countIndent counts the number of spaces at the beginning of a line.
func countIndent(line string) int {
	count := 0
	for _, r := range line {
		if r == ' ' {
			count++
		} else if r == '\t' {
			// Tab is not allowed, but count it as 4 spaces or error. Let's return 0 and let validation handle it, or treat tab as error.
			// The spec says: "Only space characters are allowed for indentation; tabs are not allowed."
			return -1
		} else {
			break
		}
	}
	return count
}

type lineInfo struct {
	text   string
	indent int
	index  int
}

// parseBlocks parses a list of blocks recursively
func parseBlocks(lines []string, multiplier int, baseIndent int) ([]ast.Node, error) {
	// Filter and convert to lineInfo structs, identifying empty lines
	var infos []lineInfo
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := countIndent(line)
		if indent < 0 {
			return nil, fmt.Errorf("tabs are not allowed for indentation (line %d)", i+1)
		}
		if trimmed == "" {
			infos = append(infos, lineInfo{text: "", indent: 0, index: i})
		} else {
			if indent%multiplier != 0 {
				return nil, fmt.Errorf("indentation %d is not a multiple of multiplier %d (line %d)", indent, multiplier, i+1)
			}
			infos = append(infos, lineInfo{text: line, indent: indent, index: i})
		}
	}

	var nodes []ast.Node
	i := 0
	n := len(infos)

	for i < n {
		// Skip empty lines at the sibling level
		if infos[i].text == "" {
			i++
			continue
		}

		info := infos[i]
		// Dedent check: if indent is less than baseIndent, stop parsing here and let the caller handle it
		if info.indent < baseIndent {
			break
		}
		// If indent is greater than baseIndent by more than one level:
		if info.indent > baseIndent {
			return nil, fmt.Errorf("indentation increased by more than one level (line %d)", info.index+1)
		}

		// We have a block at baseIndent. Let's parse its paragraph/verbatim content.
		// A block starts with the current line.
		// First, check if it is a verbatim block or standard block.
		firstLineText := strings.TrimSpace(info.text)
		tagName, attrs, rest, isBlockTag, isVerbatim, ok := parseStartTag(firstLineText)

		if ok && isVerbatim {
			// Verbatim block: consumes all lines (including empty lines) that have indentation > baseIndent
			verbNode := &ast.BlockNode{
				TagName:    tagName,
				Attributes: attrs,
				Indent:     baseIndent,
				IsVerbatim: true,
				Rest:       rest,
			}
			i++ // consume the start line
			var rawLines []string
			// Continue consuming lines that have indent > baseIndent or are empty
			minIndent := -1
			for i < n {
				l := infos[i]
				if l.text == "" {
					rawLines = append(rawLines, "")
					i++
					continue
				}
				if l.indent <= baseIndent {
					break
				}
				// Keep track of the minimum indentation to shift left
				if minIndent == -1 || l.indent < minIndent {
					minIndent = l.indent
				}
				rawLines = append(rawLines, l.text)
				i++
			}
			// Shift lines left relative to minIndent
			if minIndent > 0 {
				for idx, line := range rawLines {
					if line == "" {
						continue
					}
					// Count spaces and trim
					spaces := countIndent(line)
					if spaces >= minIndent {
						rawLines[idx] = line[minIndent:]
					} else {
						rawLines[idx] = strings.TrimSpace(line)
					}
				}
			}
			verbNode.Lines = rawLines
			nodes = append(nodes, verbNode)
			continue
		}

		// Normal block (or implicit list block)
		// Let's gather the start paragraph (all contiguous lines with the same baseIndent)
		var paraLines []string
		paraLines = append(paraLines, firstLineText) // we already stripped start line's outer spaces
		i++
		if !(ok && isBlockTag) {
			for i < n && infos[i].text != "" && infos[i].indent == baseIndent {
				// Check if the next line starts a new paragraph (e.g. starts with a block tag or is a list item)
				nextLineText := strings.TrimSpace(infos[i].text)
				_, _, _, _, _, isNextTag := parseStartTag(nextLineText)
				isList, _ := isListItem(nextLineText)
				if isNextTag || isList {
					// Starts a new sibling block at the same indentation level, so stop gathering paragraph lines
					break
				}
				paraLines = append(paraLines, nextLineText)
				i++
			}
		}

		// Combine paragraph lines (joining with space)
		paraText := strings.Join(paraLines, " ")

		// If the paragraph text starts with a block tag or macro, parse it
		var blockNode *ast.BlockNode
		if ok && isBlockTag {
			blockNode = &ast.BlockNode{
				TagName:    tagName,
				Attributes: attrs,
				Indent:     baseIndent,
				Rest:       paraText[len(firstLineText)-len(rest):], // get the rest of the text
			}
			if rest == "" && len(paraLines) > 1 {
				blockNode.Rest = strings.TrimSpace(strings.Join(paraLines[1:], " "))
			}
		} else {
			// Check if the paragraph starts with list item marker
			isList, listMarker := isListItem(paraText)
			if isList {
				// Implicit list item block
				tagName := "li"
				restText := paraText[len(listMarker):]
				blockNode = &ast.BlockNode{
					TagName: tagName,
					Indent:  baseIndent,
					Rest:    restText,
				}
				// Save type of list in attributes for list group processing
				if strings.HasPrefix(listMarker, "1.") {
					blockNode.SetAttribute("__list_type", "ol")
				} else {
					blockNode.SetAttribute("__list_type", "ul")
				}
			} else {
				// Standard paragraph block (implicit <p>)
				blockNode = &ast.BlockNode{
					TagName: "p",
					Indent:  baseIndent,
					Rest:    paraText,
				}
			}
		}

		// Now parse child blocks (all subsequent blocks with indentation exactly baseIndent + multiplier)
		var childLines []string
		for i < n {
			l := infos[i]
			if l.text == "" {
				// Keep empty lines for scanning but skip
				i++
				continue
			}
			if l.indent <= baseIndent {
				break
			}
			childLines = append(childLines, lines[l.index])
			i++
		}

		if len(childLines) > 0 {
			children, err := parseBlocks(childLines, multiplier, baseIndent+multiplier)
			if err != nil {
				return nil, err
			}
			blockNode.Children = children
		}

		nodes = append(nodes, blockNode)
	}

	// Implicit List Grouping:
	// We need to group contiguous "li" blocks of the same type under a single "ul" or "ol" block node.
	var groupedNodes []ast.Node
	var currentList *ast.BlockNode

	for _, node := range nodes {
		bNode, ok := node.(*ast.BlockNode)
		if ok && (bNode.TagName == "li" || bNode.GetAttributeOrEmpty("__list_type") != "") {
			listType := bNode.GetAttributeOrEmpty("__list_type")
			if listType == "" {
				listType = "ul" // default to ul for explicit li tag
			}
			bNode.RemoveAttribute("__list_type")

			if currentList != nil && currentList.TagName == listType {
				currentList.Children = append(currentList.Children, bNode)
			} else {
				currentList = &ast.BlockNode{
					TagName:  listType,
					Indent:   bNode.Indent,
					Children: []ast.Node{bNode},
				}
				groupedNodes = append(groupedNodes, currentList)
			}
		} else {
			currentList = nil
			groupedNodes = append(groupedNodes, node)
		}
	}

	return groupedNodes, nil
}

// parseStartTag parses a block tag at the start of a line.
// E.g. `<section id="demo" unnumbered>Demo` -> tagName="section", attrs=[...], rest="Demo", isBlockTag=true, isVerbatim=false, ok=true
func parseStartTag(line string) (tagName string, attrs []ast.Attribute, rest string, isBlockTag bool, isVerbatim bool, ok bool) {
	if !strings.HasPrefix(line, "<") {
		return "", nil, "", false, false, false
	}

	// Find the matching '>' that closes the tag
	// Note: We need to respect quoted values which might contain '>'
	inDoubleQuotes := false
	inSingleQuotes := false
	tagEndIdx := -1
	for idx := 0; idx < len(line); idx++ {
		char := line[idx]
		if char == '"' && !inSingleQuotes {
			inDoubleQuotes = !inDoubleQuotes
		} else if char == '\'' && !inDoubleQuotes {
			inSingleQuotes = !inSingleQuotes
		} else if char == '>' && !inDoubleQuotes && !inSingleQuotes {
			tagEndIdx = idx
			break
		}
	}

	if tagEndIdx == -1 {
		return "", nil, "", false, false, false
	}

	tagContent := line[1:tagEndIdx]
	rest = strings.TrimSpace(line[tagEndIdx+1:])

	// Parse tagName and attributes inside tagContent
	tagContent = strings.TrimSpace(tagContent)
	if len(tagContent) == 0 {
		return "", nil, "", false, false, false
	}

	// Extract tag name
	nameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+`)
	name := nameRegex.FindString(tagContent)
	if name == "" {
		return "", nil, "", false, false, false
	}

	attrsStr := strings.TrimSpace(tagContent[len(name):])
	attrs = parseAttributes(attrsStr)

	isBlockTag = isBlockElement(name)
	isVerbatim = (name == "pre" || name == "x-code")
	return name, attrs, rest, isBlockTag, isVerbatim, true
}

func isBlockElement(name string) bool {
	blocks := map[string]bool{
		"div":         true,
		"section":     true,
		"p":           true,
		"ul":          true,
		"ol":          true,
		"li":          true,
		"pre":         true,
		"h1":          true,
		"h2":          true,
		"h3":          true,
		"h4":          true,
		"h5":          true,
		"h6":          true,
		"article":     true,
		"header":      true,
		"footer":      true,
		"aside":       true,
		"nav":         true,
		"blockquote":  true,
		"figure":      true,
		"figcaption":  true,
		"table":       true,
		"thead":       true,
		"tbody":       true,
		"tr":          true,
		"th":          true,
		"td":          true,
		"form":        true,
		"main":        true,
		"x-include":   true,
		"x-fig":       true,
		"x-quote":     true,
		"x-code":      true,
	}
	return blocks[strings.ToLower(name)]
}

// parseAttributes parses attributes, handles shorthand notations (.class, #id, @src, ?href)
func parseAttributes(s string) []ast.Attribute {
	var attrs []ast.Attribute
	var classes []string

	// Lex attributes manually to handle quotes properly
	i := 0
	n := len(s)
	for i < n {
		// skip whitespace
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' {
			i++
			continue
		}

		// Shorthands:
		if s[i] == '.' {
			// Class shorthand
			i++
			start := i
			for i < n && s[i] != ' ' && s[i] != '\t' && s[i] != '\n' && s[i] != '\r' {
				i++
			}
			classes = append(classes, s[start:i])
			continue
		}
		if s[i] == '#' {
			// ID shorthand
			i++
			start := i
			for i < n && s[i] != ' ' && s[i] != '\t' && s[i] != '\n' && s[i] != '\r' {
				i++
			}
			attrs = append(attrs, ast.Attribute{Key: "id", Value: s[start:i]})
			continue
		}
		if s[i] == '@' {
			// Src shorthand
			i++
			start := i
			for i < n && s[i] != ' ' && s[i] != '\t' && s[i] != '\n' && s[i] != '\r' {
				i++
			}
			attrs = append(attrs, ast.Attribute{Key: "src", Value: s[start:i]})
			continue
		}
		if s[i] == '?' {
			// Href shorthand
			i++
			start := i
			for i < n && s[i] != ' ' && s[i] != '\t' && s[i] != '\n' && s[i] != '\r' {
				i++
			}
			attrs = append(attrs, ast.Attribute{Key: "href", Value: s[start:i]})
			continue
		}

		// Standard attribute: key=value or key
		start := i
		for i < n && s[i] != '=' && s[i] != ' ' && s[i] != '\t' && s[i] != '\n' && s[i] != '\r' {
			i++
		}
		key := s[start:i]

		if i < n && s[i] == '=' {
			i++ // consume '='
			if i < n && s[i] == '"' {
				i++ // consume '"'
				startVal := i
				for i < n && s[i] != '"' {
					i++
				}
				val := s[startVal:i]
				if i < n {
					i++ // consume '"'
				}
				attrs = append(attrs, ast.Attribute{Key: key, Value: val})
			} else if i < n && s[i] == '\'' {
				i++ // consume '\''
				startVal := i
				for i < n && s[i] != '\'' {
					i++
				}
				val := s[startVal:i]
				if i < n {
					i++ // consume '\''
				}
				attrs = append(attrs, ast.Attribute{Key: key, Value: val})
			} else {
				startVal := i
				for i < n && s[i] != ' ' && s[i] != '\t' && s[i] != '\n' && s[i] != '\r' {
					i++
				}
				attrs = append(attrs, ast.Attribute{Key: key, Value: s[startVal:i]})
			}
		} else {
			// Boolean attribute
			attrs = append(attrs, ast.Attribute{Key: key, Value: ""})
		}
	}

	if len(classes) > 0 {
		attrs = append(attrs, ast.Attribute{Key: "class", Value: strings.Join(classes, " ")})
	}

	return attrs
}

// isListItem checks if the text starts with a list marker (e.g. "- ", "* ", "1. ")
func isListItem(text string) (bool, string) {
	if strings.HasPrefix(text, "- ") {
		return true, "- "
	}
	if strings.HasPrefix(text, "* ") {
		return true, "* "
	}
	// Check single-digit ordered list item like "1. ", "2. ", etc.
	if len(text) >= 3 && text[1] == '.' && text[2] == ' ' && text[0] >= '0' && text[0] <= '9' {
		return true, text[:3]
	}
	return false, ""
}

// ParseInline parses raw text into a list of InlineNodes (resolving markdown & inline tags)
func ParseInline(text string) []ast.InlineNode {
	var nodes []ast.InlineNode
	i := 0
	n := len(text)

	for i < n {
		// Double backtick code: ``code``
		if i+1 < n && text[i] == '`' && text[i+1] == '`' {
			start := i + 2
			end := strings.Index(text[start:], "``")
			if end != -1 {
				nodes = append(nodes, &ast.CodeNode{Content: text[start : start+end]})
				i = start + end + 2
				continue
			}
		}

		// Single backtick code: `code`
		if text[i] == '`' {
			start := i + 1
			end := strings.IndexByte(text[start:], '`')
			if end != -1 {
				nodes = append(nodes, &ast.CodeNode{Content: text[start : start+end]})
				i = start + end + 1
				continue
			}
		}

		// Bold/Strong: *text* or **text**
		if text[i] == '*' {
			isDouble := i+1 < n && text[i+1] == '*'
			delim := "*"
			offset := 1
			if isDouble {
				delim = "**"
				offset = 2
			}
			start := i + offset
			end := strings.Index(text[start:], delim)
			if end != -1 {
				nodes = append(nodes, &ast.StrongNode{Content: text[start : start+end]})
				i = start + end + offset
				continue
			}
		}

		// Italic/Emphasis: _text_
		if text[i] == '_' {
			start := i + 1
			end := strings.IndexByte(text[start:], '_')
			if end != -1 {
				nodes = append(nodes, &ast.EmphasisNode{Content: text[start : start+end]})
				i = start + end + 1
				continue
			}
		}

		// Inline HTML Tag / Macro Tag: <tagName ...> ... </tagName> or self-closing/empty like <x-ref ...>
		if text[i] == '<' {
			// Find matching closing '>' for this tag start
			inDoubleQuotes := false
			inSingleQuotes := false
			tagEndIdx := -1
			for idx := i; idx < n; idx++ {
				char := text[idx]
				if char == '"' && !inSingleQuotes {
					inDoubleQuotes = !inDoubleQuotes
				} else if char == '\'' && !inDoubleQuotes {
					inSingleQuotes = !inSingleQuotes
				} else if char == '>' && !inDoubleQuotes && !inSingleQuotes {
					tagEndIdx = idx
					break
				}
			}

			if tagEndIdx != -1 {
				tagContent := text[i+1 : tagEndIdx]
				tagContent = strings.TrimSpace(tagContent)
				nameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+`)
				name := nameRegex.FindString(tagContent)

				if name != "" {
					attrsStr := strings.TrimSpace(tagContent[len(name):])
					attrs := parseAttributes(attrsStr)

					// Check if this is self-closing or if we can find a matching closing tag </name>
					closingTag := "</" + name + ">"
					closeIdx := strings.Index(text[tagEndIdx+1:], closingTag)

					if closeIdx != -1 {
						innerContent := text[tagEndIdx+1 : tagEndIdx+1+closeIdx]
						nodes = append(nodes, &ast.InlineTagNode{
							TagName:    name,
							Attributes: attrs,
							Children:   ParseInline(innerContent),
						})
						i = tagEndIdx + 1 + closeIdx + len(closingTag)
						continue
					} else {
						// Self-closing or no children tag (e.g. <x-ref ...>)
						nodes = append(nodes, &ast.InlineTagNode{
							TagName:    name,
							Attributes: attrs,
						})
						i = tagEndIdx + 1
						continue
					}
				}
			}
		}

		// Ordinary text character
		start := i
		for i < n {
			if (i+1 < n && text[i] == '`' && text[i+1] == '`') ||
				text[i] == '`' ||
				text[i] == '*' ||
				text[i] == '_' ||
				text[i] == '<' {
				break
			}
			i++
		}
		nodes = append(nodes, &ast.TextNode{Content: text[start:i]})
	}

	return nodes
}
