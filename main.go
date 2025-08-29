package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	var (
		promptFile     = flag.String("f", "", "Path to YAML prompt file (required)")
		outputFile     = flag.String("o", "", "Output file path (default: stdout)")
		maxWords       = flag.Int("max-words", 128000, "Maximum words in compiled output")
		delimiterStyle = flag.String("delimiter-style", "xml", "Delimiter style: xml, minimal, none, full")
		help           = flag.Bool("h", false, "Show help message")
		helpLong       = flag.Bool("help", false, "Show help message")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `pcp: Prompt Composition Processor

Usage: pcp -f <prompt-file> [-o <output-file>] [-max-words <limit>] [-delimiter-style <style>] [-h]

Compiles content from multiple sources into a single text output for AI agents.

Flags:
  -f string
        Path to YAML prompt file (required)
  -o string
        Output file path (default: stdout)
  -max-words int
        Maximum words in compiled output (default: 128000)
  -delimiter-style string
        Delimiter style: xml, minimal, none, full (default: xml)
  -h, -help
        Show this help message

Important: All errors are written to STDERR to ensure safe piping to agents.

Usage Patterns:
  RECOMMENDED: Use file output for reliable agent workflows
    pcp -f prompt.yml -o context.txt && agent < context.txt
  
  AVOID: Command substitution with piping (agent runs even if pcp fails)
    $(pcp -f prompt.yml) | agent

Prompt File Format:
  - prompt:
      - file: "relative/path/to/file.txt"
      - prompt: "nested-prompt.yml"
      - command: "ls -la"
      - text: "Literal text content"

Text Field Special Characters:
  Multiline text using YAML literal block scalar:
  - text: |
      This is line one
      This is line two with a tab:	<tab here>
      Line three
  
  Escaped characters in quoted strings:
  - text: "Line with\nnewline and\ttab"
  
  Raw strings with minimal escaping:
  - text: >
      This text will be folded
      but preserves paragraph breaks
`)
	}

	flag.Parse()

	if *help || *helpLong {
		flag.Usage()
		os.Exit(0)
	}

	if *promptFile == "" {
		fmt.Fprintf(os.Stderr, "Error: -f flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Validate delimiter style
	validStyles := map[string]bool{
		"xml":     true,
		"minimal": true,
		"none":    true,
		"full":    true,
	}
	if !validStyles[*delimiterStyle] {
		fmt.Fprintf(os.Stderr, "Error: invalid delimiter style '%s'. Must be one of: xml, minimal, none, full\n", *delimiterStyle)
		flag.Usage()
		os.Exit(1)
	}

	if err := processPromptFile(*promptFile, *outputFile, *maxWords, *delimiterStyle); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func processPromptFile(promptFile, outputFile string, maxWords int, delimiterStyle string) error {
	ctx := NewProcessingContext(promptFile, maxWords, delimiterStyle)

	if err := validatePromptFileStructure(promptFile, ctx); err != nil {
		return err
	}

	ctx = NewProcessingContext(promptFile, maxWords, delimiterStyle)

	pf, err := parsePromptFile(promptFile)
	if err != nil {
		return err
	}

	var compiledContent CompiledContent
	for _, op := range pf.Prompt {
		section, err := processOperation(op, ctx)
		if err != nil {
			return err
		}
		compiledContent.Sections = append(compiledContent.Sections, section)
	}

	output, err := compileOutput(compiledContent, delimiterStyle)
	if err != nil {
		return err
	}

	if outputFile == "" {
		fmt.Print(output)
	} else {
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output file %s: %w", outputFile, err)
		}
	}

	return nil
}

func compileOutput(content CompiledContent, delimiterStyle string) (string, error) {
	var result strings.Builder

	for i, section := range content.Sections {
		// Add section header (if any)
		if delimiterStyle != "none" {
			if i == 0 {
				// First section: remove leading newline from delimiter
				result.WriteString(strings.TrimLeft(formatSectionHeader(section.Source, delimiterStyle), "\n"))
			} else {
				result.WriteString(formatSectionHeader(section.Source, delimiterStyle))
			}
		}

		// Add the normalized content (always ends with exactly one newline)
		result.WriteString(section.Content)
	}

	// Ensure output ends with exactly one newline to prevent shell % character
	output := strings.TrimRight(result.String(), "\n") + "\n"
	return output, nil
}
