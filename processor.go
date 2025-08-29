package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func processOperation(op Operation, ctx *ProcessingContext) (ContentSection, error) {
	opType, err := op.GetType()
	if err != nil {
		return ContentSection{}, err
	}

	value := op.GetValue()

	switch opType {
	case FileOp:
		return processFileOperation(value, ctx)
	case PromptOp:
		return processPromptOperation(value, ctx)
	case CommandOp:
		return processCommandOperation(value, ctx)
	case TextOp:
		return processTextOperation(value, ctx)
	default:
		return ContentSection{}, fmt.Errorf("unknown operation type")
	}
}

func processFileOperation(filePath string, ctx *ProcessingContext) (ContentSection, error) {
	resolvedPath := ctx.ResolvePath(filePath)

	if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
		return ContentSection{}, ErrFileNotFound{File: resolvedPath}
	}

	if isBinaryFile(resolvedPath) {
		return ContentSection{}, ErrBinaryFile{File: resolvedPath}
	}

	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		return ContentSection{}, fmt.Errorf("failed to read file %s: %w", resolvedPath, err)
	}

	contentStr := string(content)
	wordCount := countWords(contentStr)
	if err := ctx.AddWords(wordCount); err != nil {
		return ContentSection{}, err
	}

	return ContentSection{
		Source:  filePath,
		Content: contentStr,
		Type:    FileOp,
	}, nil
}

func processPromptOperation(promptPath string, ctx *ProcessingContext) (ContentSection, error) {
	resolvedPath := ctx.ResolvePath(promptPath)

	if ctx.IsVisited(resolvedPath) {
		return ContentSection{}, ErrCircularReference{File: resolvedPath, Path: getVisitedPaths(ctx)}
	}

	pf, err := parsePromptFile(resolvedPath)
	if err != nil {
		return ContentSection{}, err
	}

	oldBasePath := ctx.basePath
	ctx.basePath = filepath.Dir(resolvedPath)
	ctx.MarkVisited(resolvedPath)

	var allSections []ContentSection
	for _, op := range pf.Prompt {
		section, err := processOperation(op, ctx)
		if err != nil {
			return ContentSection{}, err
		}
		allSections = append(allSections, section)
	}

	delete(ctx.visitedFiles, resolvedPath)
	ctx.basePath = oldBasePath

	var combinedContent strings.Builder
	for i, section := range allSections {
		if i > 0 {
			combinedContent.WriteString("\n")
		}
		combinedContent.WriteString(formatSectionHeader(promptPath+"->"+section.Source, ctx.delimiterStyle))
		combinedContent.WriteString(section.Content)
	}

	wordCount := countWords(combinedContent.String())
	if err := ctx.AddWords(wordCount); err != nil {
		return ContentSection{}, err
	}

	return ContentSection{
		Source:  promptPath,
		Content: combinedContent.String(),
		Type:    PromptOp,
	}, nil
}

func processCommandOperation(command string, ctx *ProcessingContext) (ContentSection, error) {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()

	outputStr := string(output)

	if err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
			fmt.Fprintf(os.Stderr, "Warning: command '%s' exited with status 1 but continuing processing\n", command)
		} else {
			return ContentSection{}, ErrCommandFailed{Command: command, Err: err}
		}
	}

	wordCount := countWords(outputStr)
	if err := ctx.AddWords(wordCount); err != nil {
		return ContentSection{}, err
	}

	return ContentSection{
		Source:  command,
		Content: outputStr,
		Type:    CommandOp,
	}, nil
}

func processTextOperation(text string, ctx *ProcessingContext) (ContentSection, error) {
	wordCount := countWords(text)
	if err := ctx.AddWords(wordCount); err != nil {
		return ContentSection{}, err
	}

	return ContentSection{
		Source:  "text",
		Content: text,
		Type:    TextOp,
	}, nil
}

func formatSectionHeader(source, delimiterStyle string) string {
	switch delimiterStyle {
	case "xml":
		return fmt.Sprintf("\n<!-- pcp-source: %s -->\n", source)
	case "minimal":
		return fmt.Sprintf("\n=== PCP SOURCE: %s ===\n", source)
	case "none":
		return "\n" // Still add separation between sections
	case "full":
		return fmt.Sprintf("\n----------------------------------\nBEGIN: %s\n----------------------------------\n", source)
	default:
		// Default to xml style for unknown styles
		return fmt.Sprintf("\n<!-- pcp-source: %s -->\n", source)
	}
}

func countWords(text string) int {
	if text == "" {
		return 0
	}
	return len(strings.Fields(text))
}
