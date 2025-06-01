
// Package cmdendpoint registers and executes 'ibmcloud sat endpoint ...' commands.
package cmdendpoint

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/flag"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/metrics"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models/link"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

// RegisterCommands registers the Satellite endpoint commands
func RegisterCommands() []cli.Command {
	requiredEndpointNameFlag := flag.StringFlag{
		Require: true,
		StringFlag: cli.StringFlag{
			Name:  models.NameFlagName,
			Usage: "Name of the endpoint to be created.",
		},
	}

	return []cli.Command{
		{
			Name:        models.NamespaceEndpoint,
			Description: "View and manage Satellite endpoints.",
			Usage:       "View and manage Satellite endpoints.",
			Subcommands: []cli.Command{
				{
					Name:        models.CmdCreate,
					Description: "Create a Satellite endpoint.",
					Usage:       "Create a Satellite endpoint.",
					Flags: []cli.Flag{
						models.RequiredLocationFlag,
						requiredEndpointNameFlag,
						models.RequiredSourceProtocolFlag,
						models.RequiredDestTypeFlag,
						models.RequiredDestHostnameFlag,
						models.RequiredDestPortFlag,
						models.DestProtocolFlag,
						models.MetricsFlag,
						models.PollIntervalFlag,
					},
					Action: endpointCreate,
				},
				{
					Name:        models.CmdGet,
					Description: "View the details of a Satellite endpoint.",
					Usage:       "View the details of a Satellite endpoint.",
					Flags: []cli.Flag{
						models.RequiredLocationFlag,
						models.RequiredEndpointFlag,
						models.JSONOutFlag,
					},
					Action: endpointGet,
				},
				{
					Name:        models.CmdList,
					Description: "List all endpoints in a Satellite location.",
					Usage:       "List all endpoints in a Satellite location.",
					Flags: []cli.Flag{
						models.RequiredLocationFlag,
						models.JSONOutFlag,
					},
					Action: endpointList,
				},
				{
					Name:        models.CmdRemove,
					Description: "Delete a Satellite endpoint.",
					Usage:       "Delete a Satellite endpoint.",
					Flags: []cli.Flag{
						models.RequiredLocationFlag,
						models.RequiredEndpointFlag,
						models.MetricsFlag,
						models.PollIntervalFlag},
					Action: endpointRemove,
				},
			},
		},
	}
}

func endpointCreate(c *cli.Context) error {
	var endpointPollInterval time.Duration

	// Initialize metrics data
	metricsData := cliutils.GetMetadataValue(c, models.MetricsFlagName).(*metrics.Data)
	metricsData.Command = fmt.Sprintf("%s-endpoint", strings.Split(cliutils.GetCommandName(c), " ")[1])
	metricsData.Enabled = cliutils.GetFlagBool(c, models.MetricsFlagName)

	pollIntervalStr, err := cliutils.GetFlagString(c, models.PollIntervalFlagName, true, "Endpoint status polling interval", nil)
	if err != nil {
		return err
	}
	if len(pollIntervalStr) > 0 {
		endpointPollInterval, err = time.ParseDuration(pollIntervalStr)
		if err != nil {
			return err
		}
	}

	endpointName, err := cliutils.GetFlagString(c, models.NameFlagName, false, "Name of the endpoint to be created", nil)
	if err != nil {
		return err
	}

	// Required. The ID of the location.
	locationID, err := cliutils.GetFlagString(c, models.LocationFlagName, false, "Location Identifier", nil)
	if err != nil {
		return err
	}

	// Required. The place where the destination resource runs, either in IBM Cloud (cloud) or your Satellite location (location).
	endpointType, err := cliutils.GetFlagString(c, models.DestTypeFlagName, false, "Endpoint destination type ('cloud' or 'location')", nil)
	if err != nil {
		return err
	}

	// Required. The URL or the externally accessible IP address of the destination resource that is to be connected to.
	// The URL should be entered without http:// or https://
	destHostname, err := cliutils.GetFlagString(c, models.DestHostnameFlagName, false, "Destination hostname or IP", nil)
	if err != nil {
		return err
	}

	// Required. The port that the destination resource listens on for incoming requests.
	// The port should match the destination protocol.
	destPort, err := cliutils.GetFlagInt(c, models.DestPortFlagName, false, "Destination port", nil)
	if err != nil {
		return err
	}
	destPortU := uint(destPort)

	// Required. The protocol that the source must use to connect to the destination resource.
	// Supported protocols include tcp, udp, tls, http, https, and http-tunnel.
	sourceProtocol, err := cliutils.GetFlagString(c, models.SourceProtocolFlagName, false, "Source protocol ('tcp', 'udp', 'tls', 'http', 'https' or 'http-tuunnel')", nil)
	if err != nil {
		return err
	}

	// Optional. The protocol of the destination resource. If you do not specify this flag, the destination protocol is inherited from the source protocol.
	// Supported protocols include tcp, udp, tls, http, https, and http-tunnel
	destProtocol, err := cliutils.GetFlagString(c, models.DestProtocolFlagName, true, "Destination protocol ('tcp', 'udp', 'tls', 'http', 'https' or 'http-tuunnel')", sourceProtocol)
	if err != nil {
		return err
	}

	endpoint := link.Endpoint{
		Name:                endpointName,
		DestinationType:     endpointType,
		SourceProtocol:      sourceProtocol,
		DestinationProtocol: destProtocol,
		DestinationHost:     destHostname,
		DestinationPort:     &destPortU,
	}

	sle := resources.GetSatLinkEndpoint(c)

	startTime := time.Now()

	endpointID, err := sle.CreateEndpoint(locationID, endpoint)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.App.Writer, "\nEndpoint creation request for '%s' successful. ID: '%s'\n", endpointName, endpointID)

	// If we're interested in timings, wait for the endpoint to be created. (Seems to be primarily source host and port data)
	if endpointPollInterval > 0 {
		fmt.Fprintf(c.App.Writer, "%s - Waiting for endpoint creation to complete", time.Now().Format(time.Stamp))
		fmt.Fprintln(c.App.Writer)
		for {
			_, endpoint, _, err := sle.GetEndpoint(locationID, endpointID)
			if err != nil {
				return err
			}

			if endpoint.SourcePort != nil && len(endpoint.SourceHost) > 0 && endpoint.Status == "enabled" {
				break
			}
			time.Sleep(endpointPollInterval)
		}
	}

	totalTime := time.Since(startTime)
	metricsData.E = append(metricsData.E, totalTime)
	fmt.Fprintf(c.App.Writer, "Total Duration: %.1fs\n", totalTime.Seconds())

	return nil
}

func endpointList(c *cli.Context) error {
	locationID, err := cliutils.GetFlagString(c, models.LocationFlagName, false, "Location Identifier", nil)
	if err != nil {
		return err
	}

	sle := resources.GetSatLinkEndpoint(c)
	rawJSON, endpoints, err := sle.ListEndpoints(locationID)
	if err != nil {
		return err
	}

	jsonoutput := c.Bool(models.JSONOutFlagName)
	if jsonoutput {
		cliutils.WriteJSON(c, rawJSON)
		return nil
	}

	fmt.Fprintln(c.App.Writer)
	fmt.Fprintln(c.App.Writer, "Name\tID\tStatus\tDestination Type\tProtocol\tLink")
	fmt.Fprintln(c.App.Writer, "----\t--\t------\t----------------\t--------\t----")

	sort.SliceStable(endpoints, func(a, b int) bool {
		return endpoints[a].Name < endpoints[b].Name
	})

	for _, e := range endpoints {
		source := e.SourceHost
		if e.SourcePort != nil {
			source = net.JoinHostPort(source, strconv.FormatUint(uint64(*e.SourcePort), 10))
		}
		destination := e.DestinationHost
		if e.DestinationPort != nil {
			destination = net.JoinHostPort(destination, strconv.FormatUint(uint64(*e.DestinationPort), 10))
		}

		fmt.Fprintf(c.App.Writer,
			"%s\t%s\t%s\t%s\t%s -> \n\t\t\t\t%s\n",
			e.Name,
			e.ID,
			e.Status,
			e.DestinationType,
			strings.Join([]string{e.SourceProtocol, source}, "\t"),
			strings.Join([]string{e.DestinationProtocol, destination}, "\t"),
		)
	}

	return nil
}

func endpointGet(c *cli.Context) error {
	locationID, err := cliutils.GetFlagString(c, models.LocationFlagName, false, "Location Identifier", nil)
	if err != nil {
		return err
	}

	endpointID, err := cliutils.GetFlagString(c, models.EndpointFlagName, false, "Endpoint Identifier", nil)
	if err != nil {
		return err
	}

	sle := resources.GetSatLinkEndpoint(c)
	rawJSON, endpoint, _, err := sle.GetEndpoint(locationID, endpointID)
	if err != nil {
		return err
	}

	jsonoutput := c.Bool(models.JSONOutFlagName)
	if jsonoutput {
		cliutils.WriteJSON(c, rawJSON)
		return nil
	}

	source := net.JoinHostPort(endpoint.SourceHost, strconv.FormatUint(uint64(*endpoint.SourcePort), 10))
	destination := net.JoinHostPort(endpoint.DestinationHost, strconv.FormatUint(uint64(*endpoint.DestinationPort), 10))

	fmt.Fprintln(c.App.Writer)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "ID:", endpoint.ID)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Name:", endpoint.Name)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Source:", strings.Join([]string{endpoint.SourceProtocol, source}, " "))
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Destination:", strings.Join([]string{endpoint.DestinationProtocol, destination}, " "))
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Destination Type:", endpoint.DestinationType)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Status:", endpoint.Status)

	return nil
}

func endpointRemove(c *cli.Context) error {
	var endpointPollInterval time.Duration

	// Initialize metrics data
	metricsData := cliutils.GetMetadataValue(c, models.MetricsFlagName).(*metrics.Data)
	metricsData.Command = fmt.Sprintf("%s-endpoint", strings.Split(cliutils.GetCommandName(c), " ")[1])
	metricsData.Enabled = cliutils.GetFlagBool(c, models.MetricsFlagName)

	pollIntervalStr, err := cliutils.GetFlagString(c, models.PollIntervalFlagName, true, "Endpoint status polling interval", nil)
	if err != nil {
		return err
	}
	if len(pollIntervalStr) > 0 {
		endpointPollInterval, err = time.ParseDuration(pollIntervalStr)
		if err != nil {
			return err
		}
	}

	locationID, err := cliutils.GetFlagString(c, models.LocationFlagName, false, "Location Identifier", nil)
	if err != nil {
		return err
	}

	endpointID, err := cliutils.GetFlagString(c, models.EndpointFlagName, false, "Endpoint Identifier", nil)
	if err != nil {
		return err
	}

	sle := resources.GetSatLinkEndpoint(c)
	_, endpoint, _, err := sle.GetEndpoint(locationID, endpointID)
	if err != nil {
		return err
	}

	startTime := time.Now()

	err = sle.DeleteEndpoint(locationID, endpointID)
	if err != nil {
		return err
	}

	fmt.Fprintf(c.App.Writer, "Endpoint deletion request for '%s' successful.\n", endpoint.Name)

	// If we're interested in timings, wait for the endpoint to be deleted.
	if endpointPollInterval > 0 {
		for _, _, notFound, err := sle.GetEndpoint(locationID, endpointID); !notFound; {
			fmt.Fprintf(c.App.Writer, "%s - Waiting for endpoint to be removed", time.Now().Format(time.Stamp))
			fmt.Fprintln(c.App.Writer)
			if err != nil {
				fmt.Fprintf(c.App.Writer, "Error checking endpoint status : %s\n", err.Error())
			}
			time.Sleep(endpointPollInterval)
		}
	}

	totalTime := time.Since(startTime)
	metricsData.E = append(metricsData.E, totalTime)
	fmt.Fprintf(c.App.Writer, "Total Duration: %.1fs\n", totalTime.Seconds())

	return nil
}
