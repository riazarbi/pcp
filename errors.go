package main

import "fmt"

var (
	ErrOperationEmpty    = fmt.Errorf("operation must specify exactly one of: file, prompt, command, text")
	ErrOperationMultiple = fmt.Errorf("operation must specify exactly one of: file, prompt, command, text")
)

type ErrInvalidYAML struct {
	File string
	Err  error
}

func (e ErrInvalidYAML) Error() string {
	return fmt.Sprintf("invalid YAML in file %s: %v", e.File, e.Err)
}

type ErrFileNotFound struct {
	File string
}

func (e ErrFileNotFound) Error() string {
	return fmt.Sprintf("file not found: %s", e.File)
}

type ErrBinaryFile struct {
	File string
}

func (e ErrBinaryFile) Error() string {
	return fmt.Sprintf("cannot process binary file: %s", e.File)
}

type ErrCircularReference struct {
	File string
	Path []string
}

func (e ErrCircularReference) Error() string {
	return fmt.Sprintf("circular reference detected in file %s (reference path: %v)", e.File, e.Path)
}

type ErrCommandFailed struct {
	Command string
	Err     error
}

func (e ErrCommandFailed) Error() string {
	return fmt.Sprintf("command execution failed: %s (%v)", e.Command, e.Err)
}

type ErrWordLimitExceeded struct {
	Current int
	Limit   int
}

func (e ErrWordLimitExceeded) Error() string {
	return fmt.Sprintf("compiled output (%d words) exceeds maximum word limit (%d words)", e.Current, e.Limit)
}
