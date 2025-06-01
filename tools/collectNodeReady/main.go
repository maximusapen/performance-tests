/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2017, 2019 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

/* collectNodeReady will generate stats when a node is considered "NotReady" or "Unknown".
 * Output: <timestamp> <ip of worker> <unknown|false>
 * Example
 * 2018-01-10 21:29:05, 10.143.138.94, false
 * 2018-01-10 21:28:15, 10.143.138.94, unknown
 * 2018-01-10 21:28:25, 10.143.138.94, unknown
 * 2018-01-10 21:28:35, 10.143.138.94, unknown
 * 2018-01-10 21:28:45, 10.143.138.94, unknown
 * 2018-01-10 21:28:55, 10.143.138.94, unknown
 * A lot of the details of running this can be seen in the /etc/cron.d/CollectNodeReady cron entry
 * where every 2 hours it collects data from for the previous 2 hours. This is useful because this data
 * is lost after two days.
 * 0 0,2,4,6,8,10,12,14,16,18,20,22 * * * nrockwel /masterlimit/bin/collectNodeReady http://10.143.115.253:30900/stage-dal09/carrier1/prometheus `date -d "-2 hours" '+\%Y-\%m-\%dT\%H:00:00Z'` 2h >> /masterlimit/cruiserDensity/pv_kube184/collectNodeReady.`date -d "-2 hours" '+\%Y-\%m-\%d'`.csv
 */

import (
	"fmt"
	"log"
	"os"
	"time"

	papi "github.com/prometheus/client_golang/api"
	prom "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"golang.org/x/net/context"
)

var logger *log.Logger

const debug bool = false

// typedefs for data holders; values indexed by only instance (hostname) or by instance & device

// SingleMap ....
type SingleMap map[string]float64

// DoubleMap ...
type DoubleMap map[string]map[string]float64

func init() {
	logger = log.New(os.Stdout, "metrics ", log.Lshortfile|log.Ltime)
}

func main() {
	var startTime, endTime time.Time
	var err error

	// default to retreiving an hour of data from the current time
	d, err := time.ParseDuration("-60m")

	if err != nil {
		logger.Fatalln("invalid duration", err)
	}

	endpoint := "http://localhost:9090"
	now := time.Now()
	startTime = now.Add(d)
	endTime = now

	if len(os.Args) > 1 {
		endpoint = os.Args[1]
	}
	// second argument must be a time
	if len(os.Args) > 2 {
		startTime, err = parseTime(now, os.Args[2])

		if err != nil {
			logger.Fatalln("invalid start time", os.Args[2], err)
		}
	}
	// third argument can be a time or a duration
	// duration can be negative which makes the first argument the end time
	if len(os.Args) > 3 {
		endTime, err = parseTime(now, os.Args[3])

		if err != nil {
			if debug {
				logger.Println("parsing", os.Args[3], "as a duration")
			}
			d, err = time.ParseDuration(os.Args[3])

			if err != nil {
				logger.Fatalln("invalid duration ", err)
			}

			endTime = startTime.Add(d)
		}
	}

	// swap times if necessary
	if endTime.Before(startTime) {
		startTime, endTime = endTime, startTime
	}

	//fmt.Println("getting Prometheus data for", endpoint, "from", startTime.Format(time.RFC822), "to", endTime.Format(time.RFC822))

	// prometheus.Client
	client, err := papi.NewClient(papi.Config{Address: endpoint})

	if err != nil {
		logger.Println("error", err)
		return
	}

	// prometheus.QueryAPI
	q := prom.NewAPI(client)

	// Range used for all queries
	// note Step of 10 must line up with scrape interval to ensure all data is collected
	r := prom.Range{Start: startTime, End: endTime, Step: 10 * time.Second}

	collectForTwoValues(&q, "kube_node_status_ready{condition=~\"unknown|false\"}", &r)
}

func collectForTwoValues(q *prom.API, query string, r *prom.Range) {
	// Background context => no timeout
	// queries return model.Value which needs to be cast to String, Scalar, Vector or Matrix based on v.Type()
	v, _, err := (*q).QueryRange(context.Background(), query, *r)

	if err != nil {
		logger.Fatalln("could not execute query", query, err)
	}

	if debug {
		logger.Println("retrieved", len(v.(model.Matrix)), "metrics for '", query, "'")
	}

	// Matrix is an array of SampleStreams
	for _, sampleStream := range v.(model.Matrix) {
		// SampleStream.Metric is a map[string]string
		// get the instance and add it to the master list, if necessary
		instance := string(sampleStream.Metric["node"])
		condition := string(sampleStream.Metric["condition"])

		device := string(sampleStream.Metric["device"])

		// SampleStream.Values is an array of SamplePairs
		// SamplePair contains a Time and a float64 Value
		// calculate the average
		if debug {
			logger.Println("retrieved", len(sampleStream.Values), "values for", instance, ":", device)
		}

		for _, samplePair := range sampleStream.Values {
			if samplePair.Value > 0.0 {
				fmt.Printf("%s, %s, %s\n", samplePair.Timestamp.Time().Format("2006-01-02 15:04:05"), instance, condition)
			}
		}
	}
}

func parseTime(base time.Time, toParse string) (time.Time, error) {
	// try as full datetime
	toReturn, err := time.Parse(time.RFC3339, toParse)

	if err != nil {
		// try as HH:MM
		toReturn, err = time.Parse("15:04", toParse)

		if err != nil {
			// just return the base time; assume errors are checked
			return base, err
		}

		// go parsing is braindead; the HH:MM time parsed is based off the _zero_ time
		// get the value for that and the start time for today and fix it up
		zero, _ := time.Parse("15:04", "00:00")
		today := base.Truncate(time.Second * 86400)

		toReturn = time.Unix(today.Unix()+toReturn.Unix()-zero.Unix(), 0)
	}

	if debug {
		logger.Println("parsed", toParse, "to", toReturn)
	}

	return toReturn, nil
}
