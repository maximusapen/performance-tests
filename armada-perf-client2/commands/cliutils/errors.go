
package cliutils

import (
	"fmt"

	"github.com/urfave/cli"
)

// UsageError indicates a CLI command usage failure and should be followed up with printing the current command's usage information
type UsageError struct {
	message string
	Usage   func() error // calling Usage() shows usage information for the current command
}

// NewUsageError creates a UsageError that prints usage info with the given func
func NewUsageError(message string, usage func() error) error {
	if usage == nil {
		// can't panic here since this is supposed to create a useful error message
		return fmt.Errorf("Error generating usage\n%s", message)
	}
	return UsageError{
		message: message,
		Usage:   usage,
	}
}

func (u UsageError) Error() string {
	return u.message
}

// IncorrectUsageError prints the current command usage and the provided usage hint
func IncorrectUsageError(c *cli.Context, message string) error {
	usage := func() error {
		return cli.ShowCommandHelp(c, c.Command.Name)
	}
	hint := "Incorrect Usage."
	if message != "" {
		hint += " " + message
	}
	return NewUsageError(hint, usage)
}

// IncorrectUsageGenericError prints the current command usage and a generic "incorrect usage" message
func IncorrectUsageGenericError(c *cli.Context) error {
	return IncorrectUsageError(c, "")
}
