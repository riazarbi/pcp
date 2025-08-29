# PCP: Prompt Composition Processor

*A command-line tool that deterministically compiles text content from multiple sources (files, commands, text snippets) into a single output for feeding to AI agents, ensuring operators have complete control over context.*

## Past Attempts

If this user story has been attempted before, the changes made will appear in the git diff. Our policy is to only make a single commit per user story, so you can always review the git diff to review progress across attempts.

## Requirements

*Specific, measurable acceptance criteria. These define when the story is complete.*

- Tool must accept a compulsory `-f` flag pointing to a valid YAML prompt file
- Tool must validate YAML file structure according to specification (single `prompt` key with array of operations)
- Tool must support four operation types under `prompt`: `file`, `prompt`, `command`, `text`
- File operations must read text/YAML files relative to prompt file location and error on binary files or missing files
- Prompt operations must recursively process other prompt files relative to current prompt file location
- Command operations must execute shell commands, include STDOUT/STDERR, warn on exit status 1, error on execution failure
- Text operations must include literal text content with support for multiline text and special characters (newlines, tabs, etc.)
- Tool must compile all content in order specified in YAML file(s)
- Tool must separate each content section with formatted header: newline + dashes + "BEGIN: [source]" + dashes + newline
- Tool must output compiled text to STDOUT by default
- Tool must support optional `-o` flag to write output to specified file
- Tool must provide comprehensive help via `-h` or `-help` flags including guidance on escaping special characters in text fields
- Tool must support optional `-max-words` flag with default limit of 128,000 words
- Tool must error if compiled output exceeds max word limit
- Tool must write all error messages to STDERR (never STDOUT) to prevent errors from being piped to agents
- Tool must be portable and built using Go
- Tool must achieve 100% test coverage with no mocking
- Project must include README with xc build and test instructions

## Rules

*Important constraints or business rules that must be followed.*

- YAML files must contain exactly one `prompt` key at root level
- All file paths must be relative to the prompt file containing the reference
- Binary files must trigger informative errors
- Missing files must trigger informative errors
- Command execution failures must trigger informative errors
- Command exit status 1 should trigger warnings but not stop processing
- Recursive prompt file validation must be performed before processing
- Content compilation must maintain exact order specified in YAML structure
- All error messages must be written to STDERR to prevent interference when output is piped
- Word count validation must occur before output generation
- Word counting must use whitespace-based tokenization for consistency

## Domain

*Core domain model in pseudo-code if applicable.*

```go
type PromptFile struct {
    Prompt []Operation `yaml:"prompt"`
}

type Operation struct {
    File    *string `yaml:"file,omitempty"`
    Prompt  *string `yaml:"prompt,omitempty"`
    Command *string `yaml:"command,omitempty"`
    Text    *string `yaml:"text,omitempty"`
}

type CompiledContent struct {
    Sections []ContentSection
}

type ContentSection struct {
    Source  string
    Content string
    Type    OperationType
}

type OperationType int
const (
    FileOp OperationType = iota
    PromptOp
    CommandOp
    TextOp
)
```

## Extra Considerations

*Edge cases, non-functional requirements, or gotchas.*

- Handle circular references in prompt file inclusions
- Manage relative path resolution across nested prompt files
- Ensure proper error context (which file, which operation) in error messages
- Handle large file contents and command outputs efficiently
- Support cross-platform command execution
- Validate YAML structure before processing any operations
- Handle special characters and encoding in file contents and command outputs
- Support multiline text in YAML text fields with proper newline and tab character handling
- Provide clear guidance on escaping special characters in YAML text fields
- Handle output size limits gracefully with informative error messages
- Ensure error messages never contaminate STDOUT when piped to other tools/agents
- Consider memory usage with large recursive prompt structures
- Handle concurrent command execution safely if implementing parallelization
- Implement efficient word counting that doesn't require loading entire content into memory

## Testing Considerations

*What types of tests are needed and what scenarios to cover.*

**CRITICAL: 100% test coverage required with no mocking. All functionality must be tested through real file system operations, actual command execution, and genuine YAML processing.**

- Unit tests for YAML parsing and validation (using real YAML files)
- Unit tests for each operation type (file, prompt, command, text) with real file system interactions
- Integration tests for complete prompt file processing using temporary directories and files
- Error handling tests for missing files, invalid commands, circular references (using actual missing files and failing commands)
- Path resolution tests for relative paths in nested structures (using real directory hierarchies)
- Command execution tests with various exit codes (using real shell commands)
- Binary file detection tests (using actual binary files)
- Large file handling tests (using generated large files)
- Cross-platform compatibility tests (testing actual OS-specific behavior)
- Special character handling tests (newlines, tabs, quotes, backslashes in text fields)
- Multiline text field tests with various YAML formatting approaches
- Error output redirection tests (ensuring errors go to STDERR, not STDOUT)
- Piping scenario tests (tool output piped to other commands)
- Word count limit tests with content at and exceeding the limit
- Performance tests with real data to ensure acceptable response times
- Memory usage tests with large recursive structures using real files

## Implementation Notes

*Architectural patterns, coding standards, or technology preferences.*

- Use Go programming language
- Follow patterns from @go_agent_best_practices.md, @go_code_organization.md, and @testing_guide.md
- Use xc task runner for build and CI/CD tooling (no cloud CI/CD)
- Build incrementally with tests added throughout development
- Implement clean separation between parsing, validation, and execution phases
- Design for testability without mocking (use real file operations in tests)
- Follow Go standard library patterns for CLI tools (flag package)
- Implement proper error wrapping and context
- Use structured logging for debugging and troubleshooting
- Structure code to enable 100% test coverage through integration testing
- Use temporary directories and files for testing file operations
- Test command execution using safe, cross-platform commands (echo, ls, etc.)

## Specification by Example

*Concrete examples: API samples, user flows, or interaction scenarios.*

### Example 1: Basic Usage
```bash
$ pcp -f my-prompt.yml
```

### Example 2: Output to File
```bash
$ pcp -f my-prompt.yml -o compiled-context.txt
```

### Example 3: Word Limit Control
```bash
# Use default 128,000 word limit
$ pcp -f my-prompt.yml

# Set custom word limit
$ pcp -f my-prompt.yml -max-words 50000

# Error example (output too large)
$ pcp -f large-prompt.yml -max-words 1000
Error: Compiled output (1,234 words) exceeds maximum word limit (1,000 words)
```

### Example 4: Piping Safety and Usage Patterns
```bash
# UNSAFE: Command substitution with piping
$ $(pcp -f prompt.yml) | agent
# Problem: If pcp errors, agent still runs with empty input

# SAFE: File output pattern (recommended)
$ pcp -f prompt.yml -o context.txt && agent < context.txt
# If pcp fails, agent won't run at all

# SAFE: Check exit status first
$ if output=$(pcp -f prompt.yml); then
    echo "$output" | agent
  else
    echo "pcp failed, not running agent" >&2
    exit 1
  fi

# SAFE: Use shell error handling
$ set -e  # Exit on any command failure
$ output=$(pcp -f prompt.yml)
$ echo "$output" | agent

# SAFE: Direct piping (errors visible, agent gets no input on failure)
$ pcp -f prompt.yml | agent
# If pcp errors, agent receives no input and error appears on console
```

### Example 5: README with xc Instructions
```markdown
# PCP: Prompt Composition Processor

## Building

```bash
xc build
```

## Testing

```bash
# Run all tests
xc test

# Run tests with coverage
xc test-coverage

# View coverage report
xc coverage-html
```

## Development

```bash
# Run linting
xc lint

# Format code
xc fmt

# Clean build artifacts
xc clean
```
```

### Example 6: Help Usage
```bash
$ pcp -h
pcp: Prompt Composition Processor

Usage: pcp -f <prompt-file> [-o <output-file>] [-max-words <limit>] [-h]

Compiles content from multiple sources into a single text output for AI agents.

Flags:
  -f string
        Path to YAML prompt file (required)
  -o string
        Output file path (default: stdout)
  -max-words int
        Maximum words in compiled output (default: 128000)
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
```

### Example 7: Sample YAML Structure
```yaml
# main-prompt.yml
- prompt:
    - file: "context/introduction.md"
    - prompt: "shared/common-context.yml"
    - command: "git status --porcelain"
    - text: |
        Additional instructions for the agent.
        This is a multiline text block.
        
        It preserves line breaks and formatting.
        	This line has a tab character.
    - file: "specs/requirements.md"
    - text: "Single line with\nnewline and\ttab characters"
```

### Example 8: Expected Output Format
```
----------------------------------
BEGIN: context/introduction.md
----------------------------------
[file contents here]

----------------------------------
BEGIN: shared/common-context.yml->file1.md
----------------------------------
[nested file contents here]

----------------------------------
BEGIN: git status --porcelain
----------------------------------
[command output here]

----------------------------------
BEGIN: text
----------------------------------
Additional instructions for the agent

----------------------------------
BEGIN: specs/requirements.md
----------------------------------
[file contents here]
```

## Verification

*Actionable checklist to verify story completion.*

- [ ] Tool compiles and runs on target platforms (Linux, macOS, Windows)
- [ ] Required `-f` flag validation works correctly
- [ ] YAML parsing correctly identifies valid and invalid prompt file structures
- [ ] File operations read text files and error appropriately on binary files
- [ ] Prompt operations recursively process nested prompt files
- [ ] Command operations execute and capture STDOUT/STDERR correctly
- [ ] Text operations include literal content correctly
- [ ] Content compilation maintains specified order
- [ ] Output formatting includes proper section separators
- [ ] STDOUT output works by default
- [ ] `-o` flag writes to specified output file correctly
- [ ] `-max-words` flag enforces word limits correctly with default of 128,000
- [ ] Word count validation prevents output generation when limit exceeded
- [ ] All error messages are written to STDERR, never STDOUT
- [ ] Piping scenarios work correctly (errors don't contaminate piped output)
- [ ] Help flags (`-h`, `-help`) display comprehensive usage information
- [ ] Error messages are informative and include proper context
- [ ] Relative path resolution works correctly across nested prompt files
- [ ] All edge cases handle gracefully with appropriate error messages
- [ ] 100% test coverage achieved with no mocking
- [ ] All tests use real file system operations, actual commands, and genuine YAML processing
- [ ] Coverage report shows 100% line and branch coverage
- [ ] Special character handling works correctly (newlines, tabs, quotes, backslashes)
- [ ] Multiline text fields process correctly with various YAML formatting styles
- [ ] Help documentation includes clear guidance on text field character escaping
- [ ] Tool can be built using xc task runner
- [ ] All xc tasks in README.md run successfully.
- [ ] Performance is acceptable for typical use cases (multiple files, moderate command outputs)