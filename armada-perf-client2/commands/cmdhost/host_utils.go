/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2020, 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdhost

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bramvdbogaerde/go-scp"
	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-api-model/json/v2/responses"
	"github.ibm.com/alchemy-containers/armada-model/model"

	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/metrics"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
	"golang.org/x/crypto/ssh"
)

// sortedHosts returns a sorted slice of hostnames
func sortedHosts(hostConfig map[string]cliutils.TomlSatelliteInfrastructureHost) []string {
	// Sort the host configuration based on the key, i.e. hostname
	shc := make([]string, 0, len(hostConfig))
	for k := range hostConfig {
		shc = append(shc, k)
	}

	if len(shc) > 0 {
		sort.Slice(shc, func(i, j int) bool {
			// Find the start of the hostnames numerical suffix
			nsii := strings.LastIndex(shc[i], "-")
			nsij := strings.LastIndex(shc[j], "-")

			if (nsii >= 0) && (nsij >= 0) {
				// Get the hostname without numerical suffix
				hni := shc[i][0:nsii]
				hnj := shc[j][0:nsij]

				// Get the numerical suffix
				hii, _ := strconv.Atoi(shc[i][nsii+1:])
				hij, _ := strconv.Atoi(shc[j][nsij+1:])

				// If the hostnames (without numerical suffix) are the same, sort on the numerical suffix, otherwise just sort on the hostname
				if hni == hnj {
					return hii < hij
				}
				return hni < hnj
			}
			return shc[i] < shc[j]
		})
	}

	return shc
}

// attachHostToLocation will copy (using scp) and/or execute (using ssh) a script on a remote host.
func attachHostToLocation(c *cli.Context, hostname string, iaasType string, host cliutils.TomlSatelliteInfrastructureHost, privateKeyFilename, scriptfilename string) error {
	var addr string

	switch iaasType {
	case "classic":
		addr = host.IP
	case "vpc-gen2":
		addr = host.VPC.FloatingIP
	}

	if len(addr) == 0 {
		addr = hostname
	}

	// Get the private key from the specified file or environment var
	var privateKey []byte
	if len(privateKeyFilename) > 0 {
		pkf, err := os.Open(privateKeyFilename)
		if err != nil {
			return err
		}
		defer pkf.Close()

		privateKey, err = ioutil.ReadAll(pkf)
		if err != nil {
			return err
		}
	} else {
		// private key file not specified, try an envar
		// (TIP: To generate an env var for testing - export SATELLITE_HOST_PRIVATE_KEY=`cat ~/.ssh/id_rsa`)
		privateKey = []byte(os.Getenv("SATELLITE_HOST_PRIVATE_KEY"))
	}

	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return err
	}

	clientConfig := ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Second * 30,
	}

	// Copy script onto host machine?
	fmt.Fprintln(c.App.Writer, "\tCopying script")

	scpClient := scp.NewClient(net.JoinHostPort(addr, "22"), &clientConfig)

	err = scpClient.Connect()
	if err != nil {
		return fmt.Errorf("couldn't establish a connection to the remote server : %s", err)
	}
	defer scpClient.Close()

	// Open the script file
	sf, err := os.Open(scriptfilename)
	if err != nil {
		return fmt.Errorf("error opening host attach script: %s", err)
	}
	defer sf.Close()

	// Copy the file over to the remote host
	err = scpClient.CopyFile(sf, filepath.Join("/tmp", filepath.Base(scriptfilename)), "0755")
	if err != nil {
		return fmt.Errorf("error while copying Satellite host attach script to remote host : %s\n%s", addr, err)
	}

	// Execute script on host machine
	fmt.Fprintln(c.App.Writer, "\tExecuting script")

	sshClient, err := ssh.Dial("tcp", net.JoinHostPort(addr, "22"), &clientConfig)
	if err != nil {
		return err
	}

	session, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// And finally, run it.
	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run(fmt.Sprintf("bash -c %s", filepath.Join("/tmp", filepath.Base(scriptfilename)))); err != nil {
		return fmt.Errorf("error while running Satellite host attach script on remote host : %s\n%s", addr, err)
	}

	if debug {
		// Display the results
		fmt.Fprintln(c.App.Writer, b.String())
	}
	return nil
}

type metricsTime struct {
	startTime    time.Time
	pollInterval time.Duration
	timeout      time.Duration
}

const hostOperationAttach = 0
const hostOperationAssign = 1
const hostOperationPoll = 2

func monitorHosts(c *cli.Context, mt metricsTime, hostsToMonitor map[string]cliutils.TomlSatelliteInfrastructureHost, hostOperation int) error {
	var pollRequestTime time.Time
	var assignmentComplete bool
	var reportChange bool

	var locationHosts = make(map[string]*metrics.Host)

	locationNameOrID, _ := cliutils.GetFlagString(c, models.LocationFlagName, false, "Location name or ID", nil)

	endpoint := resources.GetArmadaV2Endpoint(c)

	for !assignmentComplete {
		assignmentComplete = true
		hostsFound := 0

		pollRequestTime = time.Now()

		if mt.timeout > 0 && time.Since(mt.startTime) > mt.timeout {
			return fmt.Errorf("timeout waiting for hosts to be Ready/Normal. Timeout: %v", mt.timeout)
		}

		_, hosts, err := endpoint.GetHosts(locationNameOrID)
		if err != nil || len(hosts) == 0 {
			// Error occurred or no hosts returned yet, let's ignore and carry on polling
			time.Sleep(mt.pollInterval)
			assignmentComplete = false
			continue
		}

		headerWritten := false

		// In the polling case we don't actually check the
		// hosts so we can skip this loop.
		if hostOperation != hostOperationPoll {

			for _, host := range hosts {
				hostOfInterest := false

				for name := range hostsToMonitor {
					if host.Name == name {
						hostOfInterest = true
						hostsFound++
						break
					}
				}

				if !hostOfInterest {
					continue
				}

				reportChange = false

				if _, ok := locationHosts[host.ID]; !ok {
					reportChange = true
					locationHosts[host.ID] = &metrics.Host{
						Duration: time.Since(mt.startTime),
						State:    host.Health.Status,
						Message:  host.Health.Message,
					}
				} else {
					if (locationHosts[host.ID].State != host.Health.Status) || (locationHosts[host.ID].Message != host.Health.Message) {
						reportChange = true
						locationHosts[host.ID].State = host.Health.Status
						locationHosts[host.ID].Message = host.Health.Message
						locationHosts[host.ID].Duration = time.Since(mt.startTime)
					}
				}

				// Need to wait for unassigned hosts to be "ready" when attaching.
				// Need to wait for assigned hosts to be "normal" when assigning.
				hostCompleted := (hostOperation == hostOperationAssign && strings.EqualFold(host.State, model.QueueNodeAssignmentStateAssigned) && strings.EqualFold(host.Health.Status, *model.WorkerActualHealthStateNormal)) ||
					(hostOperation == hostOperationAttach && strings.EqualFold(host.State, model.QueueNodeAssignmentStateUnassigned) && strings.EqualFold(host.Health.Status, responses.QueueNodeHealthStatusReady))

				if reportChange {
					if !headerWritten {
						fmt.Fprintln(c.App.Writer, pollRequestTime.Format(time.Stamp))
						fmt.Fprintf(c.App.Writer, "\t%s\n", locationNameOrID)
						headerWritten = true
					}
					fmt.Fprintf(c.App.Writer, "\t\t%s, \"%s\", \"%s\"", host.Name, host.Health.Status, host.Health.Message)
					fmt.Fprintln(c.App.Writer)
				}

				assignmentComplete = assignmentComplete && hostCompleted
			}
		}

		// Ensure we've checked all the hosts of interest
		assignmentComplete = assignmentComplete && (hostsFound == len(hostsToMonitor))

		if headerWritten {
			fmt.Fprintln(c.App.Writer)
		}

		if !assignmentComplete {
			time.Sleep(mt.pollInterval)
		}
	}

	// Store the command duration for metrics reporting
	metricsData := cliutils.GetMetadataValue(c, models.MetricsFlagName).(*metrics.Data)
	metricsData.H.Hosts = locationHosts

	return nil
}

func monitorLocation(c *cli.Context, mt metricsTime) error {
	var coreOSEnabled bool

	// Initialize metrics data
	metricsData := cliutils.GetMetadataValue(c, models.MetricsFlagName).(*metrics.Data)
	metricsData.Command = fmt.Sprintf("%s-host", strings.Split(cliutils.GetCommandName(c), " ")[1])
	metricsData.Enabled = cliutils.GetFlagBool(c, models.MetricsFlagName)

	// Required. The name or ID of the location where the Satellite control plane or cluster exists and to which the compute host should be assigned.
	locationNameOrID, _ := cliutils.GetFlagString(c, models.LocationFlagName, false, "Location name or ID", nil)

	locationReady := false
	previousState := ""
	previousMessage := ""

	endpoint := resources.GetArmadaV2Endpoint(c)

	for !locationReady {
		pollRequestTime := time.Now()

		if mt.timeout > 0 && time.Since(mt.startTime) > mt.timeout {
			return fmt.Errorf("timeout waiting for location to be Ready/Normal. Timeout: %v", mt.timeout)
		}
		_, location, _, err := endpoint.GetLocation(locationNameOrID)
		if err != nil {
			// Error occurred, let's ignore and carry on polling
			time.Sleep(mt.pollInterval)
			locationReady = false
			continue
		}

		if (location.State != previousState) || (location.Deployments.Message != previousMessage) {
			fmt.Fprintln(c.App.Writer, pollRequestTime.Format(time.Stamp))
			fmt.Fprintf(c.App.Writer, "\t%s : \"%s\", \"%s\"\n", location.Name, location.State, location.Deployments.Message)
			fmt.Fprintln(c.App.Writer)

			previousState = location.State
			previousMessage = location.Deployments.Message
		}

		locationReady = strings.EqualFold(location.State, *model.ClusterActualHealthStateNormal)
		if !locationReady {
			time.Sleep(mt.pollInterval)
		} else {
			coreOSEnabled = location.CoreOSEnabled
		}
	}
	metricsData.H.Location = metrics.Location{Duration: time.Since(mt.startTime), CoreOS: coreOSEnabled}

	return nil
}
