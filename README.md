# PCP: Prompt Composition Processor

*A command-line tool that deterministically compiles text content from multiple sources (files, commands, text snippets) into a single output for feeding to AI agents, ensuring operators have complete control over context.*

## Overview

PCP is a Go-based CLI tool that processes YAML prompt files to compile content from various sources into a single, formatted output. It supports four operation types: file inclusion, nested prompt processing, command execution, and literal text blocks.

## Features

- **File Operations**: Read text files relative to prompt file location with binary file detection
- **Nested Prompts**: Recursively process other prompt files with circular reference detection  
- **Command Execution**: Execute shell commands and capture output with proper error handling
- **Text Blocks**: Include literal text with support for multiline content and special characters
- **Word Limits**: Configurable word count limits with validation (default: 128,000 words)
- **Safe Piping**: All errors written to STDERR to prevent contamination of piped output
- **Cross-platform**: Portable Go implementation supporting Linux, macOS, and Windows

## Installation

```bash
go install github.com/riazarbi/pcp@latest
```

Or build from source:

```bash
git clone https://github.com/riazarbi/pcp.git
cd pcp
xc build
```

## Usage

### Basic Usage

```bash
# Compile prompt to stdout
pcp -f my-prompt.yml

# Write to file (recommended for agent workflows)
pcp -f my-prompt.yml -o compiled-context.txt

# Set custom word limit
pcp -f my-prompt.yml -max-words 50000

# Use different delimiter styles
pcp -f my-prompt.yml --delimiter-style=none    # No delimiters, clean content
pcp -f my-prompt.yml --delimiter-style=minimal # Simple delimiters
pcp -f my-prompt.yml --delimiter-style=full    # Verbose original format
```

### Safe Piping Patterns

```bash
# RECOMMENDED: File output pattern
pcp -f prompt.yml -o context.txt && agent < context.txt

# SAFE: Direct piping (errors visible, no input on failure)
pcp -f prompt.yml | agent

# AVOID: Command substitution (agent runs even if pcp fails)
$(pcp -f prompt.yml) | agent
```

## Prompt File Format

Prompt files are YAML documents with a single `prompt` key containing an array of operations:

```yaml
prompt:
  - file: "relative/path/to/file.txt"
  - prompt: "nested-prompt.yml"  
  - command: "git status --porcelain"
  - text: |
      Multiline text block with
      special characters: tabs	and newlines
      
      Preserves formatting exactly.
  - text: "Single line with\\nnewline and\\ttab"
```

### Operation Types

- **file**: Include contents of text files (binary files trigger errors)
- **prompt**: Recursively process nested prompt files
- **command**: Execute shell commands and include output
- **text**: Include literal text content

### Text Field Formatting

```yaml
# Literal block scalar (preserves newlines)
- text: |
    Line one
    Line two with tab:	<tab>
    Line three

# Folded scalar (joins lines, preserves paragraphs) 
- text: >
    This text will be folded
    but preserves paragraph breaks

# Quoted string with escape sequences
- text: "Line with\\nnewline and\\ttab"
```

## Output Format

Content is compiled with formatted section headers (default XML style, agent-friendly):

```
<!-- pcp-source: filename.txt -->
[file contents]
<!-- pcp-source: git status --porcelain -->
[command output]
<!-- pcp-source: text -->
[literal text content]
```

### Delimiter Styles

Control output formatting with `--delimiter-style`:

- **xml** (default): `<!-- pcp-source: filename.txt -->` - Agent-friendly, won't interfere with content parsing
- **minimal**: `=== PCP SOURCE: filename.txt ===` - Visible but less noisy than full style  
- **full**: `----------------------------------\nBEGIN: filename.txt\n----------------------------------` - Original verbose format
- **none**: No delimiters, just concatenated content

## Error Handling

- Missing files: Informative error with file path
- Binary files: Detection and rejection with clear message
- Command failures: Distinction between execution failure and exit status 1
- Circular references: Detection in nested prompt structures
- Word limits: Validation before output generation
- YAML structure: Validation with helpful error messages

All errors are written to STDERR to ensure safe piping to downstream tools.

## Tasks

### build
```
go build -o pcp
```

### test
```
go test -v
```

### test-coverage
```
go test -coverprofile=coverage.out
go tool cover -func=coverage.out
```

### coverage-html
```
go test -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
echo "Coverage report generated: coverage.html"
```

### lint
```
go vet ./...
```

### fmt
```
go fmt ./...
```

### clean
```
rm -f pcp coverage.out coverage.html
rm -rf demo
go clean
```

### ci
```
go fmt ./...
go vet ./...
go test -coverprofile=coverage.out
go tool cover -func=coverage.out
go build -o pcp
```

### demo
```bash
echo "Creating demo files..."
mkdir -p demo

cat > demo/intro.md << 'EOF'
# PCP Demo
This is a demonstration of the Prompt Composition Processor.
EOF

cat > demo/sample.txt << 'EOF' 
This is sample content from a text file.
It contains multiple lines and demonstrates
how pcp can include file contents.
EOF

cat > demo/nested.yml << 'EOF'
prompt:
  - file: "sample.txt"
  - text: "This is nested prompt content"
EOF

cat > demo/main.yml << 'EOF'
prompt:
  - file: "intro.md"
  - command: "echo 'Current time:' && date"
  - prompt: "nested.yml"
  - text: |
      This is a multiline text block.
      It demonstrates how pcp handles
      complex text formatting.
EOF

echo "Building pcp..."
go build -o pcp

echo ""
echo "Executing: ./pcp -f demo/main.yml"  
echo "----------------------------------------"
cd demo && ../pcp -f main.yml
cd ..

echo ""
echo "Demo files created in demo/ directory"
echo "Clean up with: xc clean"
```

## Development

This project follows Go best practices with:
- 100% test coverage using real file system operations
- No mocking - all tests use actual files, commands, and YAML processing  
- Comprehensive error handling and edge case coverage
- Cross-platform compatibility testing

## License

MIT License - see LICENSE file for details.