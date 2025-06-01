
package cliutils

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/IBM-Cloud/ibm-cloud-cli-sdk/bluemix/authentication/iam"
	"github.com/IBM-Cloud/ibm-cloud-cli-sdk/common/rest"

	"github.ibm.com/alchemy-containers/armada-performance/tools/crypto/utils"
)

var mutex sync.Mutex

// encyptionKey is the key used to decrypt sensitive data from the configuration file(s).
// It's value is baked in the executable at build time
var encryptionKey string

// ArmadaConfig defines all the metadata for working with IBM Cloud and associated services
type ArmadaConfig struct {
	IKS      *IKSConfig
	SatLink  *SatLinkConfig
	VPC      *VPCConfig
	IBMCloud *IBMCloudConfig
}

// IKSConfig defines metadata for accessing the IBM Cloud Kubernetes Service
type IKSConfig struct {
	Endpoint string `toml:"endpoint"`
	Region   string `toml:"region"`
}

// SatLinkConfig defines metadata for accessing the Satellite Link API
type SatLinkConfig struct {
	Endpoint string `toml:"endpoint"`
}

// VPCConfig defines metadata for accessing the VPC Iaas API
type VPCConfig struct {
	Endpoint string `toml:"endpoint"`
}

// IBMCloudConfig defines metadata for accessing IBM Cloud
type IBMCloudConfig struct {
	AccessToken  string
	RefreshToken string

	IAMEndpoint string `toml:"iam_endpoint"`
	AccountID   string `toml:"account_id"`
	Username    string `toml:"username"`
	APIKey      string `toml:"api_key"`

	InfraAPIKey      string `toml:"infra_api_key"`
	InfraIAMEndpoint string `toml:"infra_iam_endpoint"`

	ClassicIaasUsername string `toml:"classic_infra_username"`
	ClassicIaasAPIKey   string `toml:"classic_infra_api_key"`
}

// TomlInfrastructureConfig is used for cluster and worker creation
type TomlInfrastructureConfig struct {
	Classic    *TomlClassicInfrastructureConfig   `toml:"classic"`
	VPCClassic *TomlVPCInfrastructureConfig       `toml:"vpc-classic"`
	VPC        *TomlVPCInfrastructureConfig       `toml:"vpc-gen2"`
	Satellite  *TomlSatelliteInfrastructureConfig `toml:"satellite"`
}

// TomlInfrastructureOSConfig provides a mapping between Operating System name and Iaas provider image name
type TomlInfrastructureOSConfig map[string]string

// TomlClassicInfrastructureConfig defines Classic Infrastructure specific configuration data
type TomlClassicInfrastructureConfig struct {
	Billing               string                     `toml:"billing,omitempty"`
	DisableDiskEncryption bool                       `toml:"disable_disk_encryption,omitempty"`
	Flavor                string                     `toml:"flavor,omitempty"`
	Isolation             string                     `toml:"isolation,omitempty"`
	PrivateVlan           string                     `toml:"private_vlan,omitempty"`
	PublicVlan            string                     `toml:"public_vlan,omitempty"`
	Trusted               bool                       `toml:"trusted,omitempty"`
	PortableSubnet        bool                       `toml:"portable_subnet,omitempty"`
	PrivateOnly           bool                       `toml:"privateOnly,omitempty"`
	Zone                  string                     `toml:"zone,omitempty"`
	SSHKey                int                        `toml:"ssh_key,omitempty"` // SSH Key Identifier
	OperatingSystems      TomlInfrastructureOSConfig `toml:"operating_systems,omitempty"`
}

// TomlVPCInfrastructureConfig defines VPC Infrastructure specific configuration data
type TomlVPCInfrastructureConfig struct {
	DisableDiskEncryption bool                                      `toml:"disable_disk_encryption,omitempty"`
	Flavor                string                                    `toml:"flavor,omitempty"`
	ID                    string                                    `toml:"id,omitempty"`
	Zone                  string                                    `toml:"zone,omitempty"`
	CosInstance           string                                    `toml:"cos_instance,omitempty"` // Required for ROKS on VPC-Gen2
	Locations             map[string]TomlVPCInfrastructureLocations `toml:"locations,omitempty"`
	SSHKey                string                                    `toml:"ssh_key,omitempty"` // SSH Key Name
	OperatingSystems      TomlInfrastructureOSConfig                `toml:"operating_systems,omitempty"`
}

// TomlVPCInfrastructureLocations defines VPC zone/location configuration data
type TomlVPCInfrastructureLocations struct {
	Location string `toml:"location"`
	SubnetID string `toml:"subnet_id"`
}

// TomlSatelliteInfrastructureConfig defines IBM Cloud Satellite configuration data
type TomlSatelliteInfrastructureConfig struct {
	Locations map[string]TomlSatelliteInfrastructureLocation `toml:"location"`
}

// TomlSatelliteInfrastructureLocation defines IBM Cloud Satellite location configuration data
type TomlSatelliteInfrastructureLocation struct {
	Name          string `toml:"name"`          // Location name
	Preconfigured bool   `toml:"preconfigured"` // Dynamic or preconfigured location
	ID            string `toml:"ID,omitempty"`  // Location ID name

	ManagedFrom    string                           `toml:"managed_from,omitempty"` // The zone in an IBM Cloud MZR that Satellite control plane resources are managed from
	CoresOSEnabled bool                             `toml:"coreos_enabled"`
	Hosts          TomlSatelliteInfrastructureHosts `toml:"hosts"`
}

// TomlSatelliteInfrastructureHosts defines IBM Cloud Satellite host configuration data
type TomlSatelliteInfrastructureHosts struct {
	Provisioned TomlSatelliteInfrastructureHostConfig `toml:"provisioned,omitempty"` // Compute hosts(s) identifiers that have been ordered/provisioned, but not attached to the location
	Control     TomlSatelliteInfrastructureHostConfig `toml:"control"`               // Compute host(s) to be assigned to Satellite control plane
	Cluster     TomlSatelliteInfrastructureHostConfig `toml:"cluster"`               // Compute host(s) to be assigned to Satellite clusters
}

// TomlSatelliteInfrastructureHostConfig defines host configuration data
type TomlSatelliteInfrastructureHostConfig struct {
	IaasType string                                     `toml:"iaas_type,omitempty"`
	OS       string                                     `toml:"os,omitempty"`
	Servers  map[string]TomlSatelliteInfrastructureHost `toml:"servers"`
}

// TomlSatelliteInfrastructureHost defines a IBM Cloud Satellite host
type TomlSatelliteInfrastructureHost struct {
	Hostname   string `toml:"hostname,omitempty"`
	Cluster    string `toml:"cluster,omitempty"`
	Quantity   int    `toml:"quantity,omitempty"`
	CPU        int    `toml:"cpu,omitempty"`
	Memory     int    `toml:"memory,omitempty"`
	Disk       int    `toml:"disk,omitempty"`
	IaasID     string `toml:"iaas_id,omitempty"`
	IP         string `toml:"ip_address,omitempty"`
	Datacenter string `toml:"datacenter,omitempty"`
	Zone       string `toml:"zone,omitempty"`

	Classic TomlSatelliteInfrastructureClassicHost `toml:"classic,omitempty"`
	VPC     TomlSatelliteInfrastructureVPCHost     `toml:"vpc,omitempty"`
}

// TomlSatelliteInfrastructureClassicHost definesIBM Cloud Classic Satellite host metadata
type TomlSatelliteInfrastructureClassicHost struct {
	PrivateVLAN string `toml:"privateVLAN,omitempty"`
	PublicVLAN  string `toml:"publicVLAN,omitempty"`
}

// TomlSatelliteInfrastructureVPCHost defines IBM Cloud VPC Satellite host metadata
type TomlSatelliteInfrastructureVPCHost struct {
	FloatingIP string `toml:"floating_ip,omitempty"`
	Subnet     string `toml:"subnet,omitempty"`
	VPC        string `toml:"vpc,omitempty"`
}

var infrastructureConfig *TomlInfrastructureConfig
var armadaConfig *ArmadaConfig

// GetInfrastructureConfig returns Infrastructure Configuration information
func GetInfrastructureConfig() *TomlInfrastructureConfig {
	mutex.Lock()
	defer mutex.Unlock()

	if infrastructureConfig == nil {
		infrastructureConfig = new(TomlInfrastructureConfig)

		configFile := filepath.Join(getConfigPath(), "perf-infrastructure.toml")
		err := parseConfig(configFile, infrastructureConfig)
		if err != nil {
			log.Fatalf("Error parsing infrastructure configuration file %s : %s\n", configFile, err.Error())
		}
	}

	return infrastructureConfig
}

// UpdateSatelliteInfrastructureConfig updates the Satellite data within the infrastructure configuration file
func UpdateSatelliteInfrastructureConfig(satelliteConfig *TomlSatelliteInfrastructureConfig) {
	config := GetInfrastructureConfig()

	configFile := filepath.Join(getConfigPath(), "perf-infrastructure.toml")

	mutex.Lock()
	defer mutex.Unlock()

	f, err := os.OpenFile(configFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatalf("Error updating Satellite location confiugration file : %s\n", err.Error())
	}

	config.Satellite.Locations = satelliteConfig.Locations

	te := toml.NewEncoder(f)
	te.Encode(config)

	infrastructureConfig = config
}

// UpdateLocationInfrastructureConfig updates the Satellite location data within the infrastructure configuration file
func UpdateLocationInfrastructureConfig(locationConfig TomlSatelliteInfrastructureLocation) {
	config := GetInfrastructureConfig()

	configFile := filepath.Join(getConfigPath(), "perf-infrastructure.toml")

	mutex.Lock()
	defer mutex.Unlock()

	f, err := os.OpenFile(configFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatalf("Error updating Satellite confiugration file : %s\n", err.Error())
	}

	config.Satellite.Locations[locationConfig.Name] = locationConfig

	te := toml.NewEncoder(f)
	te.Encode(config)

	infrastructureConfig = config
}

// GetDefaultLocation returns the default location configuration
func GetDefaultLocation() TomlSatelliteInfrastructureLocation {
	return GetInfrastructureConfig().Satellite.Locations["default"]
}

// GetArmadaConfig returns IBM Cloud Configuration information
func GetArmadaConfig() *ArmadaConfig {
	mutex.Lock()
	defer mutex.Unlock()
	if armadaConfig == nil {
		armadaConfig = new(ArmadaConfig)

		configFile := filepath.Join(getConfigPath(), "perf-metadata.toml")
		err := parseConfig(configFile, armadaConfig)
		if err != nil {
			log.Fatalf("Error parsing IBM Cloud configuration file %s : %s\n", configFile, err.Error())
		}

		// Check for override of username. Individuals should be using their own IBMid.
		if IBMId := os.Getenv("ARMADA_PERFORMANCE_USERNAME"); len(IBMId) > 0 {
			armadaConfig.IBMCloud.Username = IBMId
		}
		// Check for override of username. Individuals should be using their own classic infrastructure user.
		if iu := os.Getenv("ARMADA_PERFORMANCE_CLASSIC_INFRA_USERNAME"); len(iu) > 0 {
			armadaConfig.IBMCloud.ClassicIaasUsername = iu
		}

		// Need to decrypt api keys stored in config files
		if len(encryptionKey) > 0 {
			os.Setenv(utils.KeyEnvVar, encryptionKey)
		}

		// Get the IBM Cloud api key from the environment, or if not set, from the configuration file
		if apikey := os.Getenv("ARMADA_PERFORMANCE_API_KEY"); len(apikey) > 0 {
			armadaConfig.IBMCloud.APIKey = apikey // pragma: allowlist secret
		} else {
			if len(armadaConfig.IBMCloud.APIKey) > 0 {
				armadaConfig.IBMCloud.APIKey, err = utils.Decrypt(armadaConfig.IBMCloud.APIKey)
				if err != nil {
					log.Fatalf("Error decrypting IBM Cloud APIKey : %s\n", err.Error()) // pragma: allowlist secret
				}
			}
		}

		// Get the IBM Cloud infrastrcuture api key from the environment, or if not set, from the configuration file
		// For VPC, this enables us to provision hosts in either the staging or production environment.
		// If not specified, we'll assume any infrastructure will be created in the default environment
		if apikey := os.Getenv("ARMADA_PERFORMANCE_INFRA_API_KEY"); len(apikey) > 0 {
			armadaConfig.IBMCloud.InfraAPIKey = apikey // pragma: allowlist secret
		} else {
			if len(armadaConfig.IBMCloud.InfraAPIKey) > 0 {
				armadaConfig.IBMCloud.InfraAPIKey, err = utils.Decrypt(armadaConfig.IBMCloud.InfraAPIKey)
				if err != nil {
					log.Fatalf("Error decrypting IBM Cloud Infrastructure APIKey : %s\n", err.Error()) // pragma: allowlist secret
				}
			} else {
				// If not set, default to the standard IBM Cloud APIKey
				armadaConfig.IBMCloud.InfraAPIKey = armadaConfig.IBMCloud.APIKey
			}
		}

		// Get the IBM Cloud classic infrastructure api key from the environment, or if not set, from the configuration file
		if apikey := os.Getenv("ARMADA_PERFORMANCE_CLASSIC_INFRA_API_KEY"); len(apikey) > 0 {
			armadaConfig.IBMCloud.ClassicIaasAPIKey = apikey // pragma: allowlist secret
		} else {
			if len(armadaConfig.IBMCloud.ClassicIaasAPIKey) > 0 {
				armadaConfig.IBMCloud.ClassicIaasAPIKey, err = utils.Decrypt(armadaConfig.IBMCloud.ClassicIaasAPIKey)
				if err != nil {
					log.Fatalf("Error decrypting IBM Cloud Classic Iaas APIKey : %s\n", err.Error()) // pragma: allowlist secret
				}
			}
		}

		// Get the IAM access token using api key
		if len(armadaConfig.IBMCloud.APIKey) > 0 {
			auth := iam.NewClient(iam.DefaultConfig(armadaConfig.IBMCloud.IAMEndpoint), rest.NewClient())
			token, err := auth.GetToken(iam.APIKeyTokenRequest(armadaConfig.IBMCloud.APIKey))

			if err != nil {
				log.Fatalf("Unable to get access token from IAM : %s", err.Error())
			}

			armadaConfig.IBMCloud.AccessToken = token.AccessToken
			armadaConfig.IBMCloud.RefreshToken = token.RefreshToken
		} else {
			log.Fatalf("No api key provided. Check environment / configuration file.")
		}

		// Finally ensure that the infrastructure IAM endpoint is set approriately, defaulting to the standard environment
		if armadaConfig.IBMCloud.InfraIAMEndpoint == "" {
			armadaConfig.IBMCloud.InfraIAMEndpoint = armadaConfig.IBMCloud.IAMEndpoint
		}
	}

	return armadaConfig
}

// ParseConfig ...
func parseConfig(filePath string, conf interface{}) error {
	_, err := toml.DecodeFile(filePath, conf)
	return err
}

// GetConfigPath returns path of toml config file
func getConfigPath() string {
	configPath := os.Getenv("APC2_CONFIG_PATH")
	if configPath != "" {
		return configPath
	}
	goPath := os.Getenv("GOPATH")
	perfSrcPath := filepath.Join("src", "github.ibm.com", "alchemy-containers", "armada-performance", "armada-perf-client2")
	return filepath.Join(goPath, perfSrcPath, "config")
}
