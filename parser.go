package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func parsePromptFile(filePath string) (*PromptFile, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, ErrFileNotFound{File: filePath}
	}

	var promptFile PromptFile
	if err := yaml.Unmarshal(data, &promptFile); err != nil {
		return nil, ErrInvalidYAML{File: filePath, Err: err}
	}

	if err := validatePromptFile(&promptFile); err != nil {
		return nil, fmt.Errorf("validation failed for %s: %w", filePath, err)
	}

	return &promptFile, nil
}

func validatePromptFile(pf *PromptFile) error {
	if pf.Prompt == nil {
		return fmt.Errorf("missing required 'prompt' key")
	}

	for i, op := range pf.Prompt {
		if _, err := op.GetType(); err != nil {
			return fmt.Errorf("operation %d: %w", i, err)
		}
	}

	return nil
}

func isBinaryFile(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return false
	}

	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return true
		}
	}

	return false
}

func validatePromptFileStructure(filePath string, ctx *ProcessingContext) error {
	absPath, _ := filepath.Abs(filePath)

	if ctx.IsVisited(absPath) {
		return ErrCircularReference{File: filePath, Path: getVisitedPaths(ctx)}
	}

	ctx.MarkVisited(absPath)
	defer func() {
		delete(ctx.visitedFiles, absPath)
	}()

	pf, err := parsePromptFile(filePath)
	if err != nil {
		return err
	}

	for _, op := range pf.Prompt {
		opType, _ := op.GetType()
		if opType == PromptOp {
			nestedPath := ctx.ResolvePath(op.GetValue())
			if err := validatePromptFileStructure(nestedPath, ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

func getVisitedPaths(ctx *ProcessingContext) []string {
	paths := make([]string, 0, len(ctx.visitedFiles))
	for path := range ctx.visitedFiles {
		paths = append(paths, path)
	}
	return paths
}
