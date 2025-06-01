/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2018, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package prometheus

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	metricsservice "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"
	"github.ibm.com/alchemy-containers/armada-performance/metrics/carrier/config"

	papi "github.com/prometheus/client_golang/api"
	prom "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// PromAPI is a wrapper for prometheus API
type PromAPI struct {
	api prom.API
}

type deviceDetails struct {
	hostname  string
	privateIP string
	publicIP  string
}

func isPrivateIP(v string) bool {
	match, err := regexp.MatchString("^10\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}$", v)
	return err == nil && match
}
func isPublicIP(v string) bool {
	match, err := regexp.MatchString("^169\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}$", v)
	return err == nil && match
}

func getDeviceDetails(deviceID string) *deviceDetails {
	const (
		hostnameIdx = 0
		privateIdx  = 4
		publicIdx   = 3
	)
	deviceFile := filepath.Join(os.Getenv("GOPATH"), config.Prometheus.Devices)
	if _, err := os.Stat(deviceFile); !os.IsNotExist(err) {
		// #nosec G304
		f, err := os.Open(deviceFile)
		if err != nil {
			log.Fatalf("Unable to open devices.csv file. Error : %s\n", err.Error())
		}
		defer f.Close()

		devices, err := csv.NewReader(f).ReadAll()
		if err != nil {
			panic(err)
		}

		var matchOn int

		if isPrivateIP(deviceID) {
			// Private IP
			matchOn = privateIdx
		} else {
			if isPublicIP(deviceID) {
				// Public IP
				matchOn = publicIdx
			} else {
				// Hostname
				matchOn = hostnameIdx
			}
		}

		for _, d := range devices {
			// First column is hostname, 4th column is Public IP
			matchEndIdx := len(d[matchOn])
			if matchOn == hostnameIdx {
				matchEndIdx = strings.Index(d[matchOn], ".")
			}

			if matchEndIdx >= 0 && d[matchOn][0:matchEndIdx] == deviceID {
				return &deviceDetails{hostname: d[hostnameIdx], privateIP: d[privateIdx], publicIP: d[publicIdx]}
			}
		}
	}
	fmt.Printf("WARNING: Unable to find device %s in %s - ensure the file is up to date\n", deviceID, deviceFile)
	return nil
}

// NewClient returns a client for accessing carrier/tugboat Prometheus metrics
func NewClient() *PromAPI {
	roundTripper := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 60 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	endpoint := "http://" + net.JoinHostPort("localhost", strconv.Itoa(config.Prometheus.Port))

	// Setup prometheus client
	pURL := strings.Join(
		[]string{endpoint,
			config.Prometheus.Environment,
			config.Prometheus.Carrier,
			"prometheus"},
		"/")

	pclient, err := papi.NewClient(papi.Config{Address: pURL, RoundTripper: roundTripper})
	if err != nil {
		log.Fatalf("Failed to init prometheus client for %s. Error : %s\n", pURL, err.Error())
	}

	// Prometheus query api client
	return &PromAPI{api: prom.NewAPI(pclient)}
}

// GatherMetrics ...
func (p PromAPI) GatherMetrics(startTime, endTime time.Time, stepInterval time.Duration) {
	// Set up prometheus range data from user supplied values
	pr := prom.Range{
		Start: startTime,
		End:   endTime,
		Step:  stepInterval,
	}

	// Loop through each metric specified in our configuration file
	for tn, tm := range config.Prometheus.Metrics {
		// We'll send data to metrics service on a per testcase basis
		var bm []metricsservice.BluemixMetric
		for _, m := range tm {
			// Get data from Prometheus endpoint
			v, _, err := p.api.QueryRange(context.Background(), m.Query, pr)
			if err != nil {
				log.Fatalf("Could not execute query: %s\nError: %s\n", m.Query, err.Error())
			}

			// For each prometheus metric
			for _, s := range v.(model.Matrix) {
				midArr := make([]string, 0, len(s.Metric))
				for _, id := range s.Metric {
					var mf = string(id)

					// Replace ip address with cut-down hostname
					if isPrivateIP(mf) {
						mf = strings.Split(getDeviceDetails(mf).hostname, ".")[0]

						mf = strings.Replace(mf, "stage-", "", 1)
						mf = strings.Replace(mf, "carrier", "c", 1)
						mf = strings.Replace(mf, "worker-", "w", 1)
					}

					midArr = append(midArr, mf)
				}
				sort.Strings(midArr)
				mid := strings.Join(midArr, ".")

				// Grab all the values (should be one per sample interval)
				for _, sp := range s.Values {
					mn := strings.Join(
						[]string{
							m.Name,
							"sparse-avg"},
						".")

					if len(mid) > 0 {
						mn = strings.Join([]string{mid, mn}, ".")
					}

					// Include the test name (if specified) at the start of the metric name
					if len(tn) > 0 {
						mn = strings.Join([]string{tn, mn}, ".")
					}

					// Add this to the list of metrics to be published, ensuring timestamp is
					// in seconds since epoch (Prometheus timestamp is in milli-seconds)
					bm = append(bm,
						metricsservice.BluemixMetric{
							Name:      mn,
							Timestamp: sp.Timestamp.Unix(),
							Value:     float64(sp.Value),
						},
					)
				}
			}
		}
		if config.Verbose {
			log.Println(bm)
		}
		if config.Publish {
			metricsservice.WriteCarrierBluemixMetrics(bm, true, tn, "")
		}
	}

}
