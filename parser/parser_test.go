package parser

import (
	"strings"
	"testing"
)

func TestParseMetadata(t *testing.T) {
	input := `---
title: Sample Doc
author: Jesus Ruiz
---
<p>Hello World
`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error parsing: %v", err)
	}

	if doc.Metadata["title"] != "Sample Doc" {
		t.Errorf("expected title 'Sample Doc', got %v", doc.Metadata["title"])
	}
	if doc.Metadata["author"] != "Jesus Ruiz" {
		t.Errorf("expected author 'Jesus Ruiz', got %v", doc.Metadata["author"])
	}
}

func TestShorthandAttributes(t *testing.T) {
	input := `<section id="sec1" .class1 .class2 ?link @image>Hello`
	_, attrs, rest, isBlock, _, ok := parseStartTag(input)
	if !ok {
		t.Fatalf("expected parsing to succeed")
	}
	if !isBlock {
		t.Errorf("expected section to be recognized as block tag")
	}
	if rest != "Hello" {
		t.Errorf("expected rest 'Hello', got '%s'", rest)
	}

	attrMap := make(map[string]string)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value
	}

	if attrMap["id"] != "sec1" {
		t.Errorf("expected id 'sec1', got '%s'", attrMap["id"])
	}
	if attrMap["class"] != "class1 class2" {
		t.Errorf("expected class 'class1 class2', got '%s'", attrMap["class"])
	}
	if attrMap["href"] != "link" {
		t.Errorf("expected href 'link', got '%s'", attrMap["href"])
	}
	if attrMap["src"] != "image" {
		t.Errorf("expected src 'image', got '%s'", attrMap["src"])
	}
}

func TestInlineParsing(t *testing.T) {
	input := "This is *bold*, _italic_, and `code` with <x-ref href=\"target\">ref</x-ref>."
	nodes := ParseInline(input)
	if len(nodes) < 7 {
		t.Fatalf("expected at least 7 inline nodes, got %d", len(nodes))
	}
}
