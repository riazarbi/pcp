package main

import (
	"path/filepath"
)

type OperationType int

const (
	FileOp OperationType = iota
	PromptOp
	CommandOp
	TextOp
)

type PromptFile struct {
	Prompt []Operation `yaml:"prompt"`
}

type Operation struct {
	File    *string `yaml:"file,omitempty"`
	Prompt  *string `yaml:"prompt,omitempty"`
	Command *string `yaml:"command,omitempty"`
	Text    *string `yaml:"text,omitempty"`
}

func (op *Operation) GetType() (OperationType, error) {
	count := 0
	var opType OperationType

	if op.File != nil {
		count++
		opType = FileOp
	}
	if op.Prompt != nil {
		count++
		opType = PromptOp
	}
	if op.Command != nil {
		count++
		opType = CommandOp
	}
	if op.Text != nil {
		count++
		opType = TextOp
	}

	if count == 0 {
		return 0, ErrOperationEmpty
	}
	if count > 1 {
		return 0, ErrOperationMultiple
	}

	return opType, nil
}

func (op *Operation) GetValue() string {
	switch {
	case op.File != nil:
		return *op.File
	case op.Prompt != nil:
		return *op.Prompt
	case op.Command != nil:
		return *op.Command
	case op.Text != nil:
		return *op.Text
	default:
		return ""
	}
}

type ContentSection struct {
	Source  string
	Content string
	Type    OperationType
}

type CompiledContent struct {
	Sections []ContentSection
}

type ProcessingContext struct {
	basePath       string
	visitedFiles   map[string]bool
	maxWords       int
	wordCount      int
	delimiterStyle string
}

func NewProcessingContext(basePath string, maxWords int, delimiterStyle string) *ProcessingContext {
	return &ProcessingContext{
		basePath:       filepath.Dir(basePath),
		visitedFiles:   make(map[string]bool),
		maxWords:       maxWords,
		wordCount:      0,
		delimiterStyle: delimiterStyle,
	}
}

func (ctx *ProcessingContext) MarkVisited(path string) {
	absPath, _ := filepath.Abs(path)
	ctx.visitedFiles[absPath] = true
}

func (ctx *ProcessingContext) IsVisited(path string) bool {
	absPath, _ := filepath.Abs(path)
	return ctx.visitedFiles[absPath]
}

func (ctx *ProcessingContext) ResolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(ctx.basePath, path)
}

func (ctx *ProcessingContext) AddWords(count int) error {
	ctx.wordCount += count
	if ctx.wordCount > ctx.maxWords {
		return ErrWordLimitExceeded{Current: ctx.wordCount, Limit: ctx.maxWords}
	}
	return nil
}
