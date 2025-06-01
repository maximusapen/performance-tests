/*******************************************************************************
 * I
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2022, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdhost

import (
	"fmt"
	"strings"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
)

var vpcService *VPCService

// VPCService defines data for accessing VPC Iaas API
type VPCService struct {
	c               *cli.Context
	satelliteConfig *cliutils.TomlSatelliteInfrastructureConfig
	service         *vpcv1.VpcV1
	userData        string
}

func (v VPCService) String() string {
	return "vpc-gen2"
}

// ProvisionDelay adds in a delay following provisioning
// Np delay required for VPC infrastructure
func (v VPCService) ProvisionDelay() time.Duration {
	return 0
}

// SetUserData sets the sser data to transfer to the VPC virtual server instance
func (v *VPCService) SetUserData(ud string) {
	v.userData = ud
}

func (v VPCService) getPublicSSHKey() (string, error) {
	vpcConfig := cliutils.GetInfrastructureConfig().VPC
	publicKeyName, err := cliutils.GetFlagString(v.c, models.PublicKeyFlagName, true, "Public key identifier", vpcConfig.SSHKey)
	return publicKeyName, err
}

// NewVPCService returns an authenticated VPC Service object
func NewVPCService(c *cli.Context) (*VPCService, error) {
	if vpcService == nil {
		ac := cliutils.GetArmadaConfig()

		apikey := ac.IBMCloud.InfraAPIKey // pragma: allowlist secret
		iamURL := ac.IBMCloud.InfraIAMEndpoint + "/identity/token"

		vpcIaasURL := ac.VPC.Endpoint + "/v1"

		// Instantiate the service with an API key based IAM authenticator
		s, err := vpcv1.NewVpcV1(&vpcv1.VpcV1Options{
			Authenticator: &core.IamAuthenticator{
				ApiKey: apikey, // pragma: allowlist secret
				URL:    iamURL,
			},
			URL: vpcIaasURL,
		})

		if err != nil {
			return nil, err
		}

		vpcService = &VPCService{
			c:               c,
			satelliteConfig: cliutils.GetInfrastructureConfig().Satellite,
			service:         s,
		}
	}

	return vpcService, nil
}

// ListInstances lists all VPC virtual server instances
func (v VPCService) ListInstances() error {
	// Retrieve the list of regions for your account.
	ic, detailedResponse, err := v.service.ListInstances(&vpcv1.ListInstancesOptions{})
	if err != nil {
		return fmt.Errorf("Failed to list the vpc server instances: %s. Response: %s", err.Error(), detailedResponse)
	}
	for _, i := range ic.Instances {
		fmt.Fprintln(v.c.App.Writer, i)
		fmt.Fprintln(v.c.App.Writer)
	}

	return nil
}

// CreateHost creates a VPC virtual server instance
func (v VPCService) CreateHost(location string, hostname string, hc cliutils.TomlSatelliteInfrastructureHostConfig, sc cliutils.TomlSatelliteInfrastructureHost) (cliutils.TomlSatelliteInfrastructureHost, error) {
	var vh cliutils.TomlSatelliteInfrastructureHost

	// Does the host configiuration include a vpc instance definition
	if sc.VPC != (cliutils.TomlSatelliteInfrastructureVPCHost{}) {
		vpcConfig := cliutils.GetInfrastructureConfig().VPC

		if !strings.HasSuffix(hostname, "satellite") {
			hostname = strings.Join([]string{hostname, location, "satellite"}, "-")
		}

		profile := fmt.Sprintf("bx2-%dx%d", sc.CPU, sc.Memory)
		instanceProfileIdentityModel := &vpcv1.InstanceProfileIdentityByName{
			Name: &profile,
		}

		// Get the VPC identifier
		vpcID, err := v.getVPCIDFromName(sc.VPC.VPC)
		if err != nil {
			return cliutils.TomlSatelliteInfrastructureHost{}, err
		}
		vpcIDentityModel := &vpcv1.VPCIdentityByID{
			ID: vpcID,
		}

		// Get the image identifier
		imageName, ok := vpcConfig.OperatingSystems[hc.OS]
		if !ok {
			return vh, fmt.Errorf("Invalid or unknown VPC Operating System - %s", hc.OS)
		}
		imageID, err := v.getImageIDFromName(imageName)
		if err != nil {
			return cliutils.TomlSatelliteInfrastructureHost{}, err
		}
		imageIDentityModel := &vpcv1.ImageIdentityByID{
			ID: imageID,
		}

		// Get the VPC subnet identifier
		subnetID, err := v.getSubnetIDFromName(sc.VPC.Subnet)
		if err != nil {
			return cliutils.TomlSatelliteInfrastructureHost{}, err
		}
		subnetIDentityModel := &vpcv1.SubnetIdentityByID{
			ID: subnetID,
		}
		networkInterfacePrototypeModel := &vpcv1.NetworkInterfacePrototype{
			Subnet: subnetIDentityModel,
		}

		// Set the zone name
		zoneIdentityModel := &vpcv1.ZoneIdentityByName{
			Name: &sc.Zone,
		}

		// Construct the instance tamplate
		instancePrototypeModel := &vpcv1.InstancePrototypeInstanceByImage{
			Keys:                    []vpcv1.KeyIdentityIntf{},
			Name:                    &hostname,
			Profile:                 instanceProfileIdentityModel,
			VPC:                     vpcIDentityModel,
			Image:                   imageIDentityModel,
			PrimaryNetworkInterface: networkInterfacePrototypeModel,
			Zone:                    zoneIdentityModel,
			UserData:                &v.userData,
		}

		// Finally, append any SSH key to the instance template
		sshKey, err := v.getPublicSSHKey()
		if err != nil {
			return cliutils.TomlSatelliteInfrastructureHost{}, fmt.Errorf("invalid public ssh key name. %s", err)
		}
		if sshKey != "" {
			keyID, err := v.getKeyIDFromName(sshKey)
			if err != nil {
				return cliutils.TomlSatelliteInfrastructureHost{}, err
			}
			keyIDentityModel := &vpcv1.KeyIdentityByID{
				ID: keyID,
			}

			instancePrototypeModel.Keys = append(instancePrototypeModel.Keys, keyIDentityModel)
		}

		createInstanceOptions := v.service.NewCreateInstanceOptions(
			instancePrototypeModel,
		)

		// Create the VPC server instance
		instance, response, err := v.service.CreateInstance(createInstanceOptions)
		if err != nil {
			return cliutils.TomlSatelliteInfrastructureHost{}, fmt.Errorf("Failed to create vpc server instance: %s. Response: %s", err.Error(), response)
		}

		// Set our configuration data from the returned instance details
		vh = cliutils.TomlSatelliteInfrastructureHost{
			Hostname:   *instance.Name,
			IaasID:     *instance.ID,
			Datacenter: sc.Datacenter,
			Zone:       sc.Zone,
			CPU:        int(*instance.Vcpu.Count),
			Memory:     int(*instance.Memory),
			Quantity:   1,

			VPC: sc.VPC,
		}

		fmt.Fprintf(v.c.App.Writer, "\tVPC virtual server instance creation in progress. Hostname: \"%s\", ID: %s", *instance.Name, *instance.ID)
		fmt.Fprintln(v.c.App.Writer)

		// We need a public interface so create and associate a Floating IP
		fip, err := v.createAndAssociateFloatingIP(vh.IaasID)
		if err != nil {
			return vh, fmt.Errorf("Failed to create/associate floating IP with vpc server instance: %s", err.Error())
		}

		vh.VPC.FloatingIP = fip
	}
	return vh, nil
}

// CancelHost will cancel(delete) an IBM Cloud VPC virtual server instance
func (v VPCService) CancelHost(location, name, id string) error {
	if id == "" {
		instID, err := v.getInstanceIDFromName(name)
		if err != nil {
			return err
		}

		id = *instID
	}

	err := v.disassociateAndReleaseFloatingIPs(id)
	if err != nil {
		return fmt.Errorf("CancelHost: %s", err.Error())
	}

	deleteInstanceOptions := v.service.NewDeleteInstanceOptions(id)
	response, err := v.service.DeleteInstance(deleteInstanceOptions)
	if err != nil {
		return fmt.Errorf("Failed to delete VPC server instance: %s. Response: %s", err.Error(), response)
	}

	fmt.Fprintf(v.c.App.Writer, "VPC virtual server instance deletion requested. Name: \"%s\", ID: %s\n", name, id)

	return nil
}

// GetLocationHosts returns a list (a map) of an IBM Cloud VPC virtual servers in the configured account for the specified location
func (v VPCService) GetLocationHosts(locationName string) (map[string]string, error) {
	listInstancesOptions := v.service.NewListInstancesOptions()

	instances, response, err := v.service.ListInstances(listInstancesOptions)
	if err != nil {
		return nil, fmt.Errorf("Failed to list vpc server instances: %s. Response: %s", err.Error(), response)
	}

	lh := make(map[string]string)

	for _, ins := range instances.Instances {
		if strings.HasSuffix(*ins.Name, strings.Join([]string{locationName, "satellite"}, "-")) {
			lh[*ins.Name] = *ins.ID
		}
	}
	return lh, nil
}

// GetHost returns details of the specified IBM Cloud VPC virtual server
func (v VPCService) GetHost(location, name, id string) (cliutils.TomlSatelliteInfrastructureHost, error) {
	// If the Id wasn't supplied we'll need to look it up from the instance name
	if id == "" {
		instID, err := v.getInstanceIDFromName(name)
		if err != nil {
			return cliutils.TomlSatelliteInfrastructureHost{}, err
		}

		id = *instID
	}

	getInstanceOptions := v.service.NewGetInstanceOptions(id)
	instance, response, err := v.service.GetInstance(getInstanceOptions)
	if err != nil {
		return cliutils.TomlSatelliteInfrastructureHost{}, fmt.Errorf("Failed to get vpc server instance: %s. Response: %s", err.Error(), response)
	}

	// Get the Boot volume so we can set the disk size
	bootVol := v.getInstanceBootVolume(instance)

	host := cliutils.TomlSatelliteInfrastructureHost{
		Hostname:   *instance.Name,
		IaasID:     *instance.ID,
		IP:         *instance.PrimaryNetworkInterface.PrimaryIP.Address,
		CPU:        int(*instance.Vcpu.Count),
		Memory:     int(*instance.Memory),
		Disk:       int(*bootVol.Capacity),
		Quantity:   1,
		Zone:       *instance.Zone.Name,
		Datacenter: strings.Split(*instance.Name, "-")[2], // Assumes hostname format is arm-perf-<datacenter>-.......

		VPC: cliutils.TomlSatelliteInfrastructureVPCHost{
			Subnet: *instance.PrimaryNetworkInterface.Subnet.Name,
			VPC:    *instance.VPC.Name,
		},
	}
	return host, nil
}

// ReloadHost will recreate an IBM Cloud VPC virtual server instance
func (v VPCService) ReloadHost(locationConfig cliutils.TomlSatelliteInfrastructureLocation, id string, hostname string) error {
	/* NOTE: VPC does not support OS Reloads of a virtual server.
	 * To mimic this behaviour, we'll delete and then recreate.
	 * This does mean the concept of preconfigured hosts for VPC infrastrucutre should be supported, but it is of minimal value
	 */

	// Get the details for the specified host from the configuration
	var host cliutils.TomlSatelliteInfrastructureHost
	var hc cliutils.TomlSatelliteInfrastructureHostConfig
	var ok bool

	if id == "" {
		if host, ok = locationConfig.Hosts.Control.Servers[hostname]; ok {
			hc = locationConfig.Hosts.Control
		} else if host, ok = locationConfig.Hosts.Cluster.Servers[hostname]; ok {
			hc = locationConfig.Hosts.Cluster
		} else if host, ok = locationConfig.Hosts.Provisioned.Servers[hostname]; ok {
			hc = locationConfig.Hosts.Provisioned
		} else {
			return fmt.Errorf("unable to find the specified host: '%s'", hostname)
		}

		if host.IaasID == "" {
			return fmt.Errorf("IBM Cloud VPC Infrastructure ID not known for host: '%s'. Unable to simulate reload", hostname)
		}
	} else {
		host.IaasID = id
	}

	// VPC does not support OS Reloads, so need to delete and recreate.

	// Fist remove any floating IPs and then delete the instance
	err := v.disassociateAndReleaseFloatingIPs(id)
	if err != nil {
		return fmt.Errorf("unable to dissociate/release floating IP from instance. %s", err)
	}

	err = v.CancelHost(locationConfig.Name, id, hostname)
	if err != nil {
		return fmt.Errorf("unable to cancel VPC server instance. %s", err)
	}

	fmt.Fprintf(v.c.App.Writer, "VPC server instance creation requested. Hostname: \"%s\"\n", hostname)

	// Wait for it to die
	for true {
		_, err := v.GetHost(locationConfig.Name, hostname, host.IaasID)

		// A bit hokey, but we'll assume an error means it not been found, rather than any other error.
		// We're probably not going to be calling this much/ever, so will leave like this for now.
		if err != nil {
			break
		}
		time.Sleep(time.Second * 30)
	}

	// Now recreate
	h, err := v.CreateHost(locationConfig.Name, hostname, hc, hc.Servers[hostname])
	if err != nil {
		return fmt.Errorf("unable to create VPC server instance. %s", err)
	}

	fmt.Fprintf(v.c.App.Writer, "VPC Virtual server instance creation requested. Hostname: \"%s\", ID: %s\n", h.Hostname, h.IaasID)

	return nil
}

// WaitForHosts will block until all specified IBM Cloud VPC Gen2 virtual server instances are ready (running)
func (v VPCService) WaitForHosts(hosts map[string]cliutils.TomlSatelliteInfrastructureHost) error {
	fmt.Fprintf(v.c.App.Writer, "\n%s : Waiting for VPC infrastructure hosts to be ready\n", time.Now().Format(time.Stamp))

	for hostsReady := 0; hostsReady < len(hosts); {
		for n, h := range hosts {
			if h.IP == "" {
				getInstanceOptions := v.service.NewGetInstanceOptions(h.IaasID)
				instance, response, err := v.service.GetInstance(getInstanceOptions)
				if err != nil {
					return fmt.Errorf("Failed to get vpc server instance: %s. Error: %s. Response: %s", h.IaasID, err.Error(), response)
				}

				// Check for running status
				if *instance.Status == vpcv1.InstanceStatusRunningConst {
					// Update the IP andthen We're done for this host.
					// Workaround for https://github.com/golang/go/issues/3117
					var tmp = hosts[n]
					tmp.IP = *instance.PrimaryNetworkInterface.PrimaryIP.Address
					hosts[n] = tmp

					hostsReady++
					fmt.Fprintf(v.c.App.Writer, "\tInfrastructure hosts ready: %d\n", hostsReady)
				} else {
					time.Sleep(time.Second * 30)
					break
				}
			}
		}
	}

	fmt.Fprintln(v.c.App.Writer)
	return nil
}

func (v VPCService) createAndAssociateFloatingIP(instanceID string) (string, error) {
	getInstanceOptions := v.service.NewGetInstanceOptions(instanceID)
	instance, response, err := v.service.GetInstance(getInstanceOptions)
	if err != nil {
		return "", fmt.Errorf("Failed to get VPC server instance: %s. Response: %s", err.Error(), response)
	}

	// Reserve a floating IP
	fipName := strings.Join([]string{*instance.Name, "fip"}, "-")
	options := &vpcv1.CreateFloatingIPOptions{}
	options.SetFloatingIPPrototype(&vpcv1.FloatingIPPrototype{
		Name: &fipName,
		Zone: &vpcv1.ZoneIdentity{
			Name: instance.Zone.Name,
		},
	})
	floatingIP, response, err := v.service.CreateFloatingIP(options)
	if err != nil {
		return "", fmt.Errorf("Failed to create floating IP '%s'. Error: %s. Response: %s",
			fipName, err.Error(), response)
	}
	// We should have a single network interface, so associate the floating IP with the primary
	networkID := instance.PrimaryNetworkInterface.ID
	fipOptions := &vpcv1.AddInstanceNetworkInterfaceFloatingIPOptions{}
	fipOptions.SetID(*floatingIP.ID)
	fipOptions.SetInstanceID(instanceID)
	fipOptions.SetNetworkInterfaceID(*networkID)
	fip, response, err := v.service.AddInstanceNetworkInterfaceFloatingIP(fipOptions)
	if err != nil {
		return "", fmt.Errorf("Failed to associate floating IP '%s' with VPC server instance. Error: %s. Response: %s",
			*floatingIP.Name, err.Error(), response)
	}

	return *fip.Address, err
}

func (v VPCService) disassociateAndReleaseFloatingIPs(instanceID string) error {
	// Get all network interfaces associated with the VPC server instance
	liniOptions := &vpcv1.ListInstanceNetworkInterfacesOptions{}
	liniOptions.SetInstanceID(instanceID)
	networkInterfaces, response, err := v.service.ListInstanceNetworkInterfaces(liniOptions)
	if err != nil {
		return fmt.Errorf("Failed to list VPC instance network interfaces: %s. Response: %s", err.Error(), response)
	}

	// For each network interface
	for _, ni := range networkInterfaces.NetworkInterfaces {
		// Get all floating IPs associated with the netwrok interface
		fipOptions := &vpcv1.ListInstanceNetworkInterfaceFloatingIpsOptions{}
		fipOptions.SetInstanceID(instanceID)
		fipOptions.SetNetworkInterfaceID(*ni.ID)
		floatingIPs, response, err := v.service.ListInstanceNetworkInterfaceFloatingIps(fipOptions)
		if err != nil {
			return fmt.Errorf("Failed to list VPC network interface floating IPs: %s. Response: %s", err.Error(), response)
		}

		// For each floating IP
		for _, fip := range floatingIPs.FloatingIps {
			// Remove the asociation with the VPC server instance
			options := &vpcv1.RemoveInstanceNetworkInterfaceFloatingIPOptions{}
			options.SetID(*fip.ID)
			options.SetInstanceID(instanceID)
			options.SetNetworkInterfaceID(*ni.ID)
			response, err := v.service.RemoveInstanceNetworkInterfaceFloatingIP(options)
			if err != nil {
				return fmt.Errorf("Failed to remove VPC floating IP '%s' from network interface: '%s'.Error: %s. Response: %s",
					*fip.Name, *ni.Name, err.Error(), response)
			}

			// Finally release the floating IP
			fipOptions := v.service.NewDeleteFloatingIPOptions(*fip.ID)
			response, err = v.service.DeleteFloatingIP(fipOptions)
			if err != nil {
				return fmt.Errorf("Failed to release VPC floating IP: '%s'. Error: %s. Response: %s",
					*fip.Name, err.Error(), response)
			}
		}
	}

	return nil
}

// getInstanceIDFromName is used to get a VPC instance ID from its name. Most VPC instance api operations require an id.
func (v VPCService) getInstanceIDFromName(name string) (*string, error) {
	listInstancesOptions := v.service.NewListInstancesOptions()
	listInstancesOptions.SetName(name)

	instances, response, err := v.service.ListInstances(listInstancesOptions)
	if err != nil {
		return nil, fmt.Errorf("Failed to list VPC server instances: %s. Response: %s", err.Error(), response)
	}

	for _, ins := range instances.Instances {
		if *ins.Name == name {
			return ins.ID, nil
		}
	}

	// Instance not found
	return nil, fmt.Errorf("Failed to find VPC instance: %s", name)
}

// getImageIDFromName is used to get a VPC image ID from its name. Most VPC instance api operations require an id.
func (v VPCService) getImageIDFromName(name string) (*string, error) {
	listImagesOptions := v.service.NewListImagesOptions()
	listImagesOptions.SetName(name)

	images, response, err := v.service.ListImages(listImagesOptions)
	if err != nil {
		return nil, fmt.Errorf("Failed to list VPC images: %s. Response: %s", err.Error(), response)
	}

	for _, img := range images.Images {
		if *img.Name == name {
			return img.ID, nil
		}
	}

	// Image not found
	return nil, fmt.Errorf("Failed to find VPC image: %s", name)
}

// getVPCIDFromName is used to get a VPC ID from its name. Most VPC api operations require an id.
func (v VPCService) getVPCIDFromName(name string) (*string, error) {
	listVPCsOptions := v.service.NewListVpcsOptions()

	vpcs, response, err := v.service.ListVpcs(listVPCsOptions)
	if err != nil {
		return nil, fmt.Errorf("Failed to list VPCs: %s. Response: %s", err.Error(), response)
	}

	for _, vpc := range vpcs.Vpcs {
		if *vpc.Name == name {
			return vpc.ID, nil
		}
	}

	// VPC not found
	return nil, fmt.Errorf("Failed to find VPC: %s", name)
}

// getSubnetIDFromName is used to get a VPC subnet ID from its name. Most VPC subnet api operations require an id.
func (v VPCService) getSubnetIDFromName(name string) (*string, error) {
	listSubnetsOptions := v.service.NewListSubnetsOptions()

	subnets, response, err := v.service.ListSubnets(listSubnetsOptions)
	if err != nil {
		return nil, fmt.Errorf("Failed to list VPC subnets: %s. Response: %s", err.Error(), response)
	}

	for _, sn := range subnets.Subnets {
		if *sn.Name == name {
			return sn.ID, nil
		}
	}

	// Subnet not found
	return nil, fmt.Errorf("Failed to find VPC subnet: %s", name)

}

// getKeyIDFromName is used to get a VPC key ID from its name.
func (v VPCService) getKeyIDFromName(name string) (*string, error) {
	listKeysOptions := &vpcv1.ListKeysOptions{}
	keys, response, err := v.service.ListKeys(listKeysOptions)
	if err != nil {
		return nil, fmt.Errorf("Failed to list VPC keys: %s. Response: %s", err.Error(), response)
	}

	for _, k := range keys.Keys {
		if *k.Name == name {
			return k.ID, nil
		}
	}

	// Key not found
	return nil, fmt.Errorf("Failed to find VPC key: %s", name)
}

func (v VPCService) getInstanceBootVolume(instance *vpcv1.Instance) *vpcv1.Volume {
	// Get the Boot volume
	getInstanceVolumeOptions := v.service.NewGetInstanceVolumeAttachmentOptions(*instance.ID, *instance.BootVolumeAttachment.ID)
	bva, _, err := v.service.GetInstanceVolumeAttachment(getInstanceVolumeOptions)
	if err != nil {
		return nil
	}
	getVolumeOptions := v.service.NewGetVolumeOptions(*bva.Volume.ID)
	bootVol, _, err := v.service.GetVolume(getVolumeOptions)
	if err != nil {
		return nil
	}
	return bootVol
}
