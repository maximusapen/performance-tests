
package cmdhost

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/filter"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
	"github.com/softlayer/softlayer-go/sl"
	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"

	"go.uber.org/ratelimit"
)

// SoftlayerVG defines data for accessing Softlayer API
type SoftlayerVG struct {
	c               *cli.Context
	rateLimit       ratelimit.Limiter
	satelliteConfig *cliutils.TomlSatelliteInfrastructureConfig
	accountService  services.Account
	vgService       services.Virtual_Guest
}

func hostDomain(locationName string) string {
	return strings.Join([]string{locationName, "satellite"}, ".")
}

var slVG *SoftlayerVG

func (s SoftlayerVG) String() string {
	return "classic"
}

// SetUserData isn't supported/required for Softlayer classic infrastructure
func (s SoftlayerVG) SetUserData(ud string) {
}

// ProvisionDelay adds in a delay following provisioning
// see https://ibm-argonauts.slack.com/archives/CSEEYJ332/p1605716858429400?thread_ts=1605716157.428500&cid=CSEEYJ332
func (s SoftlayerVG) ProvisionDelay() time.Duration {
	return 5 * time.Minute
}

func (s SoftlayerVG) getPublicSSHKey() (int, error) {
	classicConfig := cliutils.GetInfrastructureConfig().Classic
	publicKeyID, err := cliutils.GetFlagInt(s.c, models.PublicKeyFlagName, true, "Public key identifier", classicConfig.SSHKey)
	return publicKeyID, err
}

// NewSoftlayerVG returns an authenticated Softlayer virtual guest object
func NewSoftlayerVG(c *cli.Context) *SoftlayerVG {
	if slVG == nil {
		session := session.New(
			cliutils.GetArmadaConfig().IBMCloud.ClassicIaasUsername,
			cliutils.GetArmadaConfig().IBMCloud.ClassicIaasAPIKey,
			"",
			"15",
		)

		slVG = &SoftlayerVG{
			c:               c,
			rateLimit:       ratelimit.New(1), // Limit to 1 request / second
			satelliteConfig: cliutils.GetInfrastructureConfig().Satellite,
			accountService:  services.GetAccountService(session),
			vgService:       services.GetVirtualGuestService(session),
		}
	}

	return slVG
}

// GetLocationHosts returns a list (a map) of classic virtual servers in the configured account for the specified location
func (s SoftlayerVG) GetLocationHosts(locationName string) (map[string]string, error) {
	// Get the VM's details
	filter := filter.New(
		filter.Path("virtualGuests.domain").Eq(hostDomain(locationName)),
	).Build()

	s.rateLimit.Take()
	vms, err := s.accountService.Mask("id;hostname").Filter(filter).GetVirtualGuests()
	if err != nil {
		return nil, fmt.Errorf("failed to find virtual guests for location. %s", err.Error())
	}

	lh := make(map[string]string)
	for _, h := range vms {
		lh[*h.Hostname] = strconv.Itoa(*h.Id)
	}

	return lh, nil
}

// GetHost returns details of the specified (location name and hostname) virtual server
func (s SoftlayerVG) GetHost(locationName, hostname, id string) (cliutils.TomlSatelliteInfrastructureHost, error) {
	var hostID int

	if id == "" {
		// Get the VM's details
		filter := filter.New(
			filter.Path("virtualGuests.hostname").Eq(hostname),
			filter.Path("virtualGuests.domain").Eq(hostDomain(locationName)),
		).Build()

		s.rateLimit.Take()
		vms, err := s.accountService.Mask("id").Filter(filter).GetVirtualGuests()
		if err != nil {
			return cliutils.TomlSatelliteInfrastructureHost{}, fmt.Errorf("failed to find virtual guest in account. %s", err.Error())
		}

		// Sanity check, make sure we've matched against a single virtual server with a matching hostname and domain
		if len(vms) != 1 {
			return cliutils.TomlSatelliteInfrastructureHost{}, fmt.Errorf("failed to find unique virtual guest in account. %s", hostname)
		}

		hostID = *vms[0].Id
	} else {
		var err error
		hostID, err = strconv.Atoi(id)
		if err != nil {
			return cliutils.TomlSatelliteInfrastructureHost{}, fmt.Errorf("non integer host id supplied: %s", id)
		}
	}

	vmMask := "id,hostname,domain,datacenter,primaryIpAddress,maxCpu,maxMemory,blockDevices.diskImage," +
		"operatingSystemReferenceCode,primaryNetworkComponent.networkVlan,primaryBackendNetworkComponent.networkVlan"
	vm, err := s.vgService.Id(hostID).Mask(vmMask).GetObject()
	if err != nil {
		return cliutils.TomlSatelliteInfrastructureHost{}, fmt.Errorf("failed to retrieve virtual guest details. %s", err.Error())
	}

	host := cliutils.TomlSatelliteInfrastructureHost{
		Hostname:   *vm.Hostname,
		IaasID:     strconv.Itoa(*vm.Id),
		IP:         *vm.PrimaryIpAddress,
		Datacenter: *vm.Datacenter.Name,
		CPU:        *vm.MaxCpu,
		Memory:     *vm.MaxMemory / 1024,
		Disk:       *vm.BlockDevices[0].DiskImage.Capacity,

		Classic: cliutils.TomlSatelliteInfrastructureClassicHost{
			PrivateVLAN: strconv.Itoa(*vm.PrimaryBackendNetworkComponent.NetworkVlan.Id),
			PublicVLAN:  strconv.Itoa(*vm.PrimaryNetworkComponent.NetworkVlan.Id),
		},
	}
	return host, nil
}

// CancelHost will cancel(delete) an IBM Cloud Classic Infrastructure (Softlayer) VSI
func (s SoftlayerVG) CancelHost(location, hostname, id string) error {
	if id == "" {
		// Get the VMs details
		filter := filter.New(
			filter.Path("virtualGuests.hostname").Eq(hostname),
			filter.Path("virtualGuests.domain").Eq(hostDomain(location)),
		).Build()

		s.rateLimit.Take()
		vms, err := s.accountService.Mask("id;hostname;domain").Filter(filter).GetVirtualGuests()
		if err != nil {
			return fmt.Errorf("failed to retrieve virtual guest from account. %s", err.Error())
		}

		// Sanity check, make sure we've matched against a single virtual server with a matching hostname
		if len(vms) != 1 || hostname != *vms[0].Hostname {
			return fmt.Errorf("failed to find virtual guest in account. %s", hostname)
		}

		id = strconv.Itoa(*vms[0].Id)
	}

	s.rateLimit.Take()
	hid, _ := strconv.Atoi(id)
	success, err := s.vgService.Id(hid).DeleteObject()
	if err != nil {
		return fmt.Errorf("failed to delete the IAAS host '%s' : %s", hostname, err.Error())
	} else if !success {
		return fmt.Errorf("unable to delete the IAAS host '%s'", hostname)
	}

	fmt.Fprintf(s.c.App.Writer, "virtual server cancellation requested. Hostname: \"%s\", ID: %s\n", hostname, id)

	return nil
}

// ReloadHost will reload an IBM Cloud Classic Infrastructure (Softlayer) VSI
func (s SoftlayerVG) ReloadHost(locationConfig cliutils.TomlSatelliteInfrastructureLocation, id string, hostname string) error {
	// Get the details for the specified host from the configuration
	var host cliutils.TomlSatelliteInfrastructureHost
	var ok bool

	if id == "" {
		if host, ok = locationConfig.Hosts.Control.Servers[hostname]; !ok {
			if host, ok = locationConfig.Hosts.Cluster.Servers[hostname]; !ok {
				if host, ok = locationConfig.Hosts.Provisioned.Servers[hostname]; ok {
					return fmt.Errorf("unable to find the specified host: '%s'", hostname)
				}
			}
		}

		if host.IaasID == "" {
			return fmt.Errorf("IBM Cloud Infrastructure ID not known for host: '%s'. Unable to reload", hostname)
		}
	} else {
		host.IaasID = id
	}

	// Check if there are any active transactions in progress.
	hid, _ := strconv.Atoi(host.IaasID)
	transactions, err := s.vgService.Id(hid).GetActiveTransactions()
	if err != nil {
		return fmt.Errorf("unable to get active transactions. %s", err)
	}

	// Server can not be reloaded if there are pending transactions.
	if len(transactions) > 0 {
		return fmt.Errorf("server can not be reloaded because there is a transaction in progress for the device. %s", err)

	}

	// Final sanity check. Ensure the hostname and ids match expectations
	server, err := s.vgService.Id(hid).Mask("id;hostname;domain").GetObject()
	if err != nil {
		return fmt.Errorf("unable to get server object. %s", err)
	}
	if (*server.Id != hid) || (*server.Hostname != hostname) {
		return fmt.Errorf("Hostname/ID mismatch for host. %s", hostname)
	}

	sshKey, err := s.getPublicSSHKey()
	if err != nil {
		return fmt.Errorf("invalid public ssh key identifier. %s", err)
	}
	config := datatypes.Container_Hardware_Server_Configuration{
		SshKeyIds: []int{sshKey},
	}

	s.rateLimit.Take()
	_, err = s.vgService.Id(hid).ReloadOperatingSystem(sl.String("FORCE"), &config)
	if err != nil {
		return fmt.Errorf("unable to reload the Operating System. %s", err)
	}

	fmt.Fprintf(s.c.App.Writer, "Virtual server reload requested. Hostname: \"%s\", ID: %d\n", *server.Hostname, *server.Id)

	return nil
}

// CreateHost creates a Softlayer clasic virtual server instance
func (s SoftlayerVG) CreateHost(location string, hostname string, hc cliutils.TomlSatelliteInfrastructureHostConfig, sc cliutils.TomlSatelliteInfrastructureHost) (cliutils.TomlSatelliteInfrastructureHost, error) {
	var vh cliutils.TomlSatelliteInfrastructureHost

	// Does the host configiuration include a classic definition
	if sc.Classic != (cliutils.TomlSatelliteInfrastructureClassicHost{}) {
		// If the host doesn't exist already in the configuration data
		if sc.IaasID == "" {
			classicConfig := cliutils.GetInfrastructureConfig().Classic

			image, ok := classicConfig.OperatingSystems[hc.OS]
			if !ok {
				return vh, fmt.Errorf("Invalid or unknown classic Operating System - %s", hc.OS)
			}

			privateVLAN, err := strconv.Atoi(sc.Classic.PrivateVLAN)
			if err != nil {
				return vh, fmt.Errorf("Invalid Private VLAN: %s. Error: %s", sc.Classic.PrivateVLAN, err)
			}

			publicVLAN, err := strconv.Atoi(sc.Classic.PublicVLAN)
			if err != nil {
				return vh, fmt.Errorf("Invalid Public VLAN: %s. Error: %s", sc.Classic.PublicVLAN, err)
			}

			// Now check if a virtual server with this hostname and domain already exists in Softlayer; if so we'll just skip and reuse
			// (Note its up to the user to ensure the existing host meets their requirements)
			locationHosts, err := s.GetLocationHosts(location)
			if err != nil {
				return vh, fmt.Errorf("Unable to check for existing host: %s. Error: %s", hostname, err)
			}
			if hid, ok := locationHosts[sc.Hostname]; !ok {
				sshKey, err := s.getPublicSSHKey()
				if err != nil {
					return vh, fmt.Errorf("invalid public ssh key identifier. %s", err)
				}
				s.rateLimit.Take()

				vGuestTemplate := datatypes.Virtual_Guest{
					BlockDevices: []datatypes.Virtual_Guest_Block_Device{
						{
							Device: sl.String("0"),
							DiskImage: &datatypes.Virtual_Disk_Image{
								BootableVolumeFlag: sl.Bool(true),
								Capacity:           sl.Int(sc.Disk),
								LocalDiskFlag:      sl.Bool(true),
							},
						},
					},
					Datacenter:        &datatypes.Location{Name: sl.String(sc.Datacenter)},
					Domain:            sl.String(hostDomain(location)),
					Hostname:          sl.String(sc.Hostname),
					HourlyBillingFlag: sl.Bool(true),
					LocalDiskFlag:     sl.Bool(true),
					MaxMemory:         sl.Int(sc.Memory * 1024),
					NetworkComponents: []datatypes.Virtual_Guest_Network_Component{
						{
							MaxSpeed: sl.Int(1000),
						},
					},
					PrimaryBackendNetworkComponent: &datatypes.Virtual_Guest_Network_Component{
						NetworkVlan: &datatypes.Network_Vlan{
							Id: sl.Int(privateVLAN),
						},
					},
					PrimaryNetworkComponent: &datatypes.Virtual_Guest_Network_Component{
						NetworkVlan: &datatypes.Network_Vlan{
							Id: sl.Int(publicVLAN),
						},
					},
					OperatingSystemReferenceCode: sl.String(image),
					StartCpus:                    sl.Int(sc.CPU),
					SshKeyCount:                  sl.Uint(1),
					SshKeys: []datatypes.Security_Ssh_Key{
						{
							Id: sl.Int(sshKey),
						},
					},
				}

				vGuest, err := s.vgService.Mask("id;hostname;domain").CreateObject(&vGuestTemplate)
				if err != nil {
					return vh, fmt.Errorf("unable to create virtual server. %s", err)
				}

				vh = cliutils.TomlSatelliteInfrastructureHost{
					Hostname:   *vGuest.Hostname,
					IaasID:     strconv.Itoa(*vGuest.Id),
					Datacenter: sc.Datacenter,
					Zone:       sc.Zone,
					CPU:        sc.CPU,
					Memory:     sc.Memory,
					Disk:       sc.Disk,
					Quantity:   1,

					Classic: sc.Classic,
				}

				fmt.Fprintf(s.c.App.Writer, "\tClassic virtual server creation in progress. Hostname: \"%s\", ID: %d", *vGuest.Hostname, *vGuest.Id)
				fmt.Fprintln(s.c.App.Writer)
			} else {
				hostID, err := strconv.Atoi(hid)
				if err != nil {
					return cliutils.TomlSatelliteInfrastructureHost{}, fmt.Errorf("non integer host id supplied: %s", hid)
				}
				transactions, err := s.vgService.Id(hostID).GetActiveTransactions()
				if err != nil {
					return vh, fmt.Errorf("unable to get active transactions for existing device. %s", err)
				}

				// Cannot reuse server if it has an active transaction, could be being deleted for example
				if len(transactions) > 0 {
					return vh, fmt.Errorf("unable to create host, transaction in progress on existing device. '%s : %d'", hostname, hostID)

				}
				fmt.Fprintf(s.c.App.Writer, "\tReusing classic virtual server. Check it meets your requirements. Hostname: \"%s\", ID: %s", sc.Hostname, hid)
				fmt.Fprintln(s.c.App.Writer)

				vh, err = s.GetHost(location, hostname, hid)
				if err != nil {
					return vh, err
				}
			}
		}
	}
	return vh, nil
}

// WaitForHosts will block until all the specified hosts are ready (have an assigned Primary IP address) in the Softlayer classic infrastructure
func (s SoftlayerVG) WaitForHosts(hosts map[string]cliutils.TomlSatelliteInfrastructureHost) error {
	fmt.Fprintf(s.c.App.Writer, "\n%s : Waiting for classic infrastructure hosts to be ready\n", time.Now().Format(time.Stamp))

	for hostsReady := 0; hostsReady < len(hosts); {
		for n, h := range hosts {
			if h.IP == "" {
				s.rateLimit.Take()
				hid, _ := strconv.Atoi(h.IaasID)
				transactions, err := s.vgService.Id(hid).GetActiveTransactions()
				if err != nil {
					return fmt.Errorf("unable to get active transactions. %s", err)
				}

				if len(transactions) == 0 {
					s.rateLimit.Take()
					server, err := s.vgService.Id(hid).Mask("primaryIpAddress").GetObject()
					if err != nil {
						return fmt.Errorf("unable to get virtual server object. %s", err)
					}

					if server.PrimaryIpAddress != nil {
						// No active transactions and IP address assigned. Update the IP andthen We're done for this host.

						// Workaround for https://github.com/golang/go/issues/3117
						var tmp = hosts[n]
						tmp.IP = *server.PrimaryIpAddress
						hosts[n] = tmp

						hostsReady++

						fmt.Fprintf(s.c.App.Writer, "\tInfrastructure hosts ready: %d\n", hostsReady)
					} else {
						time.Sleep(time.Second * 30)
						break
					}
				} else {
					// Ongoing transactions, check again in 30s
					time.Sleep(time.Second * 30)
				}
			}
		}
	}

	fmt.Fprintln(s.c.App.Writer)
	return nil
}
