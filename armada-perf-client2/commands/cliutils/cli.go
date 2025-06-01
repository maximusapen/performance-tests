
package cliutils

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
)

// Messages that are used multiple times should be defined as constants here
const (
	InvalidValueMsg        = "An invalid value was specified for --%s: %s"
	KeyValuePairInvalidMsg = "The following entry is not a valid key value pair: '%s'. Use '%s' as the format to specify key value pairs."
	MissingParameterMsg    = "The '--%s <%s>' flag is required."
	PermittedValuesMsg     = "Permitted values for --%s: %s"
)

// WriteJSON writes the given struct out in JSON format
func WriteJSON(c *cli.Context, out interface{}) {
	contents, err := json.MarshalIndent(out, "", "    ")
	if err != nil {
		fmt.Fprintf(c.App.Writer, "Unable to unmarshal JSON response : %s\n", err.Error())
	}
	//_, err = ui.Writer().Write(contents)
	_, err = c.App.Writer.Write(contents)
	if err != nil {
		panic(err)
	}
}

// MakeLabelsMap constructs a map of labels from a list of name/value pairs
func MakeLabelsMap(c *cli.Context) (map[string]string, error) {
	labelsMap := make(map[string]string)
	labels := c.StringSlice(models.LabelFlagName)

	for _, label := range labels {
		labelTokens := strings.SplitN(label, "=", 2)
		if len(labelTokens) != 2 {
			invalidValue := fmt.Sprintf(InvalidValueMsg, models.LabelFlagName, labels)
			return nil, IncorrectUsageError(c, invalidValue)
		}
		key, value := labelTokens[0], labelTokens[1]
		labelsMap[key] = value
	}

	return labelsMap, nil
}

// MakeHostLabelsMap creates a host label map from CLI context
func MakeHostLabelsMap(c *cli.Context) (map[string]string, error) {
	labelsMap := make(map[string]string)
	labels := c.StringSlice(models.HostLabelFlagName)

	for _, label := range labels {
		labelTokens := strings.SplitN(label, "=", 2)
		if len(labelTokens) != 2 {
			invalidValue := fmt.Sprintf(InvalidValueMsg, models.LabelFlagName, labels)
			return nil, IncorrectUsageError(c, invalidValue)

		}
		key, value := labelTokens[0], labelTokens[1]
		labelsMap[key] = value
	}
	return labelsMap, nil
}
