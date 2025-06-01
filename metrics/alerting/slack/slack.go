/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2021, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package slack

import (
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/slack-go/slack"
	"github.ibm.com/alchemy-containers/armada-performance/metrics/alerting/alert"
	"github.ibm.com/alchemy-containers/armada-performance/metrics/alerting/config"
	"github.ibm.com/alchemy-containers/armada-performance/metrics/alerting/jenkins"
)

const slackTokenEnvVar = "ARGONAUTS_ARM_PERF_ALERTS_SLACK_OAUTH_TOKEN"

// Slack emojis
const (
	errorEmoji       = ":error:"
	warningEmoji     = ":warning:"
	zscoreEmoji      = ":ztop:"
	informationEmoji = ":information_source:"
	silencedEmoji    = ":silenced:"
	failedEmoji      = ":failed:"
)

// SendAlerts will send a summary of the supplied alerts to a Slack channel and optionally a user
func SendAlerts(conf *config.Data, failures map[string]*jenkins.FailureData, alerts map[string][]alert.Alert) {
	testEnvs := conf.Environments
	token := os.Getenv(slackTokenEnvVar)
	if len(token) == 0 {
		log.Fatalf("Slack token not provided. Check '%s' environment variable", slackTokenEnvVar)
	}

	api := slack.New(token, slack.OptionDebug(false)) // Careful if setting the debug option to true. It will output the token.

	blocks := make([]slack.Block, 0)

	// Let's sort the map on the environment name to get a consistent ordering in the output
	sortedEnvs := make([]string, len(alerts))
	i := 0
	for n := range alerts {
		sortedEnvs[i] = n
		i++
	}
	sort.Strings(sortedEnvs)

	for _, e := range sortedEnvs {
		envBlocks := make([]slack.Block, 0)

		// Get all the alerts for this environment
		a := alerts[e]

		// Store the number of alerts at the various severity levels
		sevCount := make([]int, 5)
		for _, x := range a {
			sevCount[x.Sev]++
		}

		ownerActionRequired := false

		// Header block containing the environment name
		envHeader := slack.NewHeaderBlock(slack.NewTextBlockObject(slack.PlainTextType, e, false, false))

		totalSlice := make([]*slack.TextBlockObject, 0)

		// Total alerts for the current environment
		countTotalName := slack.NewTextBlockObject(slack.MarkdownType, "*Total:*", false, false)
		totalSlice = append(totalSlice, countTotalName)
		countTotalField := slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*%d*", len(a)), false, false)
		totalSlice = append(totalSlice, countTotalField)
		totalSection := slack.NewSectionBlock(nil, totalSlice, nil)

		// Text Block Fields containg the alert summary information, including the severity data
		fieldSlice := make([]*slack.TextBlockObject, 0)

		// Errors...
		if sevCount[alert.Error] > 0 {
			countErrorFieldName := slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("%s \tError: ", errorEmoji), false, false)
			countErrorFieldValue := slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("%d", sevCount[alert.Error]), false, false)
			fieldSlice = append(fieldSlice, countErrorFieldName)
			fieldSlice = append(fieldSlice, countErrorFieldValue)
			ownerActionRequired = true

		}

		// Warnings...
		if sevCount[alert.Warning] > 0 {
			countWarningFieldName := slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("%s \tWarning: ", warningEmoji), false, false)
			countWarningFieldValue := slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("%d", sevCount[alert.Warning]), false, false)
			fieldSlice = append(fieldSlice, countWarningFieldName)
			fieldSlice = append(fieldSlice, countWarningFieldValue)
			ownerActionRequired = true
		}

		// Historical...
		if sevCount[alert.Zscore] > 0 {
			countZscoreFieldName := slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("%s \tZ-Score: ", zscoreEmoji), false, false)
			countZscoreFieldValue := slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("%d", sevCount[alert.Zscore]), false, false)
			fieldSlice = append(fieldSlice, countZscoreFieldName)
			fieldSlice = append(fieldSlice, countZscoreFieldValue)
			ownerActionRequired = true
		}

		// Informational... (alert thresholds that have been identified as being excessively lenient)
		if sevCount[alert.Information] > 0 {
			countInformationFieldName := slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("%s \tInfo: ", informationEmoji), false, false)
			countInformationFieldValue := slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("%d", sevCount[alert.Information]), false, false)
			fieldSlice = append(fieldSlice, countInformationFieldName)
			fieldSlice = append(fieldSlice, countInformationFieldValue)
			ownerActionRequired = true
		}

		// Silenced... (potential alert silenced by referencing an open issue)
		if sevCount[alert.Silenced] > 0 {
			countSilencedFieldName := slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("%s \tSilenced: ", silencedEmoji), false, false)
			countSilencedFieldValue := slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("%d", sevCount[alert.Silenced]), false, false)
			fieldSlice = append(fieldSlice, countSilencedFieldName)
			fieldSlice = append(fieldSlice, countSilencedFieldValue)
		}

		// Construct section containg all alert details for the current environment
		envBlocks = append(envBlocks, envHeader)
		envBlocks = append(envBlocks, totalSection)
		if len(fieldSlice) > 0 {
			fieldsSection := slack.NewSectionBlock(nil, fieldSlice, nil)
			envBlocks = append(envBlocks, fieldsSection)
		}

		// If the environment owner should be notified, send them a DM containing the alert/failure summary and links, for their environment
		if testEnvs[e].Owner.Notify == config.Always || (testEnvs[e].Owner.Notify == config.WhenFound && len(a) > 0) {
			if config.Contains(testEnvs[e].Owner.Days, time.Now().Weekday()) {
				if conf.Options.Debug {
					fmt.Printf("Owner: %s\n", testEnvs[e].Owner.Name)
				}

				ownerBlocks := make([]slack.Block, 0)
				ownerSlice := make([]*slack.TextBlockObject, 0)

				failureText := ""

				if ownerActionRequired {
					if failures[e].Count > 0 {
						failureText = " and test failures"
					}
					ownerSlice = append(ownerSlice, slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("Hey %s, sorry to be the bearer of bad news. You've got some <%s|automation alerts>%s to investigate.", testEnvs[e].Owner.Name, conf.Slack.ResultsURL, failureText), false, false))
				} else {
					if failures[e].Count > 0 {
						failureText = "There are some test failures to look at though."
					} else {
						failureText = "Might want to look at the open issue(s) instead."
					}
					ownerSlice = append(ownerSlice, slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("Hey %s, good news. You've got no active <%s|automation alerts> to investigate right now. %s.", testEnvs[e].Owner.Name, conf.Slack.ResultsURL, failureText), false, false))
				}

				ownerSection := slack.NewSectionBlock(nil, ownerSlice, nil)
				ownerBlocks = append(ownerBlocks, ownerSection)

				ownerBlocks = append(ownerBlocks, envBlocks...)

				// Handle Jenkins build test automation failures
				if failures[e].Count > 0 {
					ownerBlocks = append(ownerBlocks, slack.NewDividerBlock())

					// Construct summary count
					ownerFailureSummary := make([]*slack.TextBlockObject, 0)
					failureTotalName := slack.NewTextBlockObject(slack.MarkdownType, "*Test Failures:*", false, false)
					ownerFailureSummary = append(ownerFailureSummary, failureTotalName)
					failureTotalField := slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*%d*", failures[e].Count), false, false)
					ownerFailureSummary = append(ownerFailureSummary, failureTotalField)
					failureSectionSummary := slack.NewSectionBlock(nil, ownerFailureSummary, nil)
					ownerBlocks = append(ownerBlocks, failureSectionSummary)

					var failureSection *slack.SectionBlock
					ownerFailures := make([]*slack.TextBlockObject, 0)

					for i, tf := range failures[e].Tests {
						d := time.Unix(tf.Timestamp/1000, 0).Format("02 Jan")
						ownerFailures = append(ownerFailures, slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("<%s|Failure %d:> %s - %s (%s)", tf.URL, i+1, tf.Name, tf.KubeVersion, d), false, false))

						// Maximum number of fields in a SectionBlock is 10 - https://api.slack.com/reference/block-kit/blocks#section
						if i > 0 && (i+1)%10 == 0 {
							failureSection = slack.NewSectionBlock(nil, ownerFailures, nil)
							ownerBlocks = append(ownerBlocks, failureSection)
							ownerFailures = make([]*slack.TextBlockObject, 0)
						}
					}

					if len(ownerFailures) > 0 {
						failureSection = slack.NewSectionBlock(nil, ownerFailures, nil)
						ownerBlocks = append(ownerBlocks, failureSection)
					}

					if conf.Options.Debug {
						fmt.Printf("\tOwner Failures: %d\n", len(ownerFailures))
					}
				}

				if conf.Options.Debug {
					fmt.Printf("\tOwner Blocks: %d\n", len(ownerBlocks))
				}

				_, _, err := api.PostMessage(
					testEnvs[e].Owner.Slack,
					slack.MsgOptionText("Armada Performance Alert", false),
					slack.MsgOptionBlocks(ownerBlocks...),
					slack.MsgOptionAsUser(false), // Add this if you want that the bot would post message as a user, otherwise it will send response using the default slackbot
				)
				if err != nil {
					log.Fatalf("Error alerting owner : %s\n", err)
				}
			}
		}
		envBlocks = append(envBlocks, slack.NewDividerBlock())
		blocks = append(blocks, envBlocks...)
	}

	detailsField := slack.NewTextBlockObject(slack.PlainTextType, "Alert Details", false, false)

	// "Details" button providing a link to Jenkins job which contains the detailed alert output
	// N.B. There is an annoying behaviour (a warning is displayed) when trying to use a button to provide a simple link
	// See https://github.com/slackapi/node-slack-sdk/issues/869 for details
	detailsButtonElement := slack.NewButtonBlockElement("", "Details", detailsField)
	detailsButtonElement.URL = conf.Slack.ResultsURL
	detailsAction := slack.NewActionBlock("actionblock", detailsButtonElement)
	blocks = append(blocks, detailsAction)

	channelID, timestamp, err := api.PostMessage(
		conf.Slack.Channel,
		slack.MsgOptionText("Armada Performance Alert", false),
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionAsUser(false), // Add this if you want that the bot would post message as a user, otherwise it will send response using the default slackbot
	)
	if err != nil {
		log.Fatalf("Error sending alerts to slack : %s\n", err)
	}

	log.Printf("Message successfully sent to Slack channel %s at %s\n", channelID, timestamp)
}
