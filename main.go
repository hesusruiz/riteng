package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"

	"github.com/hesusruiz/riteng/parser"
	"github.com/hesusruiz/riteng/processor"
)

type TemplateData struct {
	Title string
	Body  template.HTML
	Meta  map[string]interface{}
}

func main() {
	templateFlag := flag.String("template", "", "path to the Go HTML template file")
	outputFlag := flag.String("o", "", "path to the output HTML file (default: stdout)")
	flag.Parse()

	args := flag.Args()

	var input io.Reader
	var baseDir string
	var err error

	if len(args) > 0 {
		filePath := args[0]
		file, err := os.Open(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		input = file
		baseDir = filepath.Dir(filePath)
	} else {
		input = os.Stdin
		baseDir, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
			os.Exit(1)
		}
	}

	// Parse Document
	doc, err := parser.Parse(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parser error: %v\n", err)
		os.Exit(1)
	}

	// Compile to HTML Body
	proc := processor.NewProcessor(processor.ProcessOptions{
		BaseDir: baseDir,
	})
	bodyHTML, err := proc.Process(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Processor error: %v\n", err)
		os.Exit(1)
	}

	// Prepare title and metadata
	title := "Document"
	if t, ok := doc.Metadata["title"].(string); ok {
		title = t
	}

	// Load Go HTML Template
	var tmpl *template.Template
	if *templateFlag != "" {
		tmpl, err = template.ParseFiles(*templateFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading template: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Check for local template.html
		if _, err := os.Stat("template.html"); err == nil {
			tmpl, err = template.ParseFiles("template.html")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading local template.html: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Fallback default inline template
			const fallbackTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>{{.Title}}</title>
</head>
<body>
  {{if .Title}}<h1>{{.Title}}</h1>{{end}}
  {{.Body}}
</body>
</html>`
			tmpl, err = template.New("fallback").Parse(fallbackTemplate)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing fallback template: %v\n", err)
				os.Exit(1)
			}
		}
	}

	// Prepare template data
	data := TemplateData{
		Title: title,
		Body:  template.HTML(bodyHTML),
		Meta:  doc.Metadata,
	}

	// Set output destination
	var output io.Writer = os.Stdout
	if *outputFlag != "" {
		outFile, err := os.Create(*outputFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer outFile.Close()
		output = outFile
	}

	// Execute Template
	if err := tmpl.Execute(output, data); err != nil {
		fmt.Fprintf(os.Stderr, "Template execution error: %v\n", err)
		os.Exit(1)
	}
}
