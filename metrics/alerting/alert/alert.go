/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2021, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package alert

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.ibm.com/alchemy-containers/armada-performance/metrics/alerting/config"
	influxdata "github.ibm.com/alchemy-containers/armada-performance/metrics/alerting/influx"
)

// A Severity specifies the alert severity (Warning = 0, ...)
type Severity int

const (
	// Information highlights alerts that should be considered too lenient
	Information Severity = iota
	// Warning highlights simple threshold alerts that should be considered as a warning
	Warning
	// Error highlights simple threshold alerts that should be considered as an error
	Error
	// Zscore highlights alerts that are based on historical data
	Zscore
	// Silenced highlights potential alerts that have been silenced by referencing an open issue
	Silenced
)

func (s Severity) String() string {
	switch s {
	case Information:
		return "INFORMATION"
	case Warning:
		return "WARNING"
	case Error:
		return "ERROR"
	case Zscore:
		return "Z-SCORE"
	}
	return "UNKNOWN"
}

// Alert holds details of a test result alert
type Alert struct {
	Name                 string
	EnvName              string
	Carrier              string
	KubeVersion          string
	MachineType          string
	OperatingSystem      string
	Owner                config.Owner
	Sev                  Severity
	Alert                config.Alert
	Timestamp            time.Time
	InfoWarningThreshold float64
	LeniencyThreshold    float64
	Result               float64
}

func displayAlert(a Alert) {
	var threshold float64
	var colour string
	var indent string

	switch a.Sev {
	case Error:
		threshold = a.Alert.Thresholds[a.MachineType].Error
		colour = config.ColourRed
	case Warning:
		threshold = a.Alert.Thresholds[a.MachineType].Warn
		colour = config.ColourYellow
	case Information:
		threshold = a.LeniencyThreshold
		colour = config.ColourGreen
	case Zscore:
		threshold = a.Alert.Thresholds[a.MachineType].Zscore
		colour = config.ColourMagenta
	}

	if config.ConfigData.Options.Verbose {
		indent = "\t\t"
	}

	fmt.Printf("%s", colour)
	fmt.Printf("%sALERT - %s\n", indent, a.Sev)
	fmt.Printf("%s\tOwner: %s\n", indent, a.Owner.Name)

	fmt.Printf("%s\tEnvironment: %s", indent, a.Carrier)
	if len(a.MachineType) > 0 {
		fmt.Printf(", MachineType: %s", a.MachineType)
	}
	if len(a.KubeVersion) > 0 {
		fmt.Printf(", Version: %s", strings.ReplaceAll(a.KubeVersion, "_", "."))
	}
	if len(a.OperatingSystem) > 0 {
		fmt.Printf(", OS: %s", a.OperatingSystem)
	}

	fmt.Printf("\n%s\tTest: %s - %s\n", indent, a.Name, a.Alert.Name)
	fmt.Printf("%s\tTimestamp: %s, Threshold: %.6g, Result: %.6g\n", indent, a.Timestamp, threshold, a.Result)
	if a.Sev == Information {
		fmt.Printf("%s\tWarningThreshold: %.6g\n", indent, a.InfoWarningThreshold)
	}
	fmt.Printf("%s\n", config.ColourReset)
}

// ProcessData will generate alerts through the comparison of configured alert data with test results from Influx
func (a Alert) ProcessData(ac config.Alert, results influxdata.TestResults) []Alert {
	var alerts []Alert

	totalCount := len(results.Current) + len(results.Historical)
	historicalCount := len(results.Historical)

	// Get the value for the latest data point(s) - store the min and max values
	ctrMin := math.MaxFloat64
	ctrMax := 0.0
	for _, cr := range results.Current {
		ctrMin = math.Min(ctrMin, cr.Val)
		ctrMax = math.Max(ctrMax, cr.Val)
	}

	leniencyTotal := 0.0
	historicalTotal := 0.0
	for _, cr := range results.Current {
		leniencyTotal += cr.Val
	}
	for _, hr := range results.Historical {
		leniencyTotal += hr.Val
		historicalTotal += hr.Val
	}

	leniencyMean := leniencyTotal / float64(totalCount)
	historicalMean := historicalTotal / float64(historicalCount)

	leniencySD := 0.0
	historicalSD := 0.0
	for _, r := range results.Current {
		leniencySD += math.Pow(r.Val-leniencyMean, 2)
	}
	for _, r := range results.Historical {
		historicalSD += math.Pow(r.Val-historicalMean, 2)
		leniencySD += math.Pow(r.Val-leniencyMean, 2)
	}

	leniencySD = math.Sqrt(leniencySD / float64(totalCount))
	historicalSD = math.Sqrt(historicalSD / float64(historicalCount))

	if leniencySD > 0 && (totalCount > config.ConfigData.Options.History.Minimum) {
		leniencyZS := (ac.Thresholds[a.MachineType].Warn - leniencyMean) / leniencySD
		alt := Alert{
			Name:              a.Name,
			EnvName:           a.EnvName,
			Carrier:           a.Carrier,
			Owner:             a.Owner,
			KubeVersion:       a.KubeVersion,
			MachineType:       a.MachineType,
			OperatingSystem:   a.OperatingSystem,
			Alert:             ac,
			Timestamp:         time.Unix(results.Current[0].Timestamp, 0),
			Sev:               Information,
			Result:            leniencyZS,
			LeniencyThreshold: a.LeniencyThreshold,
		}

		switch ac.LimitType {
		case "floor":
			if leniencyZS < 0 && math.Abs(leniencyZS) > a.LeniencyThreshold {
				alt.InfoWarningThreshold = leniencyMean - a.LeniencyThreshold*leniencySD
				displayAlert(alt)
				alerts = append(alerts, alt)
			}
		case "ceiling":
			if leniencyZS > 0 && math.Abs(leniencyZS) > a.LeniencyThreshold {
				alt.InfoWarningThreshold = leniencyMean + a.LeniencyThreshold*leniencySD
				displayAlert(alt)
				alerts = append(alerts, alt)
			}
		}
	}

	// Check for Z-Score based alert
	if ac.Thresholds[a.MachineType].Zscore > 0 {
		if historicalSD > 0 && (historicalCount > config.ConfigData.Options.History.Minimum) {

			alt := Alert{
				Name:            a.Name,
				EnvName:         a.EnvName,
				Carrier:         a.Carrier,
				Owner:           a.Owner,
				KubeVersion:     a.KubeVersion,
				MachineType:     a.MachineType,
				OperatingSystem: a.OperatingSystem,
				Alert:           ac,
				Timestamp:       time.Unix(results.Current[0].Timestamp, 0)}

			switch ac.LimitType {
			case "floor":
				historicalZS := (ctrMax - historicalMean) / historicalSD
				if historicalZS < 0 && math.Abs(historicalZS) > ac.Thresholds[a.MachineType].Zscore {
					alt.Result = historicalZS
					alt.Sev = Zscore
					displayAlert(alt)
					alerts = append(alerts, alt)
				}
			case "ceiling":
				historicalZS := (ctrMin - historicalMean) / historicalSD
				if historicalZS > 0 && historicalZS > ac.Thresholds[a.MachineType].Zscore {
					alt.Result = historicalZS
					alt.Sev = Zscore
					displayAlert(alt)
					alerts = append(alerts, alt)
				}
			}
		}
	}

	// Check for simple threshold based alert
	if ac.Thresholds[a.MachineType].Warn > 0 || ac.Thresholds[a.MachineType].Error > 0 {
		alt := Alert{
			Name:            a.Name,
			EnvName:         a.EnvName,
			Carrier:         a.Carrier,
			Owner:           a.Owner,
			KubeVersion:     a.KubeVersion,
			MachineType:     a.MachineType,
			OperatingSystem: a.OperatingSystem,
			Alert:           ac,
			Timestamp:       time.Unix(results.Current[0].Timestamp, 0)}

		switch ac.LimitType {
		case "floor":
			alt.Result = ctrMax
			if alt.Result < ac.Thresholds[a.MachineType].Error {
				alt.Sev = Error
				displayAlert(alt)
				alerts = append(alerts, alt)
			} else if alt.Result < ac.Thresholds[a.MachineType].Warn {
				alt.Sev = Warning
				displayAlert(alt)
				alerts = append(alerts, alt)
			}
		case "ceiling":
			alt.Result = ctrMin
			if alt.Result > ac.Thresholds[a.MachineType].Error {
				alt.Sev = Error
				displayAlert(alt)
				alerts = append(alerts, alt)
			} else if alt.Result > ac.Thresholds[a.MachineType].Warn {
				alt.Sev = Warning
				displayAlert(alt)
				alerts = append(alerts, alt)
			}
		}
	}

	return alerts
}
