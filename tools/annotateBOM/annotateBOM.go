/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

import (
	"flag"
	"log"
	"time"

	metrics "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"
	"github.ibm.com/alchemy-containers/armada-performance/tools/annotateBOM/slack"
)

func main() {
	var err error

	carrierName := flag.String("carrier", "", "Name of carrier, e.g. carrier4_stage")
	bomVersion := flag.String("bomVersion", "", "Full BOM version, e.g. 1.25.6_1329")
	bomTypeStr := flag.String("bomType", "", "BOM type, i.e. Master or Worker")
	timestampStr := flag.String("timestamp", "", "[Optional] Event timestamp; default: now")

	slackEnabled := flag.Bool("slack", false, "[Optional] Slack notifications; default: false")

	flag.Parse()

	bomType, ok := metrics.ParseBOMTypeStr(*bomTypeStr)
	if !ok {
		log.Fatalf("Invalid bomType specified '%s'. Must be 'Master' or 'Worker'\n", *bomTypeStr)
	}

	timestamp := time.Now()
	if *timestampStr != "" {
		timestamp, err = time.Parse(time.RFC3339, *timestampStr)
		if err != nil {
			log.Fatalf("Invalid timestamp specified '%s'. Must be RFC3339 format, e.g. '%s'", *timestampStr, time.RFC3339)
		}

	}

	update, err := metrics.WriteGrafanaBOMAnnotations(*carrierName, *bomVersion, bomType, timestamp)
	if err != nil {
		log.Fatalf("Failed to write BOM Update annotation - %s", err)
	}

	if update && *slackEnabled {
		err = slack.WriteBOMUpdate(*carrierName, *bomVersion, bomType, timestamp)
		if err != nil {
			log.Fatalf("BOM Slack Update Failed - %s", err)
		}
	}
}
