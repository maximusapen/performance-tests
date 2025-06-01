/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	metricsservice "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"
)

type respWriter struct {
}

func (*respWriter) Write(bytes []byte) (int, error) {
	if verbose {
		log.Printf("%s \n", bytes)
	}
	return len(bytes), nil
}

var (
	metrics          bool
	verbose          bool
	regionalRegistry string
	registryKey      string
	testName         string
	international    bool
	allRegions       bool
	clusterRegion    string
	cli              *client.Client
	sEnc             string
	pushRespTime     time.Duration
	pullRespTime     time.Duration
	hyperkubeImage   string

	mylog = log.New(os.Stderr, "registry: ", log.LstdFlags|log.Lshortfile)
)

// Function to build docker image
func buildImage(imageName string) {
	dockerBuildContext, err := os.Open("artifacts/build.tar")
	if err != nil {
		mylog.Fatalln(err.Error())
	}

	buildResponse, err := cli.ImageBuild(context.Background(), dockerBuildContext,
		types.ImageBuildOptions{
			Tags: []string{imageName},
		})
	dockerBuildContext.Close()

	if buildResponse.Body != nil {
		defer func() {
			buildResponse.Body.Close()
		}()
	}
	if err != nil {
		mylog.Fatalln(err.Error())
	}
	buildWriter := &respWriter{}
	io.Copy(buildWriter, buildResponse.Body)
}

// Function to delete docker image locally
func deleteImage(ctx context.Context, imageName string) {
	imageDelete, err := cli.ImageRemove(ctx, imageName, types.ImageRemoveOptions{Force: true})
	if verbose {
		log.Printf("Delete response: %+v \n\n", imageDelete[0])
	}
	if err != nil {
		mylog.Fatalln(err.Error())
	}
}

// Function to push the docker image to the registry
func pushImage(ctx context.Context, imageName string) time.Duration {
	var pushStart time.Time

	pushStart = time.Now()
	pushResp, err := cli.ImagePush(ctx, imageName, types.ImagePushOptions{RegistryAuth: sEnc}) // pragma: allowlist secret
	if pushResp != nil {
		defer pushResp.Close()
	}
	if err != nil {
		mylog.Fatalln(err.Error())
	}
	pushWriter := &respWriter{}
	io.Copy(pushWriter, pushResp)
	pushRespTime = time.Since(pushStart)

	return pushRespTime
}

// Function to pull the docker image from registry
func pullImage(ctx context.Context, imageName string) time.Duration {
	var pullStart time.Time

	pullStart = time.Now()
	resp, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{RegistryAuth: sEnc}) // pragma: allowlist secret
	if resp != nil {
		defer func() {
			resp.Close()
		}()
	}
	if err != nil {
		mylog.Fatalln(err.Error())
	}
	writer := &respWriter{}
	io.Copy(writer, resp)
	pullRespTime = time.Since(pullStart)

	return pullRespTime
}

// Build, Push and Pull the image. Delete the local image after the push and pull.
func processImage(ctx context.Context, imageName string) {
	buildImage(imageName)
	pushImage(ctx, imageName)
	deleteImage(ctx, imageName)
	pullImage(ctx, imageName)
	deleteImage(ctx, imageName)
}

func main() {
	log.SetOutput(os.Stdout)

	flag.StringVar(&testName, "testname", "registry", "Test name in Jenkins - only needed if sending alerts to RazeeDash")
	flag.StringVar(&registryKey, "registrykey", "", "Registry access api-key")
	flag.StringVar(&regionalRegistry, "regionalRegistry", "registry.ng.bluemix.net", "Regional registry name")
	flag.BoolVar(&verbose, "verbose", false, "verbose output")
	flag.BoolVar(&metrics, "metrics", false, "send results to IBM Metrics service")
	flag.BoolVar(&international, "international", false, "run on international registry")
	flag.BoolVar(&allRegions, "allRegions", false, "To all regional registries from one cluster region")
	flag.StringVar(&clusterRegion, "clusterRegion", "", "Cluster region to use if allRegions specified")
	flag.StringVar(&hyperkubeImage, "hyperkubeImage", "", "hyperkube image name to pull")

	flag.Parse()

	registryMap := map[string]string{
		"ap-south": "au.icr.io",
		"us-south": "us.icr.io",
		"eu-de":    "de.icr.io",
		"eu-gb":    "uk.icr.io",
		"ap-north": "jp.icr.io",
	}

	// map the region and registry to the correct one eg. ap-north uses au-syd Registry
	s := strings.Split(regionalRegistry, ".")
	region := s[1]
	regionalRegistry = registryMap[region]

	if verbose {
		fmt.Printf("Mapped registry: %v\n", registryMap[region])
		fmt.Printf("Region: %s\n", region)
		fmt.Printf("Regional Registry: %s\n", regionalRegistry)
		fmt.Printf("Metrics: %t\n", metrics)
	}

	// Get new docker client environment setup
	var dockererr error
	cli, dockererr = client.NewEnvClient()
	if dockererr != nil {
		mylog.Fatalln(dockererr.Error())
	}

	//set docker context and credentials
	ctx := context.Background()
	auth := types.AuthConfig{ // pragma: allowlist secret
		Username: "iamapikey",
		Password: registryKey, // pragma: allowlist secret
	}
	authBytes, _ := json.Marshal(auth)
	sEnc = base64.StdEncoding.EncodeToString(authBytes)

	// Check if we want to run against the international registry or the regional registry
	// or if we want to run in one cluster but to all regions.
	// Build, push and pull the image and time how long it takes.

	// First run to the regional registries
	if !international && !allRegions {

		imageName := strings.Join([]string{regionalRegistry, "armada_performance", "perftest50mb:latest"}, "/")
		processImage(ctx, imageName)

		log.Printf("region is: %s\n\n", region)
		log.Printf("Push time for regional registry: %v \n\n", pushRespTime.Seconds())
		log.Printf("Pull time for regional registry: %v\n\n", pullRespTime.Seconds())

		_, specialRun := os.LookupEnv("SPECIAL_DNS_RUN")

		var regionMetricName = region
		if specialRun && region != "ap-south" {
			ips, err := net.LookupIP(regionalRegistry)
			log.Printf("ip used is %s", ips)
			if err != nil {
				mylog.Fatalln(err.Error())
			}
			ip := ips[0].String()
			log.Printf("SPECIAL DNS RUN: Appending ip %s to metric name\n\n", ip)

			regionMetricName = strings.Join([]string{regionMetricName, strings.Replace(ip, ".", "_", -1)}, "-")
			log.Printf("region MetricName is: %s\n\n", regionMetricName)
		}

		// If metrics was specified then append the metrics and send
		var bm = []metricsservice.BluemixMetric{}
		if metrics {
			bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{"registry", regionMetricName, "regional_pull", "sparse-avg"}, "."), Timestamp: time.Now().Unix(), Value: pullRespTime.Seconds()})
			bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{"registry", regionMetricName, "regional_push", "sparse-avg"}, "."), Timestamp: time.Now().Unix(), Value: pushRespTime.Seconds()})
			metricsservice.WriteBluemixMetrics(bm, true, testName, "")
		}

		if region != "ap-north" {

			// Now do a pull of the Hyperkube image but only if not using the dark-registry ap-north as is has no hyperkube image
			imageName = strings.Join([]string{regionalRegistry, "armada-master", hyperkubeImage}, "/")
			pullImage(ctx, imageName)
			deleteImage(ctx, imageName)

			log.Printf("region is: %s\n\n", region)
			log.Printf("Pull time for hyperkube image: %v\n\n", pullRespTime.Seconds())

			// If metrics was specified then append and send the metrics
			if metrics {
				bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{"registry", regionMetricName, "hyperkube_pull", "sparse-avg"}, "."), Timestamp: time.Now().Unix(), Value: pullRespTime.Seconds()})
				metricsservice.WriteBluemixMetrics(bm, true, testName, "")
			}
		}

	} else if allRegions {
		//  Now run in one cluster region but push/pull from all the different regional registries
		// Means we have to have clusterRegion in the metrics we send.

		imageName := strings.Join([]string{regionalRegistry, "armada_performance", "perftest50mb:latest"}, "/")
		processImage(ctx, imageName)

		log.Printf("Registry region is: %s\n\n", region)
		log.Printf("Cluster region is: %s\n\n", clusterRegion)
		log.Printf("Push time for registry: %v \n\n", pushRespTime.Seconds())
		log.Printf("Pull time for registry: %v\n\n", pullRespTime.Seconds())

		// If metrics was specified then send the metrics
		if metrics {
			var bm = []metricsservice.BluemixMetric{}
			bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{"registry", clusterRegion, region, "regional_pull", "sparse-avg"}, "."), Timestamp: time.Now().Unix(), Value: pullRespTime.Seconds()})
			bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{"registry", clusterRegion, region, "regional_push", "sparse-avg"}, "."), Timestamp: time.Now().Unix(), Value: pushRespTime.Seconds()})
			metricsservice.WriteBluemixMetrics(bm, true, testName, "")
		}
	} else {

		// Now do the International Registry push and pull if it was requested
		// The international/global registry is icr.io

		imageName := strings.Join([]string{"icr.io", "perftest", region + "-perftest50mb:latest"}, "/")
		processImage(ctx, imageName)

		log.Printf("region is : %s\n\n", region)
		log.Printf("Push time for International Registry: %v\n\n", pushRespTime.Seconds())
		log.Printf("Pull time for International Registry: %v\n\n", pullRespTime.Seconds())

		_, specialRun := os.LookupEnv("SPECIAL_DNS_RUN")

		var regionMetricName = region
		//Special section for testing global registry clone in ap-south
		if specialRun && region == "ap-south" {
			ips, err := net.LookupIP("icr.io")
			log.Printf("ip used is %s", ips)
			if err != nil {
				mylog.Fatalln(err.Error())
			}
			ip := ips[0].String()
			log.Printf("SPECIAL DNS RUN: Appending ip %s to metric name\n\n", ip)

			regionMetricName = strings.Join([]string{regionMetricName, strings.Replace(ip, ".", "_", -1)}, "-")
			log.Printf("region MetricName is: %s\n\n", regionMetricName)
		}

		// If metrics was specified then send the metrics
		if metrics {
			var bm = []metricsservice.BluemixMetric{}
			bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{"registry", regionMetricName, "international_push", "sparse-avg"}, "."), Timestamp: time.Now().Unix(), Value: pushRespTime.Seconds()})
			bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{"registry", regionMetricName, "international_pull", "sparse-avg"}, "."), Timestamp: time.Now().Unix(), Value: pullRespTime.Seconds()})
			metricsservice.WriteBluemixMetrics(bm, true, testName, "")
		}
	}

	log.Printf("Sleeping for 2 minutes to allow runReg.sh to collect the metrics files.\n")
	time.Sleep(120 * time.Second)
	log.Printf("Finished.\n")
}
