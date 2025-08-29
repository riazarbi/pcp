package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMain_Help(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"help flag short", []string{"-h"}},
		{"help flag long", []string{"-help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("go", append([]string{"run", "."}, tt.args...)...)
			output, err := cmd.CombinedOutput()

			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 0 {
					t.Errorf("Expected exit code 0, got: %v", err)
				}
			}

			outputStr := string(output)
			if !strings.Contains(outputStr, "pcp: Prompt Composition Processor") {
				t.Error("Help output should contain program description")
			}
			if !strings.Contains(outputStr, "Usage:") {
				t.Error("Help output should contain usage information")
			}
		})
	}
}

func TestMain_RequiredFlag(t *testing.T) {
	cmd := exec.Command("go", "run", ".")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Error("Expected non-zero exit code when -f flag is missing")
	}

	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
		t.Errorf("Expected exit code 1, got: %v", err)
	}

	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "Error: -f flag is required") {
		t.Error("Should show error message for missing -f flag")
	}
}

func TestProcessPromptFile_BasicOperations(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("Hello World"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	promptFile := filepath.Join(tmpDir, "prompt.yml")
	promptContent := `prompt:
  - file: "test.txt"
  - command: "echo 'Command output'"
  - text: |
      This is multiline text
      with special characters: tabs	and newlines
  - text: "Single line with\nnewline and\ttab"`

	err = os.WriteFile(promptFile, []byte(promptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create prompt file: %v", err)
	}

	outputFile := filepath.Join(tmpDir, "output.txt")
	err = processPromptFile(promptFile, outputFile, 128000, "xml")
	if err != nil {
		t.Fatalf("processPromptFile failed: %v", err)
	}

	output, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	outputStr := string(output)

	if !strings.Contains(outputStr, "<!-- pcp-source: test.txt -->") {
		t.Error("Output should contain file section header")
	}
	if !strings.Contains(outputStr, "Hello World") {
		t.Error("Output should contain file content")
	}
	if !strings.Contains(outputStr, "<!-- pcp-source: echo 'Command output' -->") {
		t.Error("Output should contain command section header")
	}
	if !strings.Contains(outputStr, "Command output") {
		t.Error("Output should contain command output")
	}
	if !strings.Contains(outputStr, "<!-- pcp-source: text -->") {
		t.Error("Output should contain text section headers")
	}
	if !strings.Contains(outputStr, "This is multiline text") {
		t.Error("Output should contain multiline text")
	}
}

func TestProcessPromptFile_NestedPrompts(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "nested")
	err := os.MkdirAll(nestedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	testFile := filepath.Join(nestedDir, "nested.txt")
	err = os.WriteFile(testFile, []byte("Nested content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create nested test file: %v", err)
	}

	nestedPromptFile := filepath.Join(nestedDir, "nested.yml")
	nestedPromptContent := `prompt:
  - file: "nested.txt"
  - text: "Nested text"`

	err = os.WriteFile(nestedPromptFile, []byte(nestedPromptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create nested prompt file: %v", err)
	}

	mainPromptFile := filepath.Join(tmpDir, "main.yml")
	mainPromptContent := `prompt:
  - text: "Main prompt start"
  - prompt: "nested/nested.yml"
  - text: "Main prompt end"`

	err = os.WriteFile(mainPromptFile, []byte(mainPromptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create main prompt file: %v", err)
	}

	outputFile := filepath.Join(tmpDir, "output.txt")
	err = processPromptFile(mainPromptFile, outputFile, 128000, "xml")
	if err != nil {
		t.Fatalf("processPromptFile failed: %v", err)
	}

	output, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	outputStr := string(output)

	if !strings.Contains(outputStr, "Main prompt start") {
		t.Error("Output should contain main prompt content")
	}
	if !strings.Contains(outputStr, "<!-- pcp-source: nested/nested.yml -->") {
		t.Error("Output should contain nested prompt section header")
	}
	if !strings.Contains(outputStr, "nested/nested.yml->nested.txt") {
		t.Error("Output should contain nested file reference")
	}
	if !strings.Contains(outputStr, "Nested content") {
		t.Error("Output should contain nested file content")
	}
	if !strings.Contains(outputStr, "Main prompt end") {
		t.Error("Output should contain main prompt end")
	}
}

func TestErrorHandling_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	promptFile := filepath.Join(tmpDir, "prompt.yml")
	promptContent := `prompt:
  - file: "nonexistent.txt"`

	err := os.WriteFile(promptFile, []byte(promptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create prompt file: %v", err)
	}

	err = processPromptFile(promptFile, "", 128000, "xml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}

	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("Expected 'file not found' error, got: %v", err)
	}
}

func TestErrorHandling_BinaryFile(t *testing.T) {
	tmpDir := t.TempDir()

	binaryFile := filepath.Join(tmpDir, "binary.bin")
	binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0xFF}
	err := os.WriteFile(binaryFile, binaryData, 0644)
	if err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}

	promptFile := filepath.Join(tmpDir, "prompt.yml")
	promptContent := `prompt:
  - file: "binary.bin"`

	err = os.WriteFile(promptFile, []byte(promptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create prompt file: %v", err)
	}

	err = processPromptFile(promptFile, "", 128000, "xml")
	if err == nil {
		t.Error("Expected error for binary file")
	}

	if !strings.Contains(err.Error(), "cannot process binary file") {
		t.Errorf("Expected 'cannot process binary file' error, got: %v", err)
	}
}

func TestErrorHandling_CircularReference(t *testing.T) {
	tmpDir := t.TempDir()

	promptA := filepath.Join(tmpDir, "a.yml")
	promptB := filepath.Join(tmpDir, "b.yml")

	promptAContent := `prompt:
  - prompt: "b.yml"`
	promptBContent := `prompt:
  - prompt: "a.yml"`

	err := os.WriteFile(promptA, []byte(promptAContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create prompt A: %v", err)
	}

	err = os.WriteFile(promptB, []byte(promptBContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create prompt B: %v", err)
	}

	err = processPromptFile(promptA, "", 128000, "xml")
	if err == nil {
		t.Error("Expected error for circular reference")
	}

	if !strings.Contains(err.Error(), "circular reference") {
		t.Errorf("Expected 'circular reference' error, got: %v", err)
	}
}

func TestErrorHandling_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	promptFile := filepath.Join(tmpDir, "prompt.yml")
	invalidYAML := `prompt:
  - file: "test.txt"
    invalid_structure: true
    text: "This should not be valid"`

	err := os.WriteFile(promptFile, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid YAML file: %v", err)
	}

	err = processPromptFile(promptFile, "", 128000, "xml")
	if err == nil {
		t.Error("Expected error for invalid YAML structure")
	}
}

func TestErrorHandling_CommandFailure(t *testing.T) {
	tmpDir := t.TempDir()

	promptFile := filepath.Join(tmpDir, "prompt.yml")
	promptContent := `prompt:
  - command: "nonexistent_command_that_should_fail"`

	err := os.WriteFile(promptFile, []byte(promptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create prompt file: %v", err)
	}

	err = processPromptFile(promptFile, "", 128000, "xml")
	if err == nil {
		t.Error("Expected error for failed command")
	}

	if !strings.Contains(err.Error(), "command execution failed") {
		t.Errorf("Expected 'command execution failed' error, got: %v", err)
	}
}

func TestErrorHandling_WordLimit(t *testing.T) {
	tmpDir := t.TempDir()

	promptFile := filepath.Join(tmpDir, "prompt.yml")
	longText := strings.Repeat("word ", 100)
	promptContent := `prompt:
  - text: "` + longText + `"`

	err := os.WriteFile(promptFile, []byte(promptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create prompt file: %v", err)
	}

	err = processPromptFile(promptFile, "", 50, "xml")
	if err == nil {
		t.Error("Expected error for word limit exceeded")
	}

	if !strings.Contains(err.Error(), "exceeds maximum word limit") {
		t.Errorf("Expected 'exceeds maximum word limit' error, got: %v", err)
	}
}

func TestCommandExitStatus1Warning(t *testing.T) {
	tmpDir := t.TempDir()

	promptFile := filepath.Join(tmpDir, "prompt.yml")
	promptContent := `prompt:
  - command: "sh -c 'echo output; exit 1'"`

	err := os.WriteFile(promptFile, []byte(promptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create prompt file: %v", err)
	}

	outputFile := filepath.Join(tmpDir, "output.txt")

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err = processPromptFile(promptFile, outputFile, 128000, "xml")

	w.Close()
	os.Stderr = oldStderr

	var stderrOutput bytes.Buffer
	stderrOutput.ReadFrom(r)

	if err != nil {
		t.Errorf("Should not error on exit status 1, got: %v", err)
	}

	stderrStr := stderrOutput.String()
	if !strings.Contains(stderrStr, "Warning") {
		t.Error("Should print warning for exit status 1")
	}

	output, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !strings.Contains(string(output), "output") {
		t.Error("Should still include command output despite exit status 1")
	}
}

func TestSpecialCharacterHandling(t *testing.T) {
	tmpDir := t.TempDir()

	promptFile := filepath.Join(tmpDir, "prompt.yml")
	promptContent := `prompt:
  - text: |
      Line with tab:	<-- tab
      Line with newline:
      Another line
  - text: "Quoted string with\nnewline and\ttab"
  - text: >
      Folded text that will
      be on multiple lines
      but treated as paragraph`

	err := os.WriteFile(promptFile, []byte(promptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create prompt file: %v", err)
	}

	outputFile := filepath.Join(tmpDir, "output.txt")
	err = processPromptFile(promptFile, outputFile, 128000, "xml")
	if err != nil {
		t.Fatalf("processPromptFile failed: %v", err)
	}

	output, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Line with tab:\t") {
		t.Error("Should preserve tab characters")
	}
	if !strings.Contains(outputStr, "Quoted string with\nnewline") {
		t.Error("Should handle escaped newlines in quoted strings")
	}
}

func TestStdoutOutput(t *testing.T) {
	tmpDir := t.TempDir()

	promptFile := filepath.Join(tmpDir, "prompt.yml")
	promptContent := `prompt:
  - text: "Test output"`

	err := os.WriteFile(promptFile, []byte(promptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create prompt file: %v", err)
	}

	cmd := exec.Command("go", "run", ".", "-f", promptFile)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "<!-- pcp-source: text -->") {
		t.Error("Stdout output should contain section header")
	}
	if !strings.Contains(outputStr, "Test output") {
		t.Error("Stdout output should contain text content")
	}
}

func TestCrossplatformCommands(t *testing.T) {
	tmpDir := t.TempDir()

	promptFile := filepath.Join(tmpDir, "prompt.yml")
	var promptContent string

	if os.PathSeparator == '\\' {
		promptContent = `prompt:
  - command: "echo Windows test"`
	} else {
		promptContent = `prompt:
  - command: "echo Unix test"`
	}

	err := os.WriteFile(promptFile, []byte(promptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create prompt file: %v", err)
	}

	outputFile := filepath.Join(tmpDir, "output.txt")
	err = processPromptFile(promptFile, outputFile, 128000, "xml")
	if err != nil {
		t.Fatalf("processPromptFile failed: %v", err)
	}

	output, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "test") {
		t.Error("Should execute cross-platform commands")
	}
}

func TestPerformance_LargeFiles(t *testing.T) {
	tmpDir := t.TempDir()

	largeFile := filepath.Join(tmpDir, "large.txt")
	largeContent := strings.Repeat("This is a line of text in a large file.\n", 10000)
	err := os.WriteFile(largeFile, []byte(largeContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	promptFile := filepath.Join(tmpDir, "prompt.yml")
	promptContent := `prompt:
  - file: "large.txt"`

	err = os.WriteFile(promptFile, []byte(promptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create prompt file: %v", err)
	}

	start := time.Now()
	err = processPromptFile(promptFile, "", 500000, "xml")
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("processPromptFile failed: %v", err)
	}

	if duration > time.Second*5 {
		t.Errorf("Processing took too long: %v", duration)
	}
}

func TestDelimiterStyles(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("Test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	promptFile := filepath.Join(tmpDir, "prompt.yml")
	promptContent := `prompt:
  - file: "test.txt"
  - text: "Test text"`

	err = os.WriteFile(promptFile, []byte(promptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create prompt file: %v", err)
	}

	testCases := []struct {
		style    string
		contains []string
	}{
		{
			style: "xml",
			contains: []string{
				"<!-- pcp-source: test.txt -->",
				"<!-- pcp-source: text -->",
				"Test content",
				"Test text",
			},
		},
		{
			style: "minimal",
			contains: []string{
				"=== PCP SOURCE: test.txt ===",
				"=== PCP SOURCE: text ===",
				"Test content",
				"Test text",
			},
		},
		{
			style: "full",
			contains: []string{
				"BEGIN: test.txt",
				"BEGIN: text",
				"----------------------------------",
				"Test content",
				"Test text",
			},
		},
		{
			style: "none",
			contains: []string{
				"Test content",
				"Test text",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.style, func(t *testing.T) {
			outputFile := filepath.Join(tmpDir, "output_"+tc.style+".txt")
			err = processPromptFile(promptFile, outputFile, 128000, tc.style)
			if err != nil {
				t.Fatalf("processPromptFile failed for style %s: %v", tc.style, err)
			}

			output, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			outputStr := string(output)
			for _, expected := range tc.contains {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Style %s output should contain '%s'", tc.style, expected)
				}
			}

			// For "none" style, ensure no delimiters are present
			if tc.style == "none" {
				forbiddenStrings := []string{
					"<!-- pcp-source:",
					"=== PCP SOURCE:",
					"BEGIN:",
					"----------------------------------",
				}
				for _, forbidden := range forbiddenStrings {
					if strings.Contains(outputStr, forbidden) {
						t.Errorf("Style 'none' should not contain delimiter '%s'", forbidden)
					}
				}
			}
		})
	}
}
