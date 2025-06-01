

// Package cmdlocation registers and executes 'ibmcloud sat location ...' commands.
package cmdlocation

import (
	"fmt"
	"strings"
	"time"

	"github.com/IBM-Cloud/ibm-cloud-cli-sdk/i18n"
	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-api-model/json/v2/requests"
	"github.ibm.com/alchemy-containers/armada-api-model/json/v2/responses"
	"github.ibm.com/alchemy-containers/armada-model/model"
	"github.ibm.com/alchemy-containers/armada-model/model/multishift"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cmdhost"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/flag"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/metrics"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

// RegisterCommands registers the Satellite host commands
func RegisterCommands() []cli.Command {
	requiredLocationNameFlag := flag.StringFlag{
		Require: true,
		StringFlag: cli.StringFlag{
			Name:  models.NameFlagName,
			Usage: "Name of the location to be created.",
		},
	}

	return []cli.Command{
		{
			Name:        models.NamespaceLocation,
			Description: "Create, view, and modify Satellite locations.",
			Usage:       "Create, view, and modify Satellite locations.",
			Subcommands: []cli.Command{
				{
					Name:        models.CmdCreate,
					Description: "Create a Satellite location.",
					Flags: []cli.Flag{
						requiredLocationNameFlag,
						models.RequiredManagedFromFlag,
						models.CoreOSEnabledFlag,
						models.MetricsFlag,
						models.PollIntervalFlag,
						models.TimeoutFlag,
					},
					Action: locationCreate,
				},
				{
					Name:        models.CmdGet,
					Description: "View the details of a Satellite location.",
					Usage:       "View the details of a Satellite location.",
					Flags: []cli.Flag{
						models.RequiredLocationFlag,
						models.JSONOutFlag,
					},
					Action: locationGet,
				},
				{
					Name:        models.CmdList,
					Description: "List all Satellite locations in your IBM Cloud account.",
					Usage:       "List all Satellite locations in your IBM Cloud account.",
					Flags: []cli.Flag{
						models.JSONOutFlag,
					},
					Action: locationList,
				},
				{
					Name:        models.CmdRemove,
					Description: "Delete a Satellite location. All worker nodes, apps, and containers are permanently deleted. This action cannot be undone.",
					Usage:       "Delete a Satellite location. All worker nodes, apps, and containers are permanently deleted. This action cannot be undone.",
					Flags: []cli.Flag{
						models.RequiredLocationFlag,
						models.QuantityFlag,
						models.MetricsFlag,
						models.PollIntervalFlag,
						models.TimeoutFlag,
						models.SuffixFlag},
					Action: locationRemove,
				},
				{
					Name:        models.CmdSync,
					Description: "Synchronize Satellite configuration file.",
					Usage:       "Synchronize Satellite configuration file.",
					Flags: []cli.Flag{
						models.RequiredLocationFlag,
					},
					Action: locationSync,
				},
				{
					Name:        models.NamespaceDNS,
					Description: "Create and manage subdomains for the hosts assigned to the control plane in a Satellite location.",
					Usage:       "Create and manage subdomains for the hosts assigned to the control plane in a Satellite location.",
					Subcommands: []cli.Command{
						{
							Name:        models.CmdRegister,
							Description: i18n.T("Create a subdomain for the hosts assigned to the control plane in a Satellite location."),
							Flags: []cli.Flag{
								models.RequiredLocationFlag,
								flag.StringSliceFlag{
									Require: true,
									Repeat:  true,
									StringSliceFlag: cli.StringSliceFlag{
										Name:  models.IPFlagName,
										Usage: "The public IP address for the control plane host. Specify three host IP addresses, in repeated flags. For multizone clusters, use one IP address from each zone.",
									},
								},
								models.JSONOutFlag,
							},
							Action: locationSubdomainRegister,
						},
						{
							Name:        models.CmdList,
							Description: "List the registered NLB subdomains and corresponding A records or CNAME records in a Satellite location.",
							Flags: []cli.Flag{
								models.RequiredLocationFlag,
								models.JSONOutFlag,
							},
							Action: locationSubdomainList,
						},
						{
							Name:        models.CmdGet,
							Description: i18n.T("View the details of a registered subdomain in a Satellite location."),
							Flags: []cli.Flag{
								models.RequiredLocationFlag,
								models.RequiredDNSSubdomainFlag,
								models.JSONOutFlag,
							},
							Action: locationSubdomainGet,
						},
					},
				},
			},
		},
	}
}

func locationCreate(c *cli.Context) error {
	var locationPollInterval time.Duration
	var timeout time.Duration

	// Initialize metrics data
	metricsData := cliutils.GetMetadataValue(c, models.MetricsFlagName).(*metrics.Data)
	metricsData.Command = fmt.Sprintf("%s-location", strings.Split(cliutils.GetCommandName(c), " ")[1])
	metricsData.Enabled = cliutils.GetFlagBool(c, models.MetricsFlagName)

	pollIntervalStr, err := cliutils.GetFlagString(c, models.PollIntervalFlagName, true, "Cluster status polling interval", nil)
	if err != nil {
		return err
	}
	if len(pollIntervalStr) > 0 {
		locationPollInterval, err = time.ParseDuration(pollIntervalStr)
		if err != nil {
			return err
		}
	}

	timeoutStr, err := cliutils.GetFlagString(c, models.TimeoutFlagName, true, "Location creation polling timeout", nil)
	if err != nil {
		return err
	}
	if len(timeoutStr) > 0 {
		timeout, err = time.ParseDuration(timeoutStr)
		if err != nil {
			return err
		}
	}

	name, err := cliutils.GetFlagString(c, models.NameFlagName, false, "Name of the location to be created.", nil)
	if err != nil {
		return err
	}

	locationConfig := getLocationConfig(name)

	managedFrom, err := cliutils.GetFlagString(c, models.ManagedFromFlagName, false, "Management metro name", locationConfig.ManagedFrom)
	if err != nil {
		return err
	}

	var coreosEnabled bool
	if locationConfig.CoresOSEnabled {
		coreosEnabled = cliutils.GetFlagBoolT(c, models.CoreOSEnabledFlagName)
	} else {
		coreosEnabled = cliutils.GetFlagBool(c, models.CoreOSEnabledFlagName)
	}

	// Build the config object
	config := requests.MultishiftCreateController{
		Location:         managedFrom,
		Name:             name,
		Description:      "",
		LoggingAccountID: "",
		Zones:            []string{},
		CoreOSEnabled:    coreosEnabled,
		IAAS:             requests.IAAS{Provider: multishift.IaasProviderTypeIBMCloud},
		COSConfig:        requests.COSBucket{},
		COSCredentials:   requests.COSAuthorization{},
	}

	endpoint := resources.GetArmadaV2Endpoint(c)

	startTime := time.Now()
	locationID, err := endpoint.CreateLocation(config)
	if err != nil {
		return err
	}

	// Update the configuration file for this location
	// For new, dynamic locations, add the record
	// For preconfigured locations, update the location ID
	if !locationConfig.Preconfigured {
		// New location - create record
		locationConfig = cliutils.TomlSatelliteInfrastructureLocation{
			Name:           name,
			ID:             locationID,
			ManagedFrom:    managedFrom,
			CoresOSEnabled: coreosEnabled,
			Preconfigured:  false,
		}
	} else {
		// Preconfigured location - update the ID
		locationConfig.ID = locationID
	}

	cliutils.UpdateLocationInfrastructureConfig(locationConfig)

	fmt.Fprintf(c.App.Writer, "\nLocation creation request for '%s' successful. ID: '%s'\n", name, locationID)

	if locationPollInterval > 0 {
		locationComplete := false
		previousState := ""
		previousMessage := ""
		for !locationComplete {
			pollRequestTime := time.Now()

			if timeout > 0 && time.Since(startTime) > timeout {
				return fmt.Errorf("timeout waiting for location creation to complete. Timeout: %v", timeout)
			}

			_, location, _, err := endpoint.GetLocation(locationID)
			if err != nil {
				return err
			}

			if (location.State != previousState) || (location.Deployments.Message != previousMessage) {
				fmt.Fprintln(c.App.Writer, "\n", pollRequestTime.Format(time.Stamp))
				fmt.Fprintf(c.App.Writer, "\t%s : \"%s\", \"%s\"\n", location.Name, location.State, location.Deployments.Message)
				fmt.Fprintln(c.App.Writer)

				previousState = location.State
				previousMessage = location.Deployments.Message
			}

			locationComplete = !strings.EqualFold(location.State, *model.ClusterActualStateDeploying)
			if !locationComplete {
				time.Sleep(locationPollInterval)
			} else {
				if strings.EqualFold(location.State, *model.ClusterActualStateDeleteFailed) {
					return fmt.Errorf("Location create failed. Current state: %v", location.State)
				}
			}
		}
	}

	totalTime := time.Since(startTime)
	metricsData.L = make(map[string]*metrics.Location)
	metricsData.L[name] = &metrics.Location{Duration: totalTime, CoreOS: coreosEnabled}

	fmt.Fprintf(c.App.Writer, "Total Duration: %.1fs\n", totalTime.Seconds())

	return nil
}

func locationGet(c *cli.Context) error {
	locationNameOrID, err := cliutils.GetFlagString(c, models.LocationFlagName, false, "Location name or ID", nil)
	if err != nil {
		return err
	}

	endpoint := resources.GetArmadaV2Endpoint(c)

	rawJSON, location, _, err := endpoint.GetLocation(locationNameOrID)
	if err != nil {
		return err
	}

	jsonoutput := c.Bool(models.JSONOutFlagName)
	if jsonoutput {
		cliutils.WriteJSON(c, rawJSON)
		return nil
	}

	fmt.Fprintln(c.App.Writer)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "ID:", location.ID)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Name:", location.Name)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Created:", location.CreatedDate)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Managed From:", location.Location)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "State:", location.State)
	fmt.Fprintf(c.App.Writer, "%s\t%t\n", "Ready for deployments:", location.Deployments.Enabled)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Message:", location.Deployments.Message)
	fmt.Fprintf(c.App.Writer, "%s\t%d\n", "Hosts Available:", location.Hosts.Available)
	fmt.Fprintf(c.App.Writer, "%s\t%d\n", "Hosts Total:", location.Hosts.Total)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Host Zones:", location.WorkerZones)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Provider:", location.IAAS.Provider)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Provider Region:", location.IAAS.Region)
	fmt.Fprintf(c.App.Writer, "%s\t%t\n", "CoresOS Enabled:", location.CoreOSEnabled)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Public Service Endpoint URL:", location.ServiceEndpoints.PublicServiceEndpointURL)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Private Service Endpoint URL:", location.ServiceEndpoints.PrivateServiceEndpointURL)
	fmt.Fprintf(c.App.Writer, "%s\t%d\n", "OpenVPN Server Port:", location.OpenVPNServerPort)
	fmt.Fprintf(c.App.Writer, "%s\t%d\n", "Ignition Server Port:", location.IgnitionServerPort)
	fmt.Fprintf(c.App.Writer, "%s\t%d\n", "Konnectivity Server Port:", location.KonnectivityServerPort)

	return nil
}

func locationList(c *cli.Context) error {
	endpoint := resources.GetArmadaV2Endpoint(c)

	rawJSON, locations, err := endpoint.GetLocations()
	if err != nil {
		return err
	}

	jsonoutput := c.Bool(models.JSONOutFlagName)
	if jsonoutput {
		cliutils.WriteJSON(c, rawJSON)
		return nil
	}

	fmt.Fprintln(c.App.Writer)
	fmt.Fprintln(c.App.Writer, "Name\tID\tStatus\tReady\tCreated\tHosts (used/total)\tManaged From\tCoresOS Enabled")
	fmt.Fprintln(c.App.Writer, "----\t--\t------\t-----\t-------\t------------------\t------------\t---------------")

	for _, l := range locations {
		fmt.Fprintf(c.App.Writer,
			"%s\t%s\t%s\t%t\t%s\t%d / %d\t%s\t%t\n",
			l.Name,
			l.ID,
			l.State,
			l.Deployments.Enabled,
			l.CreatedDate,
			l.Hosts.Total-l.Hosts.Available,
			l.Hosts.Total,
			l.ManagedFrom,
			l.CoreOSEnabled,
		)
	}
	return nil
}

func locationRemove(c *cli.Context) error {
	var locationPollInterval time.Duration
	var timeout time.Duration

	// Initialize metrics data
	metricsData := cliutils.GetMetadataValue(c, models.MetricsFlagName).(*metrics.Data)
	metricsData.Command = fmt.Sprintf("%s-location", strings.Split(cliutils.GetCommandName(c), " ")[1])
	metricsData.Enabled = cliutils.GetFlagBool(c, models.MetricsFlagName)

	locationNameOrID, err := cliutils.GetFlagString(c, models.LocationFlagName, false, "Location name or ID", nil)
	if err != nil {
		return err
	}

	// Flag to request that a numerical suffix is added to the specified location name
	// Note that if the specified quantity is more than one, a numerical suffix will be automatically added.
	suffix := cliutils.GetFlagBool(c, models.SuffixFlagName)

	quantity, err := cliutils.GetFlagInt(c, models.QuantityFlagName, true, "Number of locations", nil)
	if err != nil {
		return err
	}

	pollIntervalStr, err := cliutils.GetFlagString(c, models.PollIntervalFlagName, true, "Location status polling interval", nil)
	if err != nil {
		return err
	}
	if len(pollIntervalStr) > 0 {
		locationPollInterval, err = time.ParseDuration(pollIntervalStr)
		if err != nil {
			return err
		}
	}

	timeoutStr, err := cliutils.GetFlagString(c, models.TimeoutFlagName, true, "Location deletion polling timeout", nil)
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

	locationsForDeletion := make(map[string]bool)

	startTime := time.Now()

	// Request the location deletion(s)
	for l := 1; l <= quantity; l++ {
		var loc = locationNameOrID

		// Add on the numerical suffix if appropriate
		if suffix || quantity > 1 {
			loc = fmt.Sprintf("%s%d", loc, l)
		}

		_, location, _, err := endpoint.GetLocation(loc)
		if err != nil {
			return err
		}

		// Now request the deletion of the location
		if err := endpoint.RemoveLocation(location.ID); err != nil {
			return err
		}

		locationsForDeletion[location.Name] = true

		fmt.Fprintf(c.App.Writer, "Location deletion request for '%s' successful.\n", location.Name)

		// Remove location from the configuration file if it isn't associated with a preconfigured location; otherwise just clear the ID
		locationConfig := getLocationConfig(location.Name)
		satelliteConfig := cliutils.GetInfrastructureConfig().Satellite
		if !locationConfig.Preconfigured {
			delete(satelliteConfig.Locations, location.Name)
			cliutils.UpdateSatelliteInfrastructureConfig(satelliteConfig)
		} else {
			locationConfig.ID = ""
			cliutils.UpdateLocationInfrastructureConfig(locationConfig)
		}
	}

	// If we're interested in timings, wait for the location(s) to be deleted.
	if locationPollInterval > 0 {
		locationsDeleted := false
		matchedLocations := 0

		fmt.Fprintln(c.App.Writer)

		for !locationsDeleted {
			var locationsOfInterest []responses.MultishiftController

			locationsDeleted = true

			if timeout > 0 && time.Since(startTime) > timeout {
				return fmt.Errorf("timeout waiting for location deletion to complete. Timeout: %v", timeout)
			}

			// Get all locations we have access to
			_, locations, err := endpoint.GetLocations()
			if err != nil {
				time.Sleep(locationPollInterval)
				continue
			}

			// Construct a list of locations that we are deleting
			for _, loc := range locations {
				_, idMatch := locationsForDeletion[loc.ID]
				_, nameMatch := locationsForDeletion[loc.Name]
				if idMatch || nameMatch {
					locationsOfInterest = append(locationsOfInterest, loc)
					locationsDeleted = false
				}
			}

			// Report successfully deleted locations
			if matchedLocations == 0 || matchedLocations != len(locationsOfInterest) {
				if matchedLocations != 0 {
					metricsData.L = make(map[string]*metrics.Location)
					metricsData.L[locationNameOrID] = &metrics.Location{Duration: time.Since(startTime)}
				}

				if len(locationsOfInterest) > 0 {
					fmt.Fprintln(c.App.Writer, time.Now().Format(time.Stamp))
					for _, loc := range locationsOfInterest {
						fmt.Fprintf(c.App.Writer, "\t%s : \"%s\"\n", loc.Name, loc.State)
					}
					fmt.Fprintln(c.App.Writer)
				}
			}

			matchedLocations = len(locationsOfInterest)

			if !locationsDeleted {
				time.Sleep(locationPollInterval)
			}
		}
	}

	fmt.Fprintf(c.App.Writer, "Total Duration: %.1fs\n", time.Since(startTime).Seconds())

	return nil
}

func locationSync(c *cli.Context) error {
	locationName, err := cliutils.GetFlagString(c, models.LocationFlagName, false, "Location name", nil)
	if err != nil {
		return err
	}

	// Get the location details via the api
	endpoint := resources.GetArmadaV2Endpoint(c)
	_, location, notFound, err := endpoint.GetLocation(locationName)
	if err != nil {
		// Handle locations that don't exist
		if notFound {
			locationConfig := getLocationConfig(locationName)
			if locationConfig.Name == locationName {
				fmt.Fprintf(c.App.Writer, "\nSynchronizing configuration data for location '%s'\n", locationName)

				if !locationConfig.Preconfigured {
					satelliteConfig := cliutils.GetInfrastructureConfig().Satellite
					delete(satelliteConfig.Locations, locationName)
					cliutils.UpdateSatelliteInfrastructureConfig(satelliteConfig)
				} else {
					locationConfig.ID = ""
					cliutils.UpdateLocationInfrastructureConfig(locationConfig)
				}
				return nil
			}
			fmt.Fprintf(c.App.Writer, "Location '%s' not found\n", locationName)
			return nil
		}
		return err
	}

	locationConfig := getLocationConfig(location.Name)

	// For preconfigured locations that exist, just make sure the location id is correct
	if locationConfig.Preconfigured {
		locationConfig.ID = location.ID
		cliutils.UpdateLocationInfrastructureConfig(locationConfig)
		return nil
	}

	// Get the location's host details via the api
	_, hosts, err := endpoint.GetHosts(location.ID)
	if err != nil {
		return err
	}

	// Create a new location configuration object and populate with location details
	nlc := cliutils.TomlSatelliteInfrastructureLocation{
		Name:           location.Name,
		ID:             location.ID,
		ManagedFrom:    location.Location,
		CoresOSEnabled: location.CoreOSEnabled,
		Preconfigured:  locationConfig.Preconfigured,
	}

	// Initilaize host data for the new location config
	nlc.Hosts.Provisioned = locationConfig.Hosts.Provisioned
	nlc.Hosts.Provisioned.Servers = make(map[string]cliutils.TomlSatelliteInfrastructureHost)
	nlc.Hosts.Control = locationConfig.Hosts.Control
	nlc.Hosts.Control.Servers = make(map[string]cliutils.TomlSatelliteInfrastructureHost)
	nlc.Hosts.Cluster = locationConfig.Hosts.Cluster
	nlc.Hosts.Cluster.Servers = make(map[string]cliutils.TomlSatelliteInfrastructureHost)

	fmt.Fprintf(c.App.Writer, "\nSynchronizing configuration data for location '%s'\n", location.Name)

	// Loop through each host returned from the api and populate the new host configuration object with their details
	for _, h := range hosts {
		nh := cliutils.TomlSatelliteInfrastructureHost{
			Hostname: h.Name,
			Quantity: 1,
			Cluster:  h.Assignment.ClusterName,
			IP:       h.Assignment.IPAddress,
			Zone:     h.Assignment.Zone,
		}

		var dhc, ohc cliutils.TomlSatelliteInfrastructureHostConfig
		var nhc *cliutils.TomlSatelliteInfrastructureHostConfig

		switch h.Labels["env"] {
		case models.ControlLabel:
			dhc = getLocationConfig("default").Hosts.Control
			ohc = locationConfig.Hosts.Control
			nhc = &nlc.Hosts.Control
		case models.ServiceLabel:
			dhc = getLocationConfig("default").Hosts.Cluster
			ohc = locationConfig.Hosts.Cluster
			nhc = &nlc.Hosts.Cluster
		default:
		}

		nhc.OS = h.Labels["os"]

		// For metadata that isn't stored/returned by Satellite api, update the new host configuration with details from Iaas provider
		if ih, iaasType, err := cmdhost.FindHost(c, location.Name, h.Name); err == nil {
			nh.IaasID = ih.IaasID
			nh.Datacenter = ih.Datacenter
			nh.CPU = ih.CPU
			nh.Disk = ih.Disk
			nh.Memory = ih.Memory
			nh.Classic = ih.Classic
			nh.VPC = ih.VPC

			if len(nh.IP) == 0 {
				nh.IP = ih.IP
			}

			if len(nh.Zone) == 0 {
				if nh.Zone = ohc.Servers[h.Name].Zone; len(nh.Zone) == 0 {
					nh.Zone = dhc.Servers[nh.Datacenter].Zone
				}
			}

			nhc.IaasType = iaasType
		} else {
			fmt.Fprintf(c.App.Writer, "WARNING: Failed to obtain details from infrastructure service for host '%s' in location '%s' : %s\n", h.Name, location.Name, err.Error())
		}

		nhc.Servers[h.Name] = nh
	}

	// Finally, add any hosts provisioned for this location which aren't attached/asigned
	var hostinfra []cmdhost.HostInfrastructure
	classicIaas := cmdhost.NewSoftlayerVG(c)
	vpcIaas, _ := cmdhost.NewVPCService(c)
	hostinfra = append(hostinfra, classicIaas, vpcIaas)

	for _, hi := range hostinfra {
		lh, err := hi.GetLocationHosts(location.Name)
		if err != nil {
			return err
		}

		for name, id := range lh {
			if _, ok := nlc.Hosts.Control.Servers[name]; !ok {
				if _, ok := nlc.Hosts.Cluster.Servers[name]; !ok {
					nlc.Hosts.Provisioned.IaasType = hi.String()
					nlc.Hosts.Provisioned.Servers[name] = cliutils.TomlSatelliteInfrastructureHost{IaasID: id}
				}
			}
		}
	}

	if len(nlc.Hosts.Provisioned.Servers) == 0 {
		nlc.Hosts.Provisioned = cliutils.TomlSatelliteInfrastructureHostConfig{}
	}
	if len(nlc.Hosts.Control.Servers) == 0 {
		nlc.Hosts.Control = cliutils.TomlSatelliteInfrastructureHostConfig{}
	}
	if len(nlc.Hosts.Cluster.Servers) == 0 {
		nlc.Hosts.Cluster = cliutils.TomlSatelliteInfrastructureHostConfig{}
	}
	cliutils.UpdateLocationInfrastructureConfig(nlc)

	return nil
}

func locationSubdomainRegister(c *cli.Context) error {
	locationNameOrID, err := cliutils.GetFlagString(c, models.LocationFlagName, false, "Location name or ID", nil)
	if err != nil {
		return err
	}

	ips := c.StringSlice(models.IPFlagName)
	if len(ips) != 3 {
		errMessageBadIP := "You must specify three IP addresses"
		return cliutils.IncorrectUsageError(c, errMessageBadIP)
	}

	fmt.Fprintln(c.App.Writer, "Registering a subdomain for control plane hosts...")
	endpoint := resources.GetArmadaV2Endpoint(c)

	registrationResp, err := endpoint.RegisterLocationSubdomains(locationNameOrID, ips)
	if err != nil {
		return err
	}

	jsonoutput := c.Bool(models.JSONOutFlagName)
	if jsonoutput {
		cliutils.WriteJSON(c, registrationResp)
		return nil
	}

	fmt.Fprintln(c.App.Writer)
	fmt.Fprintln(c.App.Writer, "Hostname\tRecords\tSSL Cert Status\tSSL Cert Secret Name\tSecret Namespace\tStatus")
	fmt.Fprintln(c.App.Writer, "--------\t-------\t---------------\t---------------------\t---------------\t------")

	for _, domainRegistration := range registrationResp.DNSRegistrations {
		fmt.Fprintf(c.App.Writer,
			"%s\t%s\n",
			domainRegistration.Subdomain,
			strings.Join(domainRegistration.IPs, ", "),
		)
	}

	return nil
}
func locationSubdomainList(c *cli.Context) error {
	locationNameOrID, err := cliutils.GetFlagString(c, models.LocationFlagName, false, "Location name or ID", nil)
	if err != nil {
		return err
	}

	endpoint := resources.GetArmadaV2Endpoint(c)

	rawJSON, subdomains, err := endpoint.GetLocationSubdomains(locationNameOrID)
	if err != nil {
		return err
	}

	jsonoutput := c.Bool(models.JSONOutFlagName)
	if jsonoutput {
		cliutils.WriteJSON(c, rawJSON)
		return nil
	}

	fmt.Fprintln(c.App.Writer)
	fmt.Fprintln(c.App.Writer, "Hostname\tRecords\tSSL Cert Status\tSSL Cert Secret Name\tSecret Namespace\tStatus")
	fmt.Fprintln(c.App.Writer, "--------\t-------\t---------------\t---------------------\t---------------\t------")

	for _, sd := range subdomains {
		fmt.Fprintf(c.App.Writer,
			"%s\t%s\t%s\t%s\t%s\n",
			sd.NlbHost,
			strings.Join(sd.NlbIPArray, ","),
			sd.NlbSslSecretStatus,
			sd.NlbSSLSecretName,
			sd.NlbStatusMessage,
		)
	}

	return nil
}
func locationSubdomainGet(c *cli.Context) error {
	locationNameOrID, err := cliutils.GetFlagString(c, models.LocationFlagName, false, "Location name or ID", nil)
	if err != nil {
		return err
	}

	nlbDNSSubdomain, err := cliutils.GetFlagString(c, models.DNSSubdomainFlagName, false, "DNS Subdomain", nil)
	if err != nil {
		return err
	}

	endpoint := resources.GetArmadaV2Endpoint(c)

	rawJSON, sd, err := endpoint.GetLocationSubdomain(locationNameOrID, nlbDNSSubdomain)
	if err != nil {
		return err
	}

	jsonoutput := c.Bool(models.JSONOutFlagName)
	if jsonoutput {
		cliutils.WriteJSON(c, rawJSON)
		return nil
	}

	fmt.Fprintln(c.App.Writer)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Cluster:", sd.ClusterID)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Subdomain:", sd.NlbHost)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Target(s):", strings.Join(sd.NlbIPArray, ","))
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Status:", sd.NlbStatusMessage)
	fmt.Fprintln(c.App.Writer)
	fmt.Fprintln(c.App.Writer, "SSL Cert")
	fmt.Fprintf(c.App.Writer, "\t%s\t%s\n", "Secret Name:", sd.NlbSSLSecretName)
	fmt.Fprintf(c.App.Writer, "\t%s\t%s\n", "Secret Namespace:", sd.SecretNamespace)
	fmt.Fprintf(c.App.Writer, "\t%s\t%s\n", "Status:", sd.NlbSslSecretStatus)

	return nil
}
