package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	// Check for demo subcommand first
	if len(os.Args) > 1 && os.Args[1] == "demo" {
		if err := runDemo(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

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

Usage: 
  pcp -f <prompt-file> [-o <output-file>] [-max-words <limit>] [-delimiter-style <style>] [-h]
  pcp demo

Compiles content from multiple sources into a single text output for AI agents.

Commands:
  demo        Create and run a demonstration with sample files

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

func runDemo() error {
	fmt.Println("Creating PCP demonstration...")

	// Create demo directory
	if err := os.MkdirAll("demo", 0755); err != nil {
		return fmt.Errorf("failed to create demo directory: %w", err)
	}

	// Create demo files
	files := map[string]string{
		"demo/intro.md": `# PCP Demo
This is a demonstration of the Prompt Composition Processor.

PCP allows you to combine content from multiple sources:
- Files (like this markdown file)
- Command output (like git status or system info)  
- Literal text blocks
- Other prompt files (for modular organization)

All of this gets compiled into a single, AI-ready context file.
`,
		"demo/sample.txt": `This is sample content from a text file.
It contains multiple lines and demonstrates
how pcp can include file contents seamlessly.

Files can be any text format:
- Source code
- Documentation  
- Configuration files
- Data files
- And more
`,
		"demo/nested.yml": `prompt:
  - file: "sample.txt"
  - text: |
      This content comes from a nested prompt file.
      
      Nested prompts allow you to:
      - Build modular, reusable components
      - Share common content across projects
      - Keep complex prompts organized
  - command: "echo 'Nested prompts can also include commands.'"
`,
		"demo/main.yml": `prompt:
  - file: "intro.md"
  - command: "echo 'Current time:' && date"
  - prompt: "nested.yml"  
  - text: |
      This is a multiline text block that demonstrates
      how pcp handles complex text formatting.
      
      You can include:
      - Instructions for AI agents
      - Context information
      - Notes and explanations
      - Anything else you need
      
      The result is a single, well-formatted file
      that you can pipe directly to AI tools.
`,
	}

	// Write all demo files
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create %s: %w", path, err)
		}
		fmt.Printf("Created %s\n", path)
	}

	fmt.Println("\nRunning PCP demonstration...")
	fmt.Println("----------------------------------------")

	// Change to demo directory and run PCP
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir("demo"); err != nil {
		return fmt.Errorf("failed to change to demo directory: %w", err)
	}
	defer os.Chdir(originalDir)

	// Process the demo prompt file
	if err := processPromptFile("main.yml", "", 128000, "xml"); err != nil {
		return fmt.Errorf("failed to process demo: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\nDemo completed successfully.\n")
	fmt.Fprintf(os.Stderr, "Demo files created in demo/ directory\n")
	fmt.Fprintf(os.Stderr, "Clean up with: rm -rf demo/\n")
	fmt.Fprintf(os.Stderr, "\nTry different delimiter styles:\n")
	fmt.Fprintf(os.Stderr, "   pcp -f demo/main.yml -delimiter-style=minimal\n")
	fmt.Fprintf(os.Stderr, "   pcp -f demo/main.yml -delimiter-style=none\n")
	fmt.Fprintf(os.Stderr, "   pcp -f demo/main.yml -delimiter-style=full\n")

	return nil
}
