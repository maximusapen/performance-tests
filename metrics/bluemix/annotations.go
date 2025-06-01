/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package metricsservice

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	client "github.com/influxdata/influxdb1-client/v2"
	utils "github.ibm.com/alchemy-containers/armada-performance/tools/crypto/utils"
)

const grafanaAnnotationTableName = "bomEvents"

const (
	// Master : Master BOM
	Master BOMType = iota
	// Worker : Worker BOM
	Worker
)

var (
	bomTypesMap = map[string]BOMType{
		"master": Master,
		"worker": Worker,
	}
)

// BOMType is an enumeration of Armada BOM types (Master or Worker)
type BOMType int

func (b *BOMType) String() string {
	switch *b {
	case Master:
		return "Master"
	case Worker:
		return "Worker"
	}
	return ""
}

// Color defines slack colouring for each BOM type
func (b *BOMType) Color() string {
	switch *b {
	case Master:
		return "#E01E5A" // Slack Red
	case Worker:
		return "#2EB67D" // Slack Green
	}
	return ""
}

// ParseBOMTypeStr converts a string representation to a BOMType
func ParseBOMTypeStr(s string) (BOMType, bool) {
	t, ok := bomTypesMap[strings.ToLower(s)]
	return t, ok
}

// WriteGrafanaBOMAnnotations stores data associated with Grafana annotation(s) in InfluxDB
func WriteGrafanaBOMAnnotations(carrierName string, currentBOM string, t BOMType, timestamp time.Time) (bool, error) {
	var metricsCfg ServiceConfig
	ok := true

	if metricsCfg, ok = ReadMetricsTomlFile(); !ok {
		return false, errors.New("Unable to read metrics configuration file")
	}

	dbKey := os.Getenv("METRICS_DB_KEY")

	if len(dbKey) == 0 {
		// May need to read the encrypted key from metrics.toml (e.g cruiser_churn)

		// Need to decrypt the key - but only if the encryption key is set
		// The encryption key needs to be setup by the caller
		// If it isn't we will return an error
		encryptionKey := os.Getenv(utils.KeyEnvVar)
		if len(encryptionKey) > 0 {
			var err error
			dbKey, err = utils.Decrypt(metricsCfg.Metrics.InfluxdbPassword)
			if err != nil {
				return false, fmt.Errorf("Unable to decrypt Influx DB_KEY from merics.toml. Error %w", err)
			}
		} else {
			return false, fmt.Errorf("METRICS_DB_KEY is not set, and no encryption key is set, so unable to determine Key for InfluxDB")
		}
	}

	// Get the Kube Major.minor version.
	// N.B. We use major_minor version in Grafana dashboards, so join major/minor with "_" rather than "."
	kubeMajorMinorVersion := strings.Join(strings.Split(currentBOM, ".")[0:2], "_")

	httpClient, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     "http://" + net.JoinHostPort(metricsCfg.Metrics.InfluxdbHost, metricsCfg.Metrics.InfluxdbPort),
		Username: metricsCfg.Metrics.InfluxdbUser,
		Password: dbKey, // pragma: allowlist secret
		Timeout:  300 * time.Second,
	})
	if err != nil {
		return false, fmt.Errorf("failed to create Influxdb http client. Unable to write Grafana annotation data. Error %w", err)
	}
	defer httpClient.Close()

	// Get the last known master or worker BOM
	previousBOM := ""
	cmd := fmt.Sprintf("SELECT last(bom) FROM %s where resourceType='%s' and kubeVersion='%s' and carrierName='%s'", grafanaAnnotationTableName, t.String(), kubeMajorMinorVersion, carrierName)
	if metricsCfg.Metrics.Verbose {
		log.Printf("Querying influx : %s", cmd)
	}

	q := client.NewQuery(cmd, metricsCfg.Metrics.InfluxdbName, "")
	if response, err := httpClient.Query(q); err == nil && response.Error() == nil {
		if len(response.Results[0].Series) > 0 {
			inflxuResults := response.Results[0].Series[0].Values

			for _, v := range inflxuResults {
				previousBOM = v[1].(string)
			}
		}
	} else {
		if err != nil {
			return false, fmt.Errorf("error running InfluxDB query: %w", err)
		}
	}

	if metricsCfg.Metrics.Verbose {
		log.Printf("Previous BOM : %s, Current BOM: %s", previousBOM, currentBOM)
	}

	// New BOM? If so, add it to influx
	annotationWritten := false
	if previousBOM != currentBOM {
		log.Printf("New %s BOM: %s\n", t.String(), currentBOM)
		if timestamp.Unix() == 0 {
			timestamp = time.Now()
		}
		bp, err := client.NewBatchPoints(client.BatchPointsConfig{
			Database:  metricsCfg.Metrics.InfluxdbName,
			Precision: "s",
		})
		if err != nil {
			return false, fmt.Errorf("failed to create Influxdb batch point. Unable to write Grafana annotation data. Error %w", err)
		}

		tags := make(map[string]string)
		tags["carrierName"] = carrierName
		tags["kubeVersion"] = kubeMajorMinorVersion

		fields := make(map[string]interface{})
		fields["title"] = fmt.Sprintf("BOM Update - %s", carrierName)
		fields["bom"] = currentBOM
		fields["resourceType"] = t.String()

		pt, err := client.NewPoint(grafanaAnnotationTableName, tags, fields, timestamp)
		if err != nil {
			return false, fmt.Errorf("error creating Influxdb data point: %w", err)
		}
		bp.AddPoint(pt)

		if metricsCfg.Metrics.Verbose {
			log.Println(pt)
		}

		// write the batch of data to Influxdb
		if err := httpClient.Write(bp); err != nil {
			return false, fmt.Errorf("failed to write Grafana annotation(s) to Influxdb: %w", err)
		}
		log.Println("Grafana annotations successfully written to influxdb")
		annotationWritten = true
	}

	return annotationWritten, nil
}
