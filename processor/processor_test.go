package processor

import (
	"strings"
	"testing"

	"github.com/hesusruiz/riteng/parser"
)

func TestProcessorSectionNumberingAndMacros(t *testing.T) {
	input := `---
title: Test Document
---
<section id="s1">Introduction
  This is intro.
  <section id="s1.1">Sub section
    This is sub section.
  <section id="s1.2" unnumbered>Unnumbered Sub section
    Some unnumbered text.

<section id="s2">Results
  More text.
  <x-fig src="pic.jpg">Picture caption
  See <x-ref href="s1.1"> and <x-ref href="s2">.
`
	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	proc := NewProcessor(ProcessOptions{BaseDir: "."})
	htmlOutput, err := proc.Process(doc)
	if err != nil {
		t.Fatalf("processor error: %v", err)
	}

	// Verify section headers generated correct numbering
	if !strings.Contains(htmlOutput, "<h2>1. Introduction</h2>") {
		t.Errorf("expected <h2>1. Introduction</h2>, got output:\n%s", htmlOutput)
	}
	if !strings.Contains(htmlOutput, "<h3>1.1. Sub section</h3>") {
		t.Errorf("expected <h3>1.1. Sub section</h3>, got output:\n%s", htmlOutput)
	}
	if !strings.Contains(htmlOutput, "<h3>Unnumbered Sub section</h3>") {
		t.Errorf("expected <h3>Unnumbered Sub section</h3>, got output:\n%s", htmlOutput)
	}
	if !strings.Contains(htmlOutput, "<h2>2. Results</h2>") {
		t.Errorf("expected <h2>2. Results</h2>, got output:\n%s", htmlOutput)
	}

	// Verify x-fig conversion and numbering
	if !strings.Contains(htmlOutput, "<figcaption>Fig 1. Picture caption</figcaption>") {
		t.Errorf("expected Fig 1. Picture caption, got output:\n%s", htmlOutput)
	}

	// Verify x-ref replacement
	if !strings.Contains(htmlOutput, `<a href="#s1.1">Section 1.1. Sub section</a>`) {
		t.Errorf("expected Section 1.1. Sub section link, got output:\n%s", htmlOutput)
	}
	if !strings.Contains(htmlOutput, `<a href="#s2">Section 2. Results</a>`) {
		t.Errorf("expected Section 2. Results link, got output:\n%s", htmlOutput)
	}
}

func TestProcessorVerbatimCode(t *testing.T) {
	input := `<x-code id="code1">Verbatim Code Block
  line 1
    line 2
  line 3
`
	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	proc := NewProcessor(ProcessOptions{BaseDir: "."})
	htmlOutput, err := proc.Process(doc)
	if err != nil {
		t.Fatalf("processor error: %v", err)
	}

	// Check shifted verbatim block rendering
	expectedCode := "line 1\n  line 2\nline 3"
	if !strings.Contains(htmlOutput, expectedCode) {
		t.Errorf("expected verbatim code block:\n%s\ngot:\n%s", expectedCode, htmlOutput)
	}
}
