package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/hesusruiz/riteng/pkg/generator"
	"github.com/hesusruiz/riteng/pkg/lexer"
	"github.com/hesusruiz/riteng/pkg/parser"
)

func main() {
	// Flags
	inputPtr := flag.String("i", "", "Input file path (default: stdin)")
	outputPtr := flag.String("o", "", "Output file path (default: stdout)")
	flag.Parse()

	var inputData []byte
	var err error

	// Read Input
	if *inputPtr != "" {
		inputData, err = os.ReadFile(*inputPtr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
			os.Exit(1)
		}
	} else {
		inputData, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
	}

	// Process
	l := lexer.New(string(inputData))
	p := parser.New(l)
	doc := p.ParseDocument()

	gen := generator.New()
	html, err := gen.Generate(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating HTML: %v\n", err)
		os.Exit(1)
	}

	// Write Output
	if *outputPtr != "" {
		err = os.WriteFile(*outputPtr, []byte(html), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Print(html)
	}
}
