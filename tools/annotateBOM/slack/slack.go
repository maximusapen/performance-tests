/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package slack

import (
	"fmt"
	"os"
	"time"

	metrics "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"

	"github.com/slack-go/slack"
)

const slackTokenEnvVar = "ARGONAUTS_ARM_PERF_ALERTS_SLACK_OAUTH_TOKEN"
const channelID = "G5CNHCJ7R" // armada-perf-private

// WriteBOMUpdate is used to send BOM update events to Slack
func WriteBOMUpdate(carrierName string, currentBOM string, t metrics.BOMType, timestamp time.Time) error {
	token := os.Getenv(slackTokenEnvVar)
	if len(token) == 0 {
		return fmt.Errorf("Slack token not provided. Check '%s' environment variable", slackTokenEnvVar)
	}
	api := slack.New(token, slack.OptionDebug(false)) // Careful if setting the debug option to true. It will output the token.

	attachment := slack.Attachment{
		Color:   t.Color(),
		Pretext: carrierName,
		Fields: []slack.AttachmentField{
			{
				Title: t.String(),
				Value: currentBOM,
			},
		},
	}

	channelID, timestampStr, err := api.PostMessage(
		channelID,
		slack.MsgOptionText("BOM Update", false),
		slack.MsgOptionAttachments(attachment),
		slack.MsgOptionAsUser(true), // Add this if you want that the bot would post message as a user, otherwise it will send response using the default slackbot
	)
	if err != nil {
		return fmt.Errorf("Failed to send Slack %s BOM Update - %w", t.String(), err)
	}
	fmt.Printf("Message successfully sent to channel %s at %s", channelID, timestampStr)
	return nil
}
