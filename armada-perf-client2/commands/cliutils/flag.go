
package cliutils

import (
	"fmt"
	"strings"

	"github.com/urfave/cli"
)

// sentinel type for usage failure if absent
type failIfAbsent struct{}

// UsageFailureIfAbsent passed as default value to GetRequiredFlagString if there should be a usage failure if the required string is absent.
var UsageFailureIfAbsent interface{} = failIfAbsent{}

// GetFlagString returns the string value of the provided flag in the cli context's command args. returns the provided default
// if the flag wasn't used.  returns a usage error if the provided default is nil or UsageFailureIfAbsent.
func GetFlagString(c *cli.Context, flagName string, optional bool, description string, defaultVal interface{}) (string, error) {
	var val string

	mdv := GetMetadataValue(c, flagName)
	if mdv != nil {
		val = mdv.(string)
		return strings.TrimSpace(val), nil
	}

	val = c.String(flagName)
	if val != "" {
		return strings.TrimSpace(val), nil
	}

	val = c.Parent().String(flagName)
	if val != "" {
		return strings.TrimSpace(val), nil
	}

	if defaultVal != nil && defaultVal != UsageFailureIfAbsent {
		val = defaultVal.(string)
		if len(val) > 0 {
			return defaultVal.(string), nil
		}
	}

	var err error

	if !optional {
		err = IncorrectUsageError(c, fmt.Sprintf(MissingParameterMsg, flagName, description))
	}
	return "", err
}

// GetFlagInt returns the integer value of the provided flag in the cli context's command args. returns the provided default
// if the flag wasn't used.  returns a usage error if the provided default is nil or UsageFailureIfAbsent.
func GetFlagInt(c *cli.Context, flagName string, optional bool, description string, defaultVal interface{}) (int, error) {
	val := c.Int(flagName)
	if val != 0 {
		return val, nil
	}

	val = c.Parent().Int(flagName)
	if val != 0 {
		return val, nil
	}

	if defaultVal != nil && defaultVal != UsageFailureIfAbsent {
		return defaultVal.(int), nil
	}

	var err error

	if !optional {
		err = IncorrectUsageError(c, fmt.Sprintf(MissingParameterMsg, flagName, description))
	}
	return 0, err
}

// GetFlagBool returns the boolean value of the provided flag, returns false if not found
func GetFlagBool(c *cli.Context, flagName string) bool {
	val := c.Bool(flagName)
	if !val {
		val = c.Parent().Bool(flagName)
	}

	return val
}

// GetFlagBoolT returns the boolean value of the provided flag, returns true if not found
func GetFlagBoolT(c *cli.Context, flagName string) bool {
	val := c.BoolT(flagName)
	if !val {
		val = c.Parent().BoolT(flagName)
	}
	return val
}

// GetMetadataValue returns the valuefrom the Application's metadata map for the given key
func GetMetadataValue(c *cli.Context, k string) interface{} {
	md := c.App.Metadata[k]
	if md == nil {
		md = c.Parent().App.Metadata[k]
	}
	return md
}

// GetCommandName returns the full command name
func GetCommandName(c *cli.Context) string {
	name := c.Command.FullName()
	if len(name) == 0 {
		name = c.Parent().Command.FullName()
	}

	return name
}
