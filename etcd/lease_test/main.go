/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2021, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/
// Generates a set number of leases, then ramp the lease count up at the specified intervals

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	uuid "github.com/nu7hatch/gouuid"
	"github.com/spf13/cobra"
	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var rootCmd = &cobra.Command{
	Use:   "lease_test",
	Short: "Etcd lease generator",
	Long:  "A tool for generating etcd leases",

	Run: leaseCmdFunc,
}

var (
	endpoints []string

	threads              int
	leaseDurationSeconds int
	rampDurationMinutes  int

	rampCycles          int
	rampStartLeases     int
	rampIncrementLeases int

	// The default (i.e. false for rampAtConstantRate) changes the rate leases are requested to vary the number of active leases
	// Setting to true uses the same rate (i.e. fixedLeaseTickerSeconds), and changes the duration of the lease to vary the number of active leases
	rampAtConstantRate      bool
	fixedLeaseTickerSeconds int

	rampDelayMinutes int
	startupDelay     time.Duration

	tls transport.TLSInfo
)

func init() {
	rootCmd.PersistentFlags().StringSliceVar(&endpoints, "endpoints", []string{"127.0.0.1:2379"}, "gRPC endpoints")

	rootCmd.PersistentFlags().StringVar(&tls.CertFile, "cert", "", "Identify HTTPS client using this SSL certificate file")
	rootCmd.PersistentFlags().StringVar(&tls.KeyFile, "key", "", "Identify HTTPS client using this SSL key file")
	rootCmd.PersistentFlags().StringVar(&tls.TrustedCAFile, "cacert", "", "Verify certificates of HTTPS-enabled servers using this CA bundle")

	rootCmd.PersistentFlags().IntVar(&threads, "threads", 200, "The number of threads that will generate leases.")
	rootCmd.PersistentFlags().IntVar(&leaseDurationSeconds, "lease-duration-seconds", 2000, "The duration of each lease")
	rootCmd.PersistentFlags().IntVar(&rampDurationMinutes, "ramp-duration-minutes", 40, "The number of minutes to take to generate the full set of leases in etcd")
	rootCmd.PersistentFlags().IntVar(&rampCycles, "ramp-cycles", 4, "The number ramp up cycles")
	rootCmd.PersistentFlags().IntVar(&rampStartLeases, "ramp-start-leases", 250000, "The number of leases the first ramp up will create")
	rootCmd.PersistentFlags().BoolVar(&rampAtConstantRate, "ramp-at-constant-rate", false, "False: maintain constant lease duration, adjust lease request rate. True: Adjust duration of lease, and use --fixed-lease-ticker-seconds for the lease request rate")
	rootCmd.PersistentFlags().IntVar(&rampIncrementLeases, "ramp-increment-leases", 250000, "Number of additional leases to add with each ramp cycle")
	rootCmd.PersistentFlags().IntVar(&fixedLeaseTickerSeconds, "fixed-lease-ticker-seconds", 1, "The duration, in seconds, between each thread adding a lease")
	rootCmd.PersistentFlags().IntVar(&rampDelayMinutes, "ramp-delay-minutes", 10, "The delay (in minutes) between ramp cycles. If set to -1 then leases will be maintained indefinitely")
	rootCmd.Flags().DurationVar(&startupDelay, "startup-delay", startupDelay, "The time to wait before leases are created")
}

func createClient(endpoints []string) *clientv3.Client {
	cfg := clientv3.Config{Endpoints: endpoints, DialTimeout: 5 * time.Second}

	if !tls.Empty() {
		cfgtls, err := tls.ClientConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "bad tls config: %v\n", err)
			os.Exit(1)
		}
		cfg.TLS = cfgtls
	}

	client, err := clientv3.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial error: %v\n", err)
		os.Exit(1)
	}
	return client
}

var cli *clientv3.Client

func getLeaseTickerDuration(leases int) time.Duration {
	if rampAtConstantRate {
		return time.Second * time.Duration(fixedLeaseTickerSeconds)
	}
	return time.Millisecond * time.Duration(leaseDurationSeconds*1000*threads/leases)
}

func generateLease(leaseDurationSeconds int64) {
	resp, err := cli.Grant(context.TODO(), leaseDurationSeconds)
	if err != nil {
		log.Print(err)
	} else {
		// after leaseDurationSeconds seconds, the key 'foo' will be removed
		u, err := uuid.NewV4()
		_, err = cli.Put(context.TODO(), u.String(), "bar", clientv3.WithLease(resp.ID))
		if err != nil {
			log.Print(err)
		}
	}
}

func getRampScheme(rampCycle int) (int, time.Duration, int64, int) {
	leases := rampStartLeases + (rampCycle * rampIncrementLeases)
	tickerDuration := getLeaseTickerDuration(leases)
	var localLeaseDurationSeconds int64
	var rampMinutes, rampSeconds int
	if rampAtConstantRate {
		rampSeconds = ((fixedLeaseTickerSeconds * leases) / threads)
		localLeaseDurationSeconds = int64(rampSeconds)
	} else {
		rampSeconds = (int(tickerDuration/time.Millisecond) * leases) / threads / 1000
		localLeaseDurationSeconds = int64(leaseDurationSeconds)
	}
	if rampAtConstantRate {
		rampSeconds = (fixedLeaseTickerSeconds * leases) / threads
	} else {
		rampSeconds = (int(tickerDuration/time.Millisecond) * leases) / threads / 1000
	}
	rampMinutes = rampSeconds / 60
	rampDisplaySeconds := rampSeconds - (rampMinutes * 60)
	if rampDelayMinutes > 0 {
		rampMinutes = rampMinutes + rampDelayMinutes
	}
	var durationString string
	if rampDelayMinutes < 0 {
		durationString = "indefinitely"
	} else {
		durationString = "for " + strconv.Itoa(rampMinutes) + " minutes and " + strconv.Itoa(rampDisplaySeconds) + " seconds"
	}
	log.Printf("Deploy %d active leases with a duration of %v seconds, via %v threads producing lease every %v => %.1f reqs/second (all threads), %s\n", leases, localLeaseDurationSeconds, threads, tickerDuration, float32(int64(time.Minute)/int64(tickerDuration)*int64(threads))/60, durationString)
	return leases, tickerDuration, localLeaseDurationSeconds, rampSeconds
}

func leaseCmdFunc(cmd *cobra.Command, args []string) {
	if rampDelayMinutes < 0 {
		rampCycles = 1
	}
	fmt.Println("The plan")
	for i := 0; i < rampCycles; i++ {
		getRampScheme(i)
	}
	// os.Exit(1)

	cli = createClient(endpoints)

	if startupDelay > 0 {
		log.Printf("Delaying startup: %v\n", startupDelay)
		time.Sleep(startupDelay)
	}

	log.Printf("Start")
	var wg sync.WaitGroup
	for r := 0; r < rampCycles; r++ {
		leases, tickerDuration, localLeaseDurationSeconds, rampSeconds := getRampScheme(r)

		for k := 0; k < threads; k++ {
			wg.Add(1)
			go func(threadNum int) {
				defer wg.Done()

				leaseTicker := time.NewTicker(tickerDuration)
				defer leaseTicker.Stop()
				if rampDelayMinutes == 0 {
					for i := 0; i < leases/threads; i++ {
						select {
						case <-leaseTicker.C:
							generateLease(localLeaseDurationSeconds)
						}
					}
				} else if rampDelayMinutes < 0 {
					for {
						select {
						case <-leaseTicker.C:
							generateLease(localLeaseDurationSeconds)
						}
					}
				} else {
					rampTicker := time.NewTicker(time.Second*time.Duration(rampSeconds) + time.Duration(rampDelayMinutes)*time.Minute)
					defer rampTicker.Stop()
				forExit:
					for {
						select {
						case <-leaseTicker.C:
							generateLease(localLeaseDurationSeconds)
						case <-rampTicker.C:
							break forExit
						}
					}
				}
			}(k)
		}
		wg.Wait()
		leasesRespone, err := cli.Leases(context.TODO())
		if err == nil {
			log.Printf("    Leases active at end of ramp cycle: %v\n", len(leasesRespone.Leases))
		}
	}
	defer cli.Close()
	log.Printf("End")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}
