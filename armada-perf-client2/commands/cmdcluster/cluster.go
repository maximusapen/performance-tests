
package cmdcluster

import (
	"encoding/json"
	gf "flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/consts"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/flag"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/metrics"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

const (
	hardwareShared    = "shared"
	hardwareDedicated = "dedicated"
)

var verbose = false

type clusterRequest struct {
	context *cli.Context
	action  func(c *cli.Context) error
}

type clusterResponse struct {
	duration time.Duration
	err      error
}

// alignClusterTP implements a worker thread pool for handling parallel align cluster requests
func alignClusterTP(awg *sync.WaitGroup, requests <-chan clusterRequest, results chan<- error) {
	for cf := range requests {
		err := cf.action(cf.context)
		if err != nil {
			results <- err
		} else {
			results <- nil
		}

		awg.Done()
	}
}

func getAllClusters(c *cli.Context, providers []string) ([]resources.Cluster, string, error) {
	var jsonOutput strings.Builder
	var getAllProviderClusters func() ([]resources.Cluster, json.RawMessage, error)
	var clusters []resources.Cluster

	for _, provider := range providers {
		getEndpoint := func(provider string) resources.ListerEndpoint {
			return resources.GetEndpointForProvider(c, provider)
		}

		if getAllProviderClusters == nil {
			// set default get
			getAllProviderClusters = func() ([]resources.Cluster, json.RawMessage, error) {
				endpoint := getEndpoint(provider)
				return endpoint.GetAllClusters()
			}
		}

		providerClusters, providerJSON, err := getAllProviderClusters()
		if err != nil {
			return clusters, "", err
		}

		clusters = append(clusters, providerClusters...)
		jsonOutput.Write(providerJSON)
	}

	return clusters, jsonOutput.String(), nil
}

// RegisterCommands registers all `cluster` commands
func RegisterCommands() []cli.Command {
	kubeVersionFlag := cli.StringFlag{
		Name:  models.KubeVersionFlagName,
		Usage: "Specify the Kubernetes version, including at least the major.minor version. If not specified, the IKS/ROKS default version is used.",
	}

	requiredClusterNameFlag := flag.StringFlag{
		Require: true,
		StringFlag: cli.StringFlag{
			Name:  models.NameFlagName,
			Usage: "Name of the cluster to be created.",
		},
	}

	hiddenClusterIDFlag := cli.StringFlag{
		Name:   models.ClusterFlagName,
		Hidden: true,
	}

	// Use int rather than uint to support -1 for no workers
	workerCountFlag := flag.IntFlag{
		Require:    false,
		TargetName: models.WorkersFlagName,
		IntFlag: cli.IntFlag{
			Name:  models.WorkersFlagName,
			Usage: "The number of cluster worker nodes.",
			Value: consts.DefaultWorkerNum,
		},
	}

	return []cli.Command{
		{
			Name:        models.NamespaceCluster,
			Usage:       "View and modify cluster and cluster service settings.",
			Description: "View and modify cluster and cluster service settings.",
			Category:    models.ClusterManagementCategory,
			Subcommands: []cli.Command{
				{
					Name:        models.CmdCreate,
					Description: "Create a 'Classic', 'VPC' or 'Satellite' cluster",
					Usage:       "Create a 'Classic', 'VPC' or 'Satellite' cluster",
					Subcommands: []cli.Command{
						{
							Name:        models.ProviderClassic,
							Description: "Create a cluster on classic infrastructure.",
							Usage:       "Create a cluster on classic infrastructure.",
							Flags: []cli.Flag{
								requiredClusterNameFlag,
								models.QuantityFlag,
								models.ThreadFlag,
								hiddenClusterIDFlag, // needed to support worker polling
								models.RequiredZoneFlag,
								workerCountFlag,
								models.MachineTypeFlag,
								models.OperatingSystemFlag,
								kubeVersionFlag,
								models.ServiceSubnetFlag,
								models.NoSubnetFlag,
								models.PodSubnetFlag,
								models.WorkerFailuresFlag,
								models.PollIntervalFlag,
								models.MetricsFlag,
								models.SuffixFlag,
								models.TimeoutFlag,
								models.HardwareFlag,
								models.MasterPrivateServiceEndpointFlag,
								models.MasterPublicServiceEndpointFlag,
							},
							Action: classicClusterCreate,
						},
						{
							Name:        models.ProviderVPCClassic,
							Description: "Create a Virtual Private Cloud (VPC) cluster on Generation 1 infrastructure.",
							Usage:       "Create a Virtual Private Cloud (VPC) cluster on Generation 1 infrastructure.",
							Flags: []cli.Flag{
								requiredClusterNameFlag,
								models.QuantityFlag,
								models.ThreadFlag,
								hiddenClusterIDFlag, // needed to support worker polling
								models.RequiredZoneFlag,
								workerCountFlag,
								models.MachineTypeFlag,
								models.OperatingSystemFlag,
								models.RequiredVPCIDFlag,
								kubeVersionFlag,
								models.ServiceSubnetFlag,
								models.PodSubnetFlag,
								models.WorkerFailuresFlag,
								models.PollIntervalFlag,
								models.MetricsFlag,
								models.SuffixFlag,
								models.TimeoutFlag,
							},
							Action: vpcClassicClusterCreate,
						},
						{
							Name:        models.ProviderVPCGen2,
							Description: "Create a Virtual Private Cloud (VPC) cluster on Generation 2 infrastructure.",
							Usage:       "Create a Virtual Private Cloud (VPC) cluster on Generation 2 infrastructure.",
							Flags: []cli.Flag{
								requiredClusterNameFlag,
								models.QuantityFlag,
								models.ThreadFlag,
								hiddenClusterIDFlag, // needed to support worker polling
								models.RequiredZoneFlag,
								workerCountFlag,
								models.MachineTypeFlag,
								models.OperatingSystemFlag,
								models.RequiredVPCIDFlag,
								kubeVersionFlag,
								models.ServiceSubnetFlag,
								models.PodSubnetFlag,
								models.WorkerFailuresFlag,
								models.PollIntervalFlag,
								models.MetricsFlag,
								models.SuffixFlag,
								models.TimeoutFlag,
							},
							Action: vpcNextGenClusterCreate,
						},
						{
							Name:        models.ProviderSatellite,
							Description: "Create an IBM Cloud Satellite cluster on your own infrastructure.",
							Usage:       "Create an IBM Cloud Satellite cluster on your own infrastructure.",
							Flags: []cli.Flag{
								requiredClusterNameFlag,
								models.RequiredLocationFlag,
								models.OperatingSystemFlag,
								kubeVersionFlag,
								models.ZoneFlag,
								models.QuantityFlag,
								models.ThreadFlag,
								models.PollIntervalFlag,
								models.MetricsFlag,
								models.SuffixFlag,
								models.TimeoutFlag,
							},
							Action: satelliteClusterCreate,
						},
					},
				},
				{
					Name:        models.CmdConfig,
					Description: "Download the Kubernetes configuration files and certificates to connect to your cluster by using kubectl commands.",
					Usage:       "Download the Kubernetes configuration files and certificates to connect to your cluster by using kubectl commands.",
					Flags: []cli.Flag{
						models.RequiredClusterFlag,
						models.AdminFlag,
						models.NetworkFlag,
					},
					Action: clusterConfigAction,
				},
				{
					Name:        models.CmdGet,
					Description: "View the details of a cluster.",
					Usage:       "View the details of a cluster.",
					Flags: []cli.Flag{
						models.RequiredClusterFlag,
						models.JSONOutFlag,
					},
					Action: clusterGet,
				},
				{
					Name:        models.CmdList,
					Description: "List all clusters in your IBM Cloud account.",
					Usage:       "List all clusters in your IBM Cloud account.",
					Flags: []cli.Flag{
						models.ProviderForListFlag,
						models.LocationFlag,
						models.JSONOutFlag,
					},
					Action: clusterList,
				},
				{
					Name:        models.CmdRemove,
					Description: "Delete a cluster.",
					Usage:       "Delete a cluster.",
					Flags: []cli.Flag{
						models.RequiredClusterFlag,
						models.ForceDeleteStorageFlag,
						models.QuantityFlag,
						models.MetricsFlag,
						models.PollIntervalFlag,
						models.SuffixFlag,
						models.TimeoutFlag,
					},
					Action: clusterRm,
				},
				{
					Name:        models.CmdAlign,
					Description: "Adjust cluster count if necessary.",
					Usage:       "Adjust cluster count if necessary.",
					Flags: []cli.Flag{
						requiredClusterNameFlag,
						hiddenClusterIDFlag, // needed to support cluster deletion and worker polling
						models.ProviderForListFlag,
						models.LocationFlag,
						models.QuantityFlag,
						models.MetricsFlag,
						models.ThreadFlag,
						workerCountFlag,
						kubeVersionFlag,
						models.MachineTypeFlag,
						models.NoSubnetFlag,
						models.MasterPrivateServiceEndpointFlag,
						models.MasterPublicServiceEndpointFlag,
						models.ForceDeleteStorageFlag,
						models.PollIntervalFlag,
						models.TimeoutFlag,
					},
					Action: clusterAlign,
				},
				AddonSubCommandRegister(),
			},
		},
	}
}

// Handles 'cluster ls' command
func clusterList(c *cli.Context) error {
	var providers []string

	jsonoutput := c.Bool(models.JSONOutFlagName)
	providerStr := c.String(models.ProviderIDFlagName)

	if providerStr == "" {
		providers = []string{models.ProviderClassic, models.ProviderVPCGen2}
	} else {
		providers = []string{providerStr}
	}

	locationNameOrID, err := cliutils.GetFlagString(c, models.LocationFlagName, true, "Location name or ID", nil)
	if err != nil {
		return err
	}

	clusters, jsonMessage, err := getAllClusters(c, providers)
	if err != nil {
		log.Fatalf("Failed to get existing clusters : %s", err.Error())
	}

	if jsonoutput {
		fmt.Fprintln(c.App.Writer, jsonMessage)
		return nil
	}

	fmt.Fprintln(c.App.Writer)
	fmt.Fprintln(c.App.Writer, "Name\tID\tState\tCreated\tWorkers\tLocation\tVersion\tProvider")
	fmt.Fprintln(c.App.Writer, "----\t--\t-----\t-------\t-------\t---------\t-------\t--------")

	for _, cluster := range clusters {
		if len(locationNameOrID) == 0 || cluster.Location() == locationNameOrID {
			fmt.Fprintf(c.App.Writer,
				"%s\t%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
				cluster.Name(),
				cluster.ID(),
				cluster.State(),
				cluster.CreatedDate(),
				cluster.WorkerCount(),
				cluster.Location(),
				cluster.MasterVersion(),
				cluster.Provider(),
			)
		}
	}

	return nil
}

// Handles 'cluster get' command
func clusterGet(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	jsonoutput := c.Bool(models.JSONOutFlagName)

	endpoint := resources.GetArmadaV2Endpoint(c)
	rawJSON, v2Cluster, err := endpoint.GetCluster(clusterNameOrID)

	if err != nil {
		return err
	}

	if jsonoutput {
		cliutils.WriteJSON(c, rawJSON)
		return nil
	}

	fmt.Fprintln(c.App.Writer)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Name:", v2Cluster.Name())
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "ID:", v2Cluster.ID())
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Type:", v2Cluster.ClusterType())
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Provider:", v2Cluster.Provider())
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "State:", v2Cluster.State())
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Created:", v2Cluster.CreatedDate())
	fmt.Fprintf(c.App.Writer, "%s\t%d\n", "Workers:", v2Cluster.WorkerCount())
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Location:", v2Cluster.Location())
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Master Version:", v2Cluster.MasterVersion())
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Master State:", v2Cluster.MasterState())
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Master Status:", v2Cluster.MasterStatus())
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Master Health:", v2Cluster.MasterHealth())
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Master URL:", v2Cluster.MasterURL())
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Ingress Hostname:", v2Cluster.IngressHostname())
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Ingress Secret:", v2Cluster.IngressSecret()) // pragma: allowlist secret

	if len(v2Cluster.IngressStatus()) > 0 {
		fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Ingress Status:", v2Cluster.IngressStatus())

	}
	if len(v2Cluster.IngressMessage()) > 0 {
		fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Ingress Message:", v2Cluster.IngressMessage())
	}

	return nil
}

// Handles 'cluster rm' command
func clusterRm(c *cli.Context) error {
	var clusterPollInterval, timeout time.Duration

	// Initialize metrics data
	metricsData := cliutils.GetMetadataValue(c, models.MetricsFlagName).(*metrics.Data)
	metricsData.Command = fmt.Sprintf("%s-cluster", strings.Split(cliutils.GetCommandName(c), " ")[1])
	metricsData.Enabled = cliutils.GetFlagBool(c, models.MetricsFlagName)

	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	// Flag to request that a numerical suffix is added to the specified cluster name
	// Note that if the specified quantity is more than one, a numerical suffix will be automatically added.
	suffix := cliutils.GetFlagBool(c, models.SuffixFlagName)

	quantity, err := cliutils.GetFlagInt(c, models.QuantityFlagName, true, "Number of clusters", nil)
	if err != nil {
		return err
	}

	clusterPollIntervalStr, err := cliutils.GetFlagString(c, models.PollIntervalFlagName, true, "Cluster status polling interval", nil)
	if err != nil {
		return err
	}

	if len(clusterPollIntervalStr) > 0 {
		clusterPollInterval, err = time.ParseDuration(clusterPollIntervalStr)
		if err != nil {
			return err
		}
	}

	// Delete resources too? Default to yes.
	resourceDeletion := cliutils.GetFlagBoolT(c, models.ForceDeleteStorageFlagName)
	if verbose {
		if resourceDeletion {
			fmt.Printf("Removing cluster '%s', persistent storage...\n", clusterNameOrID)
		} else {
			fmt.Printf("Removing cluster '%s'\n", clusterNameOrID)
		}
	}

	timeoutStr, err := cliutils.GetFlagString(c, models.TimeoutFlagName, true, "Cluster deletion polling timeout", nil)
	if err != nil {
		return err
	}
	if len(timeoutStr) > 0 {
		timeout, err = time.ParseDuration(timeoutStr)
		if err != nil {
			return err
		}
	}

	endpoint := resources.GetArmadaEndpoint(c)

	clustersForDeletion := make(map[string]bool)

	startTime := time.Now()

	for cl := 1; cl <= quantity; cl++ {
		var clusterName = clusterNameOrID

		if suffix || quantity > 1 {
			clusterName = fmt.Sprintf("%s%d", clusterName, cl)
		}
		if err := endpoint.RemoveCluster(clusterName, resourceDeletion); err != nil {
			return err
		}

		clustersForDeletion[clusterName] = true

		dsinfo := ""
		if resourceDeletion {
			dsinfo = " and storage"
		}
		fmt.Fprintf(c.App.Writer, "Cluster deletion request for cluster '%s'%s successful.\n", clusterName, dsinfo)
	}

	// Wait for the  cluster(s) to be deleted.
	// For now we'll just report the time to delete all clusters. This could be extended reasonably simply to report the deletion times of individual clusters.
	if clusterPollInterval > 0 {
		clustersDeleted := false
		matchedClusters := 0
		workerCount := 0

		fmt.Fprintln(c.App.Writer)

		for !clustersDeleted {
			var clustersOfInterest []resources.Cluster

			if timeout > 0 && time.Since(startTime) > timeout {
				return fmt.Errorf("timeout waiting for cluster deletion to complete. Timeout: %v", timeout)
			}

			// Get all clusters we have access to
			clusters, _, err := getAllClusters(c, []string{models.ProviderClassic, models.ProviderVPCGen2})
			if err != nil {
				time.Sleep(clusterPollInterval)
				continue
			}

			clustersDeleted = true

			// Construct a list of clusters that we are deleting
			for _, cl := range clusters {
				_, idMatch := clustersForDeletion[cl.ID()]
				_, nameMatch := clustersForDeletion[cl.Name()]
				if idMatch || nameMatch {
					clustersOfInterest = append(clustersOfInterest, cl)
					if workerCount == 0 {
						workerCount = cl.WorkerCount()
					}

					if !strings.EqualFold(cl.State(), "delete_failed") {
						clustersDeleted = false
					}
				}
			}

			if matchedClusters == 0 || matchedClusters != len(clustersOfInterest) {
				if matchedClusters != 0 {
					// Let's assume all clusters have the same number of workers for now. Mostly there will only be one cluster anyway.
					metricsData.C.WorkerCount = workerCount
					metricsData.C.Duration = append(metricsData.C.Duration, time.Since(startTime))
				}

				if len(clustersOfInterest) > 0 {
					fmt.Fprintln(c.App.Writer, time.Now().Format(time.Stamp))
					for _, cl := range clustersOfInterest {
						fmt.Fprintf(c.App.Writer, "\t%s : %s\n", cl.Name(), cl.State())
					}
					fmt.Fprintln(c.App.Writer)
				}
			}

			matchedClusters = len(clustersOfInterest)

			if !clustersDeleted {
				time.Sleep(clusterPollInterval)
			}
		}

		fmt.Fprintf(c.App.Writer, "Total Duration: %.1fs\n", time.Since(startTime).Seconds())
	}

	return nil
}

// Handles 'cluster align' command
func clusterAlign(c *cli.Context) error {
	var provider string
	var awg sync.WaitGroup

	clusterNamePrefix, err := cliutils.GetFlagString(c, models.NameFlagName, false, "Name of the cluster to be created.", nil)
	if err != nil {
		return err
	}

	quantity, err := cliutils.GetFlagInt(c, models.QuantityFlagName, true, "Number of clusters", nil)
	if err != nil {
		return err
	}

	threads, err := cliutils.GetFlagInt(c, models.ThreadFlagName, true, "Number of threads to handle multiple requests in parallel", nil)
	if err != nil {
		return err
	}

	// Get provider, default to VPC Gen2
	providerStr := c.String(models.ProviderIDFlagName)
	if providerStr == "" {
		provider = models.ProviderVPCGen2
	} else {
		provider = providerStr
	}

	clusters, _, err := getAllClusters(c, []string{provider})
	if err != nil {
		log.Fatalf("Failed to list clusters : %s", err.Error())
	}

	var maxIndex int
	var existingKubeVersion string

	existingClusters := make([]string, 0, 100)
	for _, c := range clusters {
		if strings.HasPrefix(c.Name(), clusterNamePrefix) {
			index, err := strconv.Atoi(strings.TrimPrefix(c.Name(), clusterNamePrefix))
			if err == nil {
				existingClusters = append(existingClusters, c.Name())
				existingKubeVersion = c.TargetVersion()

				if index > maxIndex {
					maxIndex = index
				}
			}
		}
	}

	// Setup a pool of worker threads to handle cluster creation request(s)
	diff := len(existingClusters) - quantity
	totalRequests := diff
	if totalRequests < 0 {
		totalRequests = -totalRequests
	}

	requests := make(chan clusterRequest, totalRequests)
	results := make(chan error, totalRequests)
	awg.Add(totalRequests)

	for t := 1; t <= threads; t++ {
		go alignClusterTP(&awg, requests, results)
	}

	// Reset quantity flag
	c.Set(models.QuantityFlagName, "1")

	switch {
	case diff < 0:
		// Need to create cluster(s)
		kubeVersion, err := cliutils.GetFlagString(c, models.KubeVersionFlagName, true, "Cluster Kubernetes version", existingKubeVersion)
		if err != nil {
			return err
		}
		c.Set(models.KubeVersionFlagName, kubeVersion)

		index := maxIndex + 1
		for i := diff; i < 0; i++ {
			tc := cli.NewContext(cli.NewApp(), &gf.FlagSet{}, c)
			tc.App.Metadata = map[string]interface{}{
				models.NameFlagName:    fmt.Sprintf("%s%d", clusterNamePrefix, index),
				models.ClusterFlagName: fmt.Sprintf("%s%d", clusterNamePrefix, index),
			}

			r := clusterRequest{context: tc}

			switch provider {
			case models.ProviderClassic:
				r.action = classicClusterCreate

			case models.ProviderVPCClassic:
				r.action = vpcClassicClusterCreate

			case models.ProviderVPCGen2:
				r.action = vpcNextGenClusterCreate

			case models.ProviderSatellite:
				r.action = satelliteClusterCreate
			}

			requests <- r

			index++
		}
	case diff > 0:
		// Need to delete cluster(s)
		for i := 0; i < diff; i++ {
			tc := cli.NewContext(cli.NewApp(), &gf.FlagSet{}, c)
			tc.App.Metadata = map[string]interface{}{
				models.ClusterFlagName: existingClusters[i],
			}

			r := clusterRequest{context: tc, action: clusterRm}
			requests <- r
		}
	}

	awg.Wait()

	// Return any error
	close(results)
	for resp := range results {
		if resp != nil {
			return resp
		}
	}

	return nil
}
