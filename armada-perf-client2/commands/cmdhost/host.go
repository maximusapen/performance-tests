/*******************************************************************************
 * I
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2020, 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdhost

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-api-model/json/v2/requests"
	"github.ibm.com/alchemy-containers/armada-model/model"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/metrics"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

var debug = true

// HostInfrastructure defines supported infrastructure operations
type HostInfrastructure interface {
	CreateHost(string, string, cliutils.TomlSatelliteInfrastructureHostConfig, cliutils.TomlSatelliteInfrastructureHost) (cliutils.TomlSatelliteInfrastructureHost, error)
	CancelHost(string, string, string) error
	GetHost(string, string, string) (cliutils.TomlSatelliteInfrastructureHost, error)
	ReloadHost(cliutils.TomlSatelliteInfrastructureLocation, string, string) error
	GetLocationHosts(string) (map[string]string, error)
	WaitForHosts(map[string]cliutils.TomlSatelliteInfrastructureHost) error

	ProvisionDelay() time.Duration

	SetUserData(string)

	String() string
}

// RegisterCommands registers the Satellite host commands
func RegisterCommands() []cli.Command {
	hostCountFlag := cli.StringFlag{
		Name:  models.QuantityFlagName,
		Usage: "The number of hosts to attach/assign. Do not specify with '--hosts' flag",
	}

	return []cli.Command{
		{
			Name:        models.NamespaceHost,
			Description: "View and modify Satellite host settings.",
			Usage:       "View and modify Satellite host settings.",
			Subcommands: []cli.Command{
				{
					Name:        models.CmdAssign,
					Description: "Assign a host to a Satellite location or cluster.",
					Usage:       "Assign a host to a Satellite location or cluster.",
					Flags: []cli.Flag{
						models.RequiredLocationFlag,
						models.RequiredClusterFlag,
						models.HostsFlag,
						hostCountFlag,
						models.WorkerPoolFlag,
						models.LabelsFlag,
						models.RequiredZoneFlag,
						models.MetricsFlag,
						models.PollIntervalFlag,
						models.TimeoutFlag,
					},
					Action: hostAssign,
				},
				{
					Name:        models.CmdAttach,
					Description: "Create/run a script on a host in your infrastructure. The script when executed, attaches the host to your Satellite location.",
					Usage:       "Create/run a script on a host in your infrastructure. The script when executed, attaches the host to your Satellite location.",
					Flags: []cli.Flag{
						models.RequiredLocationFlag,
						hostCountFlag,
						models.InfrastructureTypeFlag,
						models.OperatingSystemFlag,
						models.SuffixStrFlag,
						models.PublicKeyFlag,
						models.PrivateKeyFlag,
						models.AutomateFlag,
						models.ControlFlag,
						models.LabelsFlag,
						models.MetricsFlag,
						models.PollIntervalFlag,
						models.TimeoutFlag,
					},
					Action: hostAttach,
				},
				{
					Name:        models.CmdGet,
					Description: "View the details of a compute host within a location.",
					Usage:       "View the details of a compute host within a location.",
					Flags: []cli.Flag{
						models.RequiredLocationFlag,
						models.RequiredHostFlag,
						models.JSONOutFlag,
					},
					Action: hostGet,
				},
				{
					Name:        models.CmdList,
					Description: "List all compute hosts that are visible within a location.",
					Usage:       "List all compute hosts that are visible within a location.",
					Flags: []cli.Flag{
						models.RequiredLocationFlag,
						models.JSONOutFlag,
						models.MetricsFlag,
						models.PollIntervalFlag,
						models.TimeoutFlag,
					},
					Action: hostList,
				},
				{
					Name:        models.CmdRemove,
					Description: "Remove a host from a location.",
					Usage:       "Remove a host from a location.",
					Flags: []cli.Flag{
						models.RequiredLocationFlag,
						models.HostsFlag,
						models.ProvisionedFlag,
						models.ClusterFlag,
						models.ControlFlag,
						models.CancelFlag,
						models.ForceFlag,
						models.ReloadFlag,
						models.MetricsFlag,
						models.PollIntervalFlag,
					},
					Action: hostRemove,
				},
				{
					Name:        models.CmdUpdate,
					Description: "Update host information, such as labels.",
					Usage:       "Update host information, such as labels.",
					Flags: []cli.Flag{
						models.RequiredLocationFlag,
						models.RequiredHostFlag,
						models.LabelsFlag,
					},
					Action: hostUpdate,
				},
			},
		},
	}
}

// Handle Satellite host assignment commands
func hostAssign(c *cli.Context) error {
	var hostPollInterval time.Duration
	var timeout time.Duration
	var zone string

	hosts := make(map[string]cliutils.TomlSatelliteInfrastructureHost)

	// Initialize metrics data
	metricsData := cliutils.GetMetadataValue(c, models.MetricsFlagName).(*metrics.Data)
	metricsData.Command = fmt.Sprintf("%s-host", strings.Split(cliutils.GetCommandName(c), " ")[1])
	metricsData.Enabled = cliutils.GetFlagBool(c, models.MetricsFlagName)

	// Initialize api endpoint
	endpoint := resources.GetArmadaV2Endpoint(c)

	// Get preconfigured host(s) data
	satelliteConfig := cliutils.GetInfrastructureConfig().Satellite

	// Required. The name or ID of the location where the Satellite control plane or cluster exists and to which the compute host should be assigned.
	locationNameOrID, err := cliutils.GetFlagString(c, models.LocationFlagName, false, "Location name or ID", nil)
	if err != nil {
		return err
	}

	_, location, _, err := endpoint.GetLocation(locationNameOrID)
	if err != nil {
		return err
	}

	// Get the location configuration data.
	locationConfig, ok := satelliteConfig.Locations[location.Name]
	if !ok {
		return fmt.Errorf("location '%s' not found in perf-infrastructure.toml configuration file", location.Name)
	}

	// Optional. The name or ID of the cluster to which the compute host should be assigned. If not specified, will assume assignment to control plane
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, locationNameOrID)
	if err != nil {
		return err
	}
	// If cluster and location are the same, we should assign to control plane
	satelliteControlPlane := clusterNameOrID == locationNameOrID

	// Optional. The name or ID of the worker pool in your cluster to which the compute host should be added. Not required for assignment of hosts to a Satellite control plane.
	workerPoolID, err := cliutils.GetFlagString(c, models.WorkerPoolFlagName, true, models.WorkerPoolFlagValue, nil)
	if err != nil {
		return err
	}

	hostPollIntervalStr, err := cliutils.GetFlagString(c, models.PollIntervalFlagName, true, "Worker status polling interval", nil)
	if err != nil {
		return err
	}
	if len(hostPollIntervalStr) > 0 {
		hostPollInterval, err = time.ParseDuration(hostPollIntervalStr)
		if err != nil {
			return err
		}
	}

	timeoutStr, err := cliutils.GetFlagString(c, models.TimeoutFlagName, true, "Host assign polling timeout", nil)
	if err != nil {
		return err
	}
	if len(timeoutStr) > 0 {
		timeout, err = time.ParseDuration(timeoutStr)
		if err != nil {
			return err
		}
	}

	// Optional. The name or ID of the host to be assigned to a Satellite control plane or cluster.
	requestedHosts := c.StringSlice(models.HostFlagName)

	// Optional. A list of key-value-pair labels that you want to add to your compute hosts. Labels can help find hosts more easily later.
	labelsMap, err := cliutils.MakeLabelsMap(c)
	if err != nil {
		return err
	}

	var labelVal string
	if satelliteControlPlane {
		labelVal = models.ControlLabel
	} else {
		labelVal = models.ServiceLabel
	}

	// If no labels were specified, we'll assign a simple environment usage label
	if len(labelsMap) == 0 {
		labelsMap["env"] = labelVal
	}

	// Distinguish between control plane and cluster hosts assignment for metrics reporting
	metricsData.Command = fmt.Sprintf("%s.%s", metricsData.Command, labelsMap["env"])

	// If hosts were specified, we'll use those. Must also have a zone specified in this case.
	if len(requestedHosts) > 0 {
		for _, rh := range requestedHosts {
			hosts[rh] = cliutils.TomlSatelliteInfrastructureHost{}
		}

		// Required. The name of the zone where the compute host should be assigned. The zone must belong to the multizone metro city that the location is manage from.
		zone, err = cliutils.GetFlagString(c, models.ZoneFlagName, false, "IBM Cloud Zone", nil)
		if err != nil {
			return err
		}
	} else {
		// No hosts specified, we'll use any hosts specified in the configuration file.

		// If cluster and location names are the same, we'll assign the pre-configured control plane hosts; otherwise the cluster hosts
		if satelliteControlPlane {
			hosts = locationConfig.Hosts.Control.Servers
		} else {
			hosts = locationConfig.Hosts.Cluster.Servers
		}

		// Optional. The name of the zone where the compute host should be assigned. The zone must belong to the multizone metro city that the location is managed from.
		zone, err = cliutils.GetFlagString(c, models.ZoneFlagName, true, "IBM Cloud Zone", nil)
		if err != nil {
			return err
		}
	}

	if hostCount := len(hosts); hostCount > 0 {
		// Check if we're to assign all the hosts
		quantityStr, err := cliutils.GetFlagString(c, models.QuantityFlagName, true, "Number of hosts to assign", nil)
		if err != nil {
			return err
		}
		if len(quantityStr) > 0 {
			hostCount, err = strconv.Atoi(quantityStr)
			if err != nil {
				return fmt.Errorf("invalid value for '--%s' specified. Enter a numerical value", models.QuantityFlagName)
			}
		}

		// Sort hosts based on hostname
		shc := sortedHosts(hosts)

		fmt.Fprintln(c.App.Writer)
		startTime := time.Now()
		hostsAssigned := 0
		for _, name := range shc {
			if hostsAssigned == hostCount {
				break
			}

			host := hosts[name]

			// Don't try to assign any already assigned nodes.
			if host.Cluster == "" {
				// Documentation suggests that either name or ID can be used. Experience suggests we need the ID.
				_, h, err := endpoint.GetHost(locationNameOrID, name)
				if err != nil {
					return err
				}

				// Check the host has been attached to the location.
				if h.ID == "" {
					return fmt.Errorf("host '%s' not attched to location. Error in configuration file", name)
				}

				// Double check the host's assignment status.
				if h.State == model.QueueNodeAssignmentStateAssigned {
					return fmt.Errorf("host '%s' already assigned. Error in configuration file", name)
				}

				// Has the zone been overriden
				if len(zone) > 0 {
					host.Zone = zone
				}

				// Build the config object
				config := requests.MultishiftCreateAssignment{
					MultishiftNodeQueue: requests.MultishiftNodeQueue{
						ControllerID: locationNameOrID,
						MultishiftNodeLabelFragment: requests.MultishiftNodeLabelFragment{
							Labels: labelsMap,
						},
					},
					Cluster:    clusterNameOrID,
					WorkerPool: workerPoolID,
					Zone:       host.Zone,
					HostID:     h.ID,
				}

				_, err = endpoint.CreateHostAssignment(config)
				if err != nil {
					return err
				}

				// Mark this host as assigned in the configuration file
				if satelliteControlPlane {
					tmp := locationConfig.Hosts.Control.Servers[h.Name]
					tmp.Cluster = "infrastructure"
					locationConfig.Hosts.Control.Servers[h.Name] = tmp
					fmt.Fprintf(c.App.Writer, "Assigned the host '%s' to location '%s' control plane\n", h.Name, location.Name)
				} else {
					_, cluster, err := endpoint.GetCluster(clusterNameOrID)
					if err != nil {
						fmt.Printf("WARNING : Cluster '%s' not found\n", clusterNameOrID)
					} else {
						tmp := locationConfig.Hosts.Cluster.Servers[h.Name]
						tmp.Cluster = cluster.Name()
						locationConfig.Hosts.Cluster.Servers[h.Name] = tmp
					}
					fmt.Fprintf(c.App.Writer, "Assigned the host '%s' to cluster '%s'\n", h.Name, cluster.Name())
				}
				cliutils.UpdateLocationInfrastructureConfig(locationConfig)

				hostsAssigned++
			}
		}

		fmt.Fprintln(c.App.Writer)

		if hostsAssigned > 0 && hostPollInterval > 0 {
			mt := metricsTime{pollInterval: hostPollInterval, startTime: time.Now(), timeout: timeout}

			fmt.Fprintln(c.App.Writer, "Monitoring Hosts\n----------------")
			err := monitorHosts(c, mt, hosts, hostOperationAssign)
			if err != nil {
				return err
			}

			// For the Satellite control plane, once all hosts have been succesfully assigned to the location,
			// there is a further period of time required for the location to be ready for deployments.
			if satelliteControlPlane {
				fmt.Fprintf(c.App.Writer, "Duration: %.1fs\n", time.Since(startTime).Seconds())
				fmt.Fprintln(c.App.Writer)

				fmt.Fprintln(c.App.Writer, "Monitoring Location\n-------------------")
				err = monitorLocation(c, mt)
				if err != nil {
					return err
				}
			}
		}

		fmt.Fprintf(c.App.Writer, "Total Duration: %.1fs\n", time.Since(startTime).Seconds())
		fmt.Fprintln(c.App.Writer)

		return nil
	}

	return fmt.Errorf("no available hosts attached to location '%s'", location.Name)
}

// Handle Satellite host attachment commands
// This will create, download and optionally run a script that makes a host(s) visible to a location.
func hostAttach(c *cli.Context) error {
	var hostPollInterval time.Duration
	var timeout time.Duration

	// Initialize metrics data
	metricsData := cliutils.GetMetadataValue(c, models.MetricsFlagName).(*metrics.Data)
	metricsData.Command = fmt.Sprintf("%s-host", strings.Split(cliutils.GetCommandName(c), " ")[1])
	metricsData.Enabled = cliutils.GetFlagBool(c, models.MetricsFlagName)

	// Initialize api endpoint
	endpoint := resources.GetArmadaV2Endpoint(c)

	satelliteConfig := cliutils.GetInfrastructureConfig().Satellite

	// Required. The name or ID of the location where you want to make compute hosts visible.
	locationNameOrID, err := cliutils.GetFlagString(c, models.LocationFlagName, false, "Location name or ID", nil)
	if err != nil {
		return err
	}

	_, location, _, err := endpoint.GetLocation(locationNameOrID)
	if err != nil {
		return err
	}

	// Should the script be automatically executed on the specified hosts ?
	automate := c.Bool(models.AutomateFlagName)
	controlPlane := c.Bool(models.ControlFlagName)

	// Get the location configuration data.
	var locationConfig cliutils.TomlSatelliteInfrastructureLocation
	var ok bool

	if locationConfig, ok = satelliteConfig.Locations[location.Name]; !ok {
		return fmt.Errorf("location '%s' not found in perf-infrastructure.toml configuration file", location.Name)
	}

	iaasType, err := cliutils.GetFlagString(c, models.InfrastructureTypeFlagName, true, "Infrastructure Type ('classic' or 'vpc-gen2'", nil)
	if err != nil {
		return err
	}

	privateKeyFilename, err := cliutils.GetFlagString(c, models.PrivateKeyFlagName, true, "Private Key file location", nil)
	if err != nil {
		return err
	}

	// Optional. A list of key-value-pair labels that you want to add to your compute hosts. Labels can help find hosts more easily later.
	labelsMap, err := cliutils.MakeLabelsMap(c)
	if err != nil {
		return err
	}

	hostPollIntervalStr, err := cliutils.GetFlagString(c, models.PollIntervalFlagName, true, "Worker status polling interval", nil)
	if err != nil {
		return err
	}
	if len(hostPollIntervalStr) > 0 {
		hostPollInterval, err = time.ParseDuration(hostPollIntervalStr)
		if err != nil {
			return err
		}
	}

	timeoutStr, err := cliutils.GetFlagString(c, models.TimeoutFlagName, true, "Host attach polling timeout", nil)
	if err != nil {
		return err
	}
	if len(timeoutStr) > 0 {
		timeout, err = time.ParseDuration(timeoutStr)
		if err != nil {
			return err
		}
	}

	if automate {
		if controlPlane {
			labelsMap["env"] = models.ControlLabel
		} else {
			labelsMap["env"] = models.ServiceLabel
		}
	}

	// Distinguish between control plane and cluster hosts attachment for metrics reporting
	metricsData.Command = fmt.Sprintf("%s.%s", metricsData.Command, labelsMap["env"])

	var hostConfig *cliutils.TomlSatelliteInfrastructureHostConfig

	osName, err := cliutils.GetFlagString(c, models.OperatingSystemFlagName, true, "Operating System Name", "")
	if err != nil {
		return err
	}

	if controlPlane {
		if iaasType != "" {
			locationConfig.Hosts.Control.IaasType = iaasType
		} else {
			locationConfig.Hosts.Control.IaasType = cliutils.GetDefaultLocation().Hosts.Control.IaasType
		}
		if osName != "" {
			locationConfig.Hosts.Control.OS = osName
		} else {
			locationConfig.Hosts.Control.OS = cliutils.GetDefaultLocation().Hosts.Control.OS
		}

		hostConfig = &locationConfig.Hosts.Control
	} else {
		if iaasType != "" {
			locationConfig.Hosts.Cluster.IaasType = iaasType
		} else {
			locationConfig.Hosts.Cluster.IaasType = cliutils.GetDefaultLocation().Hosts.Cluster.IaasType
		}
		if osName != "" {
			locationConfig.Hosts.Cluster.OS = osName
		} else {
			locationConfig.Hosts.Cluster.OS = cliutils.GetDefaultLocation().Hosts.Cluster.OS
		}

		hostConfig = &locationConfig.Hosts.Cluster
	}

	config := requests.MultishiftCreateScript{
		MultishiftNodeQueue: requests.MultishiftNodeQueue{
			ControllerID: locationNameOrID,
			MultishiftNodeLabelFragment: requests.MultishiftNodeLabelFragment{
				Labels: labelsMap,
			},
		},
		ResetHostKey: false,
	}
	if strings.HasPrefix(hostConfig.OS, "RHCOS") {
		config.OperatingSystem = "RHCOS"
	} else if strings.HasPrefix(hostConfig.OS, "REDHAT") {
		config.OperatingSystem = "RHEL"
	}

	startTime := time.Now()
	hostScript, err := endpoint.CreateHostScript(config)
	if err != nil {
		return err
	}

	var f *os.File
	var filename string
	if strings.HasPrefix(hostConfig.OS, "REDHAT") {
		filename = fmt.Sprintf("register-host_%s.sh", locationConfig.ID)
		f, err = os.OpenFile(filepath.Join("/tmp", filename), os.O_CREATE|os.O_WRONLY, 0744)
		if err != nil {
			return err
		}

		// RedHat subscription manager commands required for IBM Cloud infrastructure virtual servers
		if strings.HasPrefix(hostConfig.OS, "REDHAT_7") {
			f.WriteString("#!/usr/bin/env bash\n")
			f.WriteString("subscription-manager refresh\n")
			f.WriteString("subscription-manager repos --enable rhel-server-rhscl-7-rpms\n")
			f.WriteString("subscription-manager repos --enable rhel-7-server-optional-rpms\n")
			f.WriteString("subscription-manager repos --enable rhel-7-server-rh-common-rpms\n")
			f.WriteString("subscription-manager repos --enable rhel-7-server-supplementary-rpms\n")
			f.WriteString("subscription-manager repos --enable rhel-7-server-extras-rpms\n")
		} else if strings.HasPrefix(hostConfig.OS, "REDHAT_8") {
			f.WriteString("#!/usr/bin/env bash\n")
			f.WriteString("subscription-manager refresh\n")
			f.WriteString("subscription-manager repos --enable rhel-8-for-x86_64-appstream-rpms\n")
			f.WriteString("subscription-manager repos --enable rhel-8-for-x86_64-baseos-rpms\n")

			// Temporary workaround for bootstrap problem specific to RHEL_8
			f.WriteString("subscription-manager release --set=8\n")
			f.WriteString("subscription-manager repos --disable='*eus*'\n")
		}
	} else if strings.HasPrefix(hostConfig.OS, "RHCOS") {
		filename = filepath.Join("/tmp", fmt.Sprintf("register-host_%s.ign", locationConfig.ID))
		f, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0744)
		if err != nil {
			return err
		}
	}

	// Add the downloaded script
	if _, err := f.Write(hostScript.Bytes()); err != nil {
		return err
	}

	fmt.Fprintf(c.App.Writer, "\nHost registration script written to '%s'.", filename)
	fmt.Fprintln(c.App.Writer)

	if automate {
		var hostCount = -1

		quantityStr, err := cliutils.GetFlagString(c, models.QuantityFlagName, true, "Number of hosts to attach", nil)
		if err != nil {
			return err
		}
		if len(quantityStr) > 0 {
			hostCount, err = strconv.Atoi(quantityStr)
			if err != nil {
				return fmt.Errorf("invalid value for '--%s' specified. Enter a numerical value", models.QuantityFlagName)
			}
		}

		hostnameSuffixStr, err := cliutils.GetFlagString(c, models.SuffixFlagName, true, "Hostname suffix", nil)
		if err != nil {
			return err
		}

		var hosts map[string]cliutils.TomlSatelliteInfrastructureHost

		// If the location name matches a preconfigured location, we'll get the configured hosts.
		// Otherwise, we'll create the infrastructure hosts dynamically.
		if locationConfig.Preconfigured {
			hosts = hostConfig.Servers
		} else {
			hosts, err = createHosts(c, controlPlane, hostCount, hostnameSuffixStr, locationConfig)
			if err != nil {
				return err
			}
			hostCount = len(hosts)

			// Wait for hosts to be provisioned
			if hostCount > 0 {
				// Record details of the ordered hosts to simplify cleanup if there is a problem with the host provisioning.
				if locationConfig.Hosts.Provisioned.Servers == nil {
					locationConfig.Hosts.Provisioned.Servers = make(map[string]cliutils.TomlSatelliteInfrastructureHost)
				}
				for n, h := range hosts {
					locationConfig.Hosts.Provisioned.Servers[n] = h
				}

				var hostinfra HostInfrastructure

				locationConfig.Hosts.Provisioned.IaasType = hostConfig.IaasType
				locationConfig.Hosts.Provisioned.OS = hostConfig.OS
				cliutils.UpdateLocationInfrastructureConfig(locationConfig)

				switch hostConfig.IaasType {
				case "classic":
					// Create Classic Host
					hostinfra = NewSoftlayerVG(c)
				case "vpc-gen2":
					// Create VPC-Gen2 host
					var err error
					hostinfra, err = NewVPCService(c)
					if err != nil {
						return err
					}
				case "":
					return fmt.Errorf("Host infrastructure type not specified")
				default:
					return fmt.Errorf("Unknown host infrastructure type: %s", hostConfig.IaasType)
				}

				err = hostinfra.WaitForHosts(hosts)
				if err != nil {
					// Something went wrong checking host status with iaas provider. Let's just wait 10 minutes and hope for the best
					fmt.Fprintln(c.App.Writer, "Warning: Issues were encountered checking host status with IaaS provider. Waiting 10 minutes. Hosts may not be ready")
					time.Sleep(10 * time.Minute)
				}

				if delay := hostinfra.ProvisionDelay(); delay > 0 {
					// Add in a delay and adjust timing measurements
					fmt.Fprintf(c.App.Writer, "%s : Waiting %d minutes for %s infrastructure to complete updates before running registration script\n", time.Now().Format(time.Stamp), delay/time.Minute, hostinfra)
					time.Sleep(delay)
					fmt.Fprintln(c.App.Writer)
					startTime = startTime.Add(delay)
				}
			} else {
				fmt.Fprintf(c.App.Writer, "Requested number of hosts already attached/assigned\n")
			}
		}

		if hostCount < 0 {
			hostCount = len(hosts)
		}

		shc := sortedHosts(hosts)[0:hostCount]
		for i, name := range shc {
			if i == hostCount {
				break
			}

			fmt.Fprintf(c.App.Writer, "%s : Processing host '%s' on location '%s'\n", time.Now().Format(time.Stamp), name, location.Name)

			// RHEL requires the downloaded script to be copied and executed on the host
			if strings.HasPrefix(hostConfig.OS, "REDHAT") {
				err := attachHostToLocation(c, name, hostConfig.IaasType, hosts[name], privateKeyFilename, filepath.Join("/tmp", filename))
				if err != nil {
					return err
				}
			}

			if !locationConfig.Preconfigured {
				if hostConfig.Servers == nil {
					hostConfig.Servers = make(map[string]cliutils.TomlSatelliteInfrastructureHost)
				}
				// Host now attached to location, move from raw provisioned list
				hostConfig.Servers[name] = hosts[name]
				delete(locationConfig.Hosts.Provisioned.Servers, name)

				// Update the configuration file
				cliutils.UpdateLocationInfrastructureConfig(locationConfig)
			}

			fmt.Fprintln(c.App.Writer)
		}

		// If there are no provisioned hosts left that aren't attached to the location, reset the list
		if len(locationConfig.Hosts.Provisioned.Servers) == 0 {
			locationConfig.Hosts.Provisioned = cliutils.TomlSatelliteInfrastructureHostConfig{}
			cliutils.UpdateLocationInfrastructureConfig(locationConfig)
		}

		if hostPollInterval > 0 {
			mt := metricsTime{pollInterval: hostPollInterval, startTime: startTime, timeout: timeout}

			fmt.Fprintln(c.App.Writer, "Monitoring Hosts\n----------------")
			err := monitorHosts(c, mt, hosts, hostOperationAttach)
			if err != nil {
				return err
			}
		}
	}

	fmt.Fprintf(c.App.Writer, "Total Duration: %.1fs\n", time.Since(startTime).Seconds())
	fmt.Fprintln(c.App.Writer)

	return nil
}

// List the hosts in an IBM Cloud Satellite location
func hostList(c *cli.Context) error {
	var hostPollInterval time.Duration
	var timeout time.Duration

	locationNameOrID, err := cliutils.GetFlagString(c, models.LocationFlagName, false, "Location name or ID", nil)
	if err != nil {
		return err
	}

	hostPollIntervalStr, err := cliutils.GetFlagString(c, models.PollIntervalFlagName, true, "Worker status polling interval", nil)
	if err != nil {
		return err
	}
	if len(hostPollIntervalStr) > 0 {
		hostPollInterval, err = time.ParseDuration(hostPollIntervalStr)
		if err != nil {
			return err
		}
	}

	timeoutStr, err := cliutils.GetFlagString(c, models.TimeoutFlagName, true, "Host list polling timeout", nil)
	if err != nil {
		return err
	}
	if len(timeoutStr) > 0 {
		timeout, err = time.ParseDuration(timeoutStr)
		if err != nil {
			return err
		}
	}

	endpoint := resources.GetArmadaV2Endpoint(c)
	if hostPollInterval > 0 {
		err := monitorHosts(c,
			metricsTime{pollInterval: hostPollInterval, startTime: time.Now(), timeout: timeout},
			map[string]cliutils.TomlSatelliteInfrastructureHost{},
			hostOperationPoll,
		)
		if err != nil {
			return err
		}
	} else {
		rawJSON, hosts, err := endpoint.GetHosts(locationNameOrID)
		if err != nil {
			return err
		}

		jsonoutput := c.Bool(models.JSONOutFlagName)
		if jsonoutput {
			cliutils.WriteJSON(c, rawJSON)
			return nil
		}

		fmt.Fprintln(c.App.Writer)
		fmt.Fprintln(c.App.Writer, "Name\tID\tState\tStatus\tZone\tCluster ID\tCluster Name\tWorker ID\tWorker IP")
		fmt.Fprintln(c.App.Writer, "----\t--\t-----\t------\t----\t----------\t------------\t---------\t---------")

		for _, l := range hosts {
			fmt.Fprintf(c.App.Writer,
				"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				l.Name,
				l.ID,
				l.State,
				l.Health.Message,
				l.Assignment.Zone,
				l.Assignment.ClusterID,
				l.Assignment.ClusterName,
				l.Assignment.WorkerID,
				l.Assignment.IPAddress,
			)
		}
	}
	return nil
}

// Remove a set of host(s) from an IBM Cloud Satellite location or cluster
func hostRemove(c *cli.Context) error {
	satelliteConfig := cliutils.GetInfrastructureConfig().Satellite

	endpoint := resources.GetArmadaV2Endpoint(c)

	locationNameOrID, err := cliutils.GetFlagString(c, models.LocationFlagName, false, "Location name or ID", nil)
	if err != nil {
		return err
	}

	_, location, _, err := endpoint.GetLocation(locationNameOrID)
	if err != nil {
		return err
	}

	clusterName, err := cliutils.GetFlagString(c, models.ClusterFlagName, true, "Cluster name", nil)
	if err != nil {
		return err
	}

	requestedHosts := c.StringSlice(models.HostFlagName)
	controlPlane := c.Bool(models.ControlFlagName)
	provisioned := c.Bool(models.ProvisionedFlagName)

	reload := cliutils.GetFlagBool(c, models.ReloadFlagName)
	cancel := cliutils.GetFlagBool(c, models.CancelFlagName)
	forceCancellation := cliutils.GetFlagBool(c, models.ForceFlagName)

	// If hosts have been specified, then remove them, otherwise remove all cluster or control plane hosts
	hostsToRemove := make(map[string]cliutils.TomlSatelliteInfrastructureHost)
	for _, h := range requestedHosts {
		hostsToRemove[h] = cliutils.TomlSatelliteInfrastructureHost{}
	}

	// Get the location configutarion data.
	var locationConfig cliutils.TomlSatelliteInfrastructureLocation
	var hostConfig cliutils.TomlSatelliteInfrastructureHostConfig

	var locationConfigFound bool
	if locationConfig, locationConfigFound = satelliteConfig.Locations[location.Name]; !locationConfigFound {
		fmt.Printf("WARNING : Location '%s' not found in perf-infrastructure.toml configuration file\n", location.Name)
	} else {
		if provisioned {
			if !locationConfig.Preconfigured && len(locationConfig.Hosts.Provisioned.Servers) > 0 {
				var hostinfra HostInfrastructure

				switch locationConfig.Hosts.Provisioned.IaasType {
				case "classic":
					// Create Classic Host
					hostinfra = NewSoftlayerVG(c)
				case "vpc-gen2":
					// Create VPC-Gen2 host
					var err error
					hostinfra, err = NewVPCService(c)
					if err != nil {
						return err
					}
				default:
					return fmt.Errorf("Unknown host infrastructure type: %s", locationConfig.Hosts.Provisioned.IaasType)
				}

				for name, h := range locationConfig.Hosts.Provisioned.Servers {
					if cancel {
						// Send VM cancellation request
						if err := hostinfra.CancelHost(location.Name, name, h.IaasID); err != nil {
							fmt.Printf("WARNING : Unable to cancel host '%s' : %s\n", name, err.Error())
							continue
						}
						delete(locationConfig.Hosts.Provisioned.Servers, name)
						cliutils.UpdateLocationInfrastructureConfig(locationConfig)
					} else if reload {
						// Send OS reload request
						if err := hostinfra.ReloadHost(locationConfig, h.IaasID, name); err != nil {
							fmt.Printf("WARNING : Unable to reload host '%s' : %s\n", name, err.Error())
							continue
						}
					}
				}

				if len(locationConfig.Hosts.Provisioned.Servers) == 0 {
					locationConfig.Hosts.Provisioned = cliutils.TomlSatelliteInfrastructureHostConfig{}
					cliutils.UpdateLocationInfrastructureConfig(locationConfig)
				}
			}

			// Only processing reload/cancellation of provisioned hosts, nothing else to do
			return nil
		}

		if controlPlane {
			hostConfig = satelliteConfig.Locations[location.Name].Hosts.Control
		} else {
			hostConfig = satelliteConfig.Locations[location.Name].Hosts.Cluster
		}

		if len(hostsToRemove) == 0 {
			// Specific hosts were not specified, thus remove all cluster/control plane hosts
			if controlPlane {
				hostsToRemove = hostConfig.Servers
			} else {
				clusterHosts := hostConfig.Servers
				for n, v := range clusterHosts {
					if clusterName == "" || v.Cluster == clusterName {
						hostsToRemove[n] = v
					}
				}
			}
		}
	}

	if len(hostsToRemove) == 0 {
		var hostType string
		if controlPlane {
			hostType = "control plane"
		} else {
			hostType = "cluster"
		}
		fmt.Printf("No %s hosts found attached to location '%s'\n", hostType, location.Name)
		return nil
	}

	shc := sortedHosts(hostsToRemove)

	for _, name := range shc {
		// Documentation suggests that either name or ID can be used. Experience suggests we need the ID.
		_, h, err := endpoint.GetHost(locationNameOrID, name)
		if err != nil {
			return err
		}

		if len(h.ID) == 0 {
			fmt.Printf("WARNING : Requested host '%s' not attached to location\n", name)
			h.Name = name
		} else {
			if err := endpoint.RemoveHost(locationNameOrID, h.ID); err != nil {
				return err
			}

			fmt.Fprintf(c.App.Writer, "\nHost '%s' deletion request from location '%s' successful.\n", name, locationNameOrID)
		}

		if locationConfigFound {
			var hostInfra HostInfrastructure

			switch hostConfig.IaasType {
			case "classic":
				// Create Classic Host
				hostInfra = NewSoftlayerVG(c)
			case "vpc-gen2":
				// Create VPC-Gen2 host
				var err error
				hostInfra, err = NewVPCService(c)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("Unknown host infrastructure type: %s", locationConfig.Hosts.Provisioned.IaasType)
			}

			// For dynamic locations, remove the host from the configuration file
			// For preconfigured locations, remove the location/cluster association for the host
			if locationConfig.Preconfigured {
				// For simplicity we'll assume it's a control plane host, if not found then it must be assgined to a cluster
				if tmp, ok := locationConfig.Hosts.Control.Servers[h.Name]; ok {
					tmp.Cluster = ""
					locationConfig.Hosts.Control.Servers[h.Name] = tmp
				} else if tmp, ok := locationConfig.Hosts.Cluster.Servers[h.Name]; ok {
					tmp.Cluster = ""
					locationConfig.Hosts.Cluster.Servers[h.Name] = tmp
				}
			} else {
				// Note that delete is a no-op if the key doesn't exist, so just attempt deletion from all for simplicity.
				delete(locationConfig.Hosts.Control.Servers, h.Name)
				delete(locationConfig.Hosts.Cluster.Servers, h.Name)
			}
			cliutils.UpdateLocationInfrastructureConfig(locationConfig)

			// Handle virtual server reload/cancel requests, ignoring cancellation of preconfigured loction hosts, unless forced
			if !forceCancellation && locationConfig.Preconfigured {
				cancel = false
			}

			if cancel {
				// Send VM cancellation request
				if err := hostInfra.CancelHost(location.Name, h.Name, ""); err != nil {
					return err
				}
				delete(locationConfig.Hosts.Provisioned.Servers, h.Name)
				cliutils.UpdateLocationInfrastructureConfig(locationConfig)
			} else if reload {
				// Send OS reload request
				if err := hostInfra.ReloadHost(locationConfig, "", h.Name); err != nil {
					return err
				}

				// Reloading dynamic hosts, put back in the provisioned list
				if !locationConfig.Preconfigured {
					if controlPlane {
						locationConfig.Hosts.Provisioned.Servers[h.Name] = locationConfig.Hosts.Control.Servers[h.Name]
					} else {
						locationConfig.Hosts.Provisioned.Servers[h.Name] = locationConfig.Hosts.Cluster.Servers[h.Name]
					}
					cliutils.UpdateLocationInfrastructureConfig(locationConfig)
				}
			}
		}
	}

	// Clean up configuration, removing confiugration data for sections with no servers
	if len(locationConfig.Hosts.Control.Servers) == 0 {
		locationConfig.Hosts.Control = cliutils.TomlSatelliteInfrastructureHostConfig{}
	}
	if len(locationConfig.Hosts.Cluster.Servers) == 0 {
		locationConfig.Hosts.Cluster = cliutils.TomlSatelliteInfrastructureHostConfig{}
	}

	cliutils.UpdateLocationInfrastructureConfig(locationConfig)

	return nil
}

func hostGet(c *cli.Context) error {
	locationNameOrID, err := cliutils.GetFlagString(c, models.LocationFlagName, false, "Location name or ID", nil)
	if err != nil {
		return err
	}

	hostNameOrID, err := cliutils.GetFlagString(c, models.HostFlagName, false, "Host name or ID", nil)
	if err != nil {
		return err
	}

	endpoint := resources.GetArmadaV2Endpoint(c)

	rawJSON, host, err := endpoint.GetHost(locationNameOrID, hostNameOrID)
	if err != nil {
		return err
	}

	if host.ID != "" {
		jsonoutput := c.Bool(models.JSONOutFlagName)
		if jsonoutput {
			cliutils.WriteJSON(c, rawJSON)
			return nil
		}

		fmt.Fprintln(c.App.Writer)
		fmt.Fprintf(c.App.Writer, "%s\t%s\n", "ID:", host.ID)
		fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Name:", host.Name)
		fmt.Fprintf(c.App.Writer, "%s\t%s\n", "State:", host.State)
		fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Status:", host.Health.Status)
		fmt.Fprintln(c.App.Writer, "Assignment")
		fmt.Fprintf(c.App.Writer, "\t%s\t%s\n", "Cluster Name:", host.Assignment.ClusterName)
		fmt.Fprintf(c.App.Writer, "\t%s\t%s\n", "Cluster ID:", host.Assignment.ClusterID)
		fmt.Fprintf(c.App.Writer, "\t%s\t%s\n", "Worker Pool:", host.Assignment.WorkerPoolName)
		fmt.Fprintf(c.App.Writer, "\t%s\t%s\n", "Zone:", host.Assignment.Zone)
		fmt.Fprintf(c.App.Writer, "\t%s\t%s\n", "Worker ID:", host.Assignment.WorkerID)
		fmt.Fprintf(c.App.Writer, "\t%s\t%s\n", "Worker IP:", host.Assignment.IPAddress)
		fmt.Fprintf(c.App.Writer, "\t%s\t%s\n", "Requested Date:", host.Assignment.RequestedDate)
		fmt.Fprintf(c.App.Writer, "\t%s\t%s\n", "Received Date:", host.Assignment.ReceivedDate)

		if len(host.Labels) > 0 {
			fmt.Fprintln(c.App.Writer, "Labels")
			for k, v := range host.Labels {
				fmt.Fprintf(c.App.Writer, "\t%s\t%s\n", k, v)
			}
			return nil
		}
	}

	return fmt.Errorf("requested host '%s' not found", hostNameOrID)
}

func hostUpdate(c *cli.Context) error {
	locationNameOrID, err := cliutils.GetFlagString(c, models.LocationFlagName, false, "Location name or ID", nil)
	if err != nil {
		return err
	}

	hostNameOrID, err := cliutils.GetFlagString(c, models.HostFlagName, false, "Host name or ID", nil)
	if err != nil {
		return err
	}

	// Optional. A list of key-value-pair labels that you want to add to your compute hosts. Labels can help find hosts more easily later.
	labelsMap, err := cliutils.MakeLabelsMap(c)
	if err != nil {
		return err
	}

	endpoint := resources.GetArmadaV2Endpoint(c)

	// Documentation suggests that either name or ID can be used. Experience suggests we need the ID.
	_, h, err := endpoint.GetHost(locationNameOrID, hostNameOrID)
	if err != nil {
		return err
	}

	// Build the config object
	config := requests.MultishiftUpdateNode{
		MultishiftNodeQueue: requests.MultishiftNodeQueue{
			ControllerID: locationNameOrID,
			MultishiftNodeLabelFragment: requests.MultishiftNodeLabelFragment{
				Labels: labelsMap,
			},
		},
		NodeID: h.ID,
	}

	err = endpoint.UpdateHost(config)
	if err != nil {
		return err
	}

	return nil
}

func createHosts(c *cli.Context, control bool, quantity int, hostnameSuffix string, locationConfig cliutils.TomlSatelliteInfrastructureLocation) (map[string]cliutils.TomlSatelliteInfrastructureHost, error) {
	var hostConfig cliutils.TomlSatelliteInfrastructureHostConfig
	var hostType, iaasType, osName string
	var hi int

	var hostinfra HostInfrastructure

	fmt.Fprintf(c.App.Writer, "\n%s : Creating IBM Cloud infrastructure hosts\n", time.Now().Format(time.Stamp))

	if len(hostnameSuffix) > 0 {
		hostnameSuffix = hostnameSuffix + "-"
	}

	if control {
		hostType = "control"
		iaasType = locationConfig.Hosts.Control.IaasType
		osName = locationConfig.Hosts.Control.OS

		currentHostCount := len(locationConfig.Hosts.Control.Servers)
		hi = currentHostCount + 1

		if len(locationConfig.Hosts.Control.Servers) > 0 {
			hostConfig = locationConfig.Hosts.Control

			if quantity > currentHostCount {
				quantity -= currentHostCount
				hostConfig = cliutils.GetDefaultLocation().Hosts.Control
			}
		} else {
			hostConfig = cliutils.GetDefaultLocation().Hosts.Control
		}
	} else {
		hostType = "worker"
		iaasType = locationConfig.Hosts.Cluster.IaasType
		osName = locationConfig.Hosts.Cluster.OS

		currentHostCount := len(locationConfig.Hosts.Cluster.Servers)
		hi = currentHostCount + 1

		if len(locationConfig.Hosts.Cluster.Servers) > 0 {
			hostConfig = locationConfig.Hosts.Cluster

			if quantity > currentHostCount {
				quantity -= currentHostCount
				hostConfig = cliutils.GetDefaultLocation().Hosts.Cluster
			}
		} else {
			hostConfig = cliutils.GetDefaultLocation().Hosts.Cluster
		}
	}

	hostConfig.IaasType = iaasType
	hostConfig.OS = osName

	// Setup the infrastructure provider interface
	switch iaasType {
	case "classic":
		// Create Classic hosts
		hostinfra = NewSoftlayerVG(c)
	case "vpc-gen2":
		// Create VPC-Gen2 hosts
		var err error
		hostinfra, err = NewVPCService(c)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(hostConfig.OS, "RHCOS") {
			b, err := ioutil.ReadFile(fmt.Sprintf("/tmp/register-host_%s.ign", locationConfig.ID))
			if err != nil {
				return nil, err
			}
			hostinfra.SetUserData(string(b))
		}

	default:
		return nil, fmt.Errorf("Unknown host infrastructure type: %s", iaasType)
	}

	hosts := make(map[string]cliutils.TomlSatelliteInfrastructureHost)

	// Sort the host configuration based on the hostname
	shc := sortedHosts(hostConfig.Servers)

	for _, sh := range shc {
		hc := hostConfig.Servers[sh]

		// If the host doesn't exist already
		if hc.IaasID == "" {
			if quantity >= 0 {
				hc.Quantity = quantity
			}

			for i := 1; i <= hc.Quantity; i++ {
				if len(hostConfig.Servers[sh].Hostname) == 0 {
					hc.Hostname = fmt.Sprintf("arm-perf-%s-%s-%s%d", hc.Datacenter, hostType, hostnameSuffix, hi)
				}

				var vh cliutils.TomlSatelliteInfrastructureHost
				var err error

				vh, err = hostinfra.CreateHost(locationConfig.Name, hc.Hostname, hostConfig, hc)
				if err != nil {
					return nil, err
				}

				// We might not have created any hosts (e.g. config is not for this infrastructure type)
				if vh.IaasID != "" {
					hi++
					hosts[vh.Hostname] = vh // Make createHost calls interface and have a common hostname
				}
			}
		}
	}

	return hosts, nil
}

// FindHost seaches all supported IaaS providers for the specified loction/host
func FindHost(c *cli.Context, location string, hostname string) (cliutils.TomlSatelliteInfrastructureHost, string, error) {
	var host cliutils.TomlSatelliteInfrastructureHost

	// Generate list of supported Iaas providers
	hostinfra := []HostInfrastructure{NewSoftlayerVG(c)}
	vpci, err := NewVPCService(c)
	if err != nil {
		return host, "", err
	}
	hostinfra = append(hostinfra, vpci)

	// Search throguh supported providers, stop on first match
	for _, hi := range hostinfra {
		host, _ = hi.GetHost(location, hostname, "")
		if host.IaasID != "" {
			return host, hi.String(), nil
		}
	}

	return host, "", nil
}
