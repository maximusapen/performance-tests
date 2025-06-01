/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2019, 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package models

import (
	"fmt"
	"strings"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-model/model"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/consts"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/flag"
)

// Command constants
const (
	ControlLabel = "control"
	ServiceLabel = "cluster"

	MasterReadyStatus = "Ready"

	NamespaceAddon      = "addon"
	NamespaceALB        = "alb"
	NamespaceCluster    = "cluster"
	NamespaceDNS        = "dns"
	NamespaceEndpoint   = "endpoint"
	NamespaceHost       = "host"
	NamespaceLabel      = "label"
	NamespaceLocation   = "location"
	NamespaceNLBDNS     = "nlb-dns"
	NamespaceTaint      = "taint"
	NamespaceWorker     = "worker"
	NamespaceWorkerPool = "worker-pool"
	NamespaceZone       = "zone"

	NamespaceObservability = "ob"
	NamespaceLogging       = "logging"
	NamespaceMonitoring    = "monitoring"

	NamespaceSatellite = "sat"

	CmdAdd        = "add"
	CmdAgent      = "agent"
	CmdAlign      = "align"
	CmdAssign     = "assign"
	CmdAttach     = "attach"
	CmdConfig     = "config"
	CmdCreate     = "create"
	CmdDisable    = "disable"
	CmdEnable     = "enable"
	CmdGet        = "get"
	CmdList       = "ls"
	CmdLogging    = "logging"
	CmdMonitoring = "monitoring"
	CmdRebalance  = "rebalance"
	CmdRegister   = "register"
	CmdReload     = "reload"
	CmdRemove     = "rm"
	CmdReplace    = "replace"
	CmdResize     = "resize"
	CmdSet        = "set"
	CmdSync       = "sync"
	CmdUpdate     = "update"
	CmdVersions   = "versions"

	// NameFlagName is the name of the flag for providing a name when creating something
	NameFlagName = "name"

	// The set of constants associated with the flags for the command names.
	AdminFlagName                        = "admin"
	AutomateFlagName                     = "automate"
	CancelFlagName                       = "cancel"
	ControlFlagName                      = "control"
	CoreOSEnabledFlagName                = "coreos-enabled"
	CosInstanceFlagName                  = "cos-instance"
	DestHostnameFlagName                 = "dest-hostname"
	DestPortFlagName                     = "dest-port"
	DestProtocolFlagName                 = "dest-protocol"
	DestTypeFlagName                     = "dest-type"
	DisableDiskEncryptionFlagName        = "disable-disk-encrypt"
	DNSSubdomainFlagName                 = "subdomain"
	EndpointFlagName                     = "endpoint"
	FlavorFlagName                       = "flavor"
	ForceFlagName                        = "force"
	ForceDeleteStorageFlagName           = "force-delete-storage"
	HardwareFlagName                     = "hardware" // used to indicate the the level of classic hardware isolation (dedicated or shared)
	HostFlagName                         = "host"
	HostLabelFlagName                    = "host-label"
	ImageFlagName                        = "image"
	InfrastructureTypeFlagName           = "infrastructure-type"
	InstanceFlagName                     = "instance"
	IPFlagName                           = "ip"
	IsolationPublic                      = "public"
	IsolationPrivate                     = "private"
	JSONOutFlagName                      = "json" // used to print the command output in JSON format
	KubeVersionFlagName                  = "kube-version"
	LabelFlagName                        = "label"
	LocationFlagName                     = "location"
	LoggingKeyFlagName                   = "logdna-ingestion-key"
	MachineTypeFlagName                  = "machine-type"
	ManagedFromFlagName                  = "managed-from"
	MasterPrivateServiceEndpointFlagName = "private-service-endpoint"
	MasterPublicServiceEndpointFlagName  = "public-service-endpoint"
	MetricsFlagName                      = "metrics"
	NetworkFlagName                      = "network"
	NLBHostFlagName                      = "nlb-host"
	NoSubnetFlagName                     = "no-subnet"
	OperatingSystemFlagName              = "operating-system"
	PodSubnetFlagName                    = "pod-subnet"
	PollIntervalFlagName                 = "poll-interval"
	PublicKeyFlagName                    = "public-key"
	PrivateKeyFlagName                   = "private-key"
	ProvisionedFlagName                  = "provisioned"
	ProviderIDFlagName                   = "provider"
	ReloadFlagName                       = "reload"
	ServiceSubnetFlagName                = "service-subnet"
	SizePerZoneFlagName                  = "size-per-zone"
	SourceProtocolFlagName               = "source-protocol"
	SSHKeyFlagName                       = "ssh-key"
	SubnetIDFlagName                     = "subnet-id"
	SuffixFlagName                       = "suffix"
	TaintFlagName                        = "taint"
	TimeoutFlagName                      = "timeout"
	UpdateFlagName                       = "update"
	VPCIDFlagName                        = "vpc-id"
	WorkerFailuresFlagName               = "max-worker-failures"

	// ClusterFlag ShortName, Name and Value are the names and description of the flag for identifying the cluster
	ClusterFlagShortName = "c"
	ClusterFlagName      = "cluster"
	ClusterFlagValue     = "cluster name or ID"

	// Worker flags
	WorkerFlagShortName = "w"
	WorkerFlagName      = "worker"
	WorkerFlagValue     = "worker ID"
	WorkersFlagName     = "workers"

	// Worker Pool flags
	WorkerPoolFlagShortName = "p"
	WorkerPoolFlagName      = "worker-pool"
	WorkerPoolFlagValue     = "worker-pool ID"
	WorkerPoolsFlagName     = "worker-pools"

	// Zone Flags
	ZoneFlagName  = "zone"
	ZoneFlagValue = "zone name"

	// Name of flag to specify the number of clusters to create/delete
	QuantityFlagName = "quantity"

	// Name of flag to specify the number of threads to be used for parallel requests
	ThreadFlagName = "threads"

	//PermittedHardwareValues is the set of options for the hardware flag
	PermittedHardwareValues = "dedicated | shared"

	// Private and Public VLAN Flag Names is the name of the flag used to specify the ID of the public VLAN (Classic)
	PrivateVLANFlagName = "private-vlan"
	PublicVLANFlagName  = "public-vlan"

	// Provider type subcommand names
	ProviderClassic    = model.ProviderClassicExternal
	ProviderVPCClassic = model.ProviderVPCClassicExternal
	ProviderVPCGen2    = model.ProviderVPCGen2External
	ProviderSatellite  = model.ProviderUPIExternal

	AccountManagementCategory = "Account Management Commands"
	ClusterManagementCategory = "Cluster Management Commands"
	ClusterComponentsCategory = "Cluster Components Commands"
	InformationalCategory     = "Informational Commands"
	ObservabilityCategory     = "Observability Configuration Commands"
	LoggingCategory           = "Logging Commands"
	MonitoringCategory        = "Monitoring Commands"
	SatelliteCategory         = "Satellite Management Commands"
)

var (
	// AdminFlag is ued to retreive administrator certificates and keys
	AdminFlag = cli.BoolFlag{
		Name:  AdminFlagName,
		Usage: "Retrieve administrator certificates and PEM keys.",
	}

	// InfrastructureTypeFlag is used to specify the cloud infrastructure type (e.g. 'classic', 'vpc-gen2')
	InfrastructureTypeFlag = cli.StringFlag{
		Name:  InfrastructureTypeFlagName,
		Usage: "Infrastructure type for hosts ('classic' or 'vpc-gen2').",
	}

	// JSONOutFlag is used to print the command output in JSON format
	JSONOutFlag = cli.BoolFlag{
		Name:     JSONOutFlagName,
		Required: false,
		Usage:    "Prints the command output in JSON format.",
	}

	// MetricsFlag is used to request that performance metrics are generated
	MetricsFlag = cli.BoolFlag{
		Name:     MetricsFlagName,
		Required: false,
		Usage:    "Sends metrics to our metrics service (currently influx db).",
	}

	// NetworkFlag is used to retrieve the Calico network configuration. Only valid in conjunction with the AdminFlag.
	NetworkFlag = cli.BoolFlag{
		Name:  NetworkFlagName,
		Usage: "Retrieve Calico network configuration.",
	}

	// QuantityFlag is used to request that the command is run against the specified number of clusters
	QuantityFlag = flag.UintFlag{
		Require:    false,
		TargetName: QuantityFlagName,
		UintFlag: cli.UintFlag{
			Name:  QuantityFlagName,
			Usage: "The number of clusters.",
			Value: consts.DefaultClusterQuantity,
		},
	}

	// ThreadFlag is used to request that the command is run against the specified number of clusters
	ThreadFlag = flag.UintFlag{
		Require:    false,
		TargetName: ThreadFlagName,
		UintFlag: cli.UintFlag{
			Name:  ThreadFlagName,
			Usage: "The number of parallel requests.",
			Value: consts.DefaultThreads,
		},
	}

	// SuffixStrFlag is used to request that a textual suffix is added to a resource name
	SuffixStrFlag = cli.StringFlag{
		Name:  SuffixFlagName,
		Usage: "Add textual suffix to a resources's name.",
	}

	// SuffixFlag is used to request that a numerical suffix is added to a resource name
	SuffixFlag = cli.BoolFlag{
		Name:  SuffixFlagName,
		Usage: "Add numerical suffix to a cluster's name.",
	}

	// WorkerPoolNameFlag is the flag for naming a worker pool
	WorkerPoolNameFlag = cli.StringFlag{
		Name:  NameFlagName,
		Usage: "Name of worker pool.",
	}
	// RequiredWorkerPoolNameFlag is the required flag for providing a worker pool name
	RequiredWorkerPoolNameFlag = flag.StringFlag{
		Require:    true,
		StringFlag: WorkerPoolNameFlag,
	}

	// ClusterFlag is the flag for identifying the cluster
	ClusterFlag = cli.StringFlag{
		Name: strings.Join([]string{
			ClusterFlagName,
			ClusterFlagShortName,
		}, ", "),
		Usage: "Specify the cluster name or ID.",
	}
	// RequiredClusterFlag is the required flag for identifying the cluster
	RequiredClusterFlag = flag.StringFlag{
		Require:    true,
		StringFlag: ClusterFlag,
	}

	// WorkerFlag is the flag for identifying a worker node
	WorkerFlag = cli.StringFlag{
		Name: strings.Join([]string{
			WorkerFlagName,
			WorkerFlagShortName,
		}, ", "),
		Usage: "Specify the worker ID.",
	}
	// RequiredWorkerFlag is the required flag for identifying a worker node
	RequiredWorkerFlag = flag.StringFlag{
		Require:    true,
		StringFlag: WorkerFlag,
	}

	// ForceDeleteStorageFlag requetss that a cluster's persistent storage is removed - default to true
	ForceDeleteStorageFlag = cli.BoolTFlag{
		Name:  ForceDeleteStorageFlagName,
		Usage: "Force the removal of a cluster's persistent storage. (default: true)",
	}

	// WorkerPoolFlag is the flag for identifying a worker pool
	WorkerPoolFlag = cli.StringFlag{
		Name: strings.Join([]string{
			WorkerPoolFlagName,
			WorkerPoolFlagShortName,
		}, ", "),
		Usage: "Specify the worker Pool Name or ID.",
	}
	// RequiredWorkerPoolFlag is the required flag for identifying a worker pool
	RequiredWorkerPoolFlag = flag.StringFlag{
		Require:    true,
		StringFlag: WorkerPoolFlag,
	}

	// SizePerZoneFlag is the flag used to specify the size of a worker pool in each zone
	SizePerZoneFlag = cli.StringFlag{
		Name:  SizePerZoneFlagName,
		Usage: "The number of workers per zone.",
	}
	// RequiredSizePerZoneFlag is the required flag used to specify the size of a worker pool in each zone
	RequiredSizePerZoneFlag = flag.StringFlag{
		Require:    true,
		StringFlag: SizePerZoneFlag,
	}

	// MachineTypeFlag is the flag used to specify the machine type for a worker
	MachineTypeFlag = cli.StringFlag{
		Name:  MachineTypeFlagName,
		Usage: "The flavor of the worker node.",
	}

	// ZoneFlag is used to specify a zone/datacenter
	ZoneFlag = cli.StringFlag{
		Name:  ZoneFlagName,
		Usage: "Specify the zone for the resource.",
	}

	// RequiredZoneFlag is the required flag for specifying a zone
	RequiredZoneFlag = flag.StringFlag{
		Require:    true,
		StringFlag: ZoneFlag,
	}

	// LocationFlag is used to specify a satellite location
	LocationFlag = cli.StringFlag{
		Name:  LocationFlagName,
		Usage: "Specify the name or ID of the Satellite location.",
	}

	// RequiredLocationFlag is the required flag for specifying a satellite location
	RequiredLocationFlag = flag.StringFlag{
		Require:    true,
		StringFlag: LocationFlag,
	}

	// DNSSubdomainFlag is used to specify a location's DNS sub domain
	DNSSubdomainFlag = cli.StringFlag{
		Name:  DNSSubdomainFlagName,
		Usage: "The DNS subdomain.",
	}

	// RequiredDNSSubdomainFlag is the required flag for specifying a location's DNS sub domain
	RequiredDNSSubdomainFlag = flag.StringFlag{
		Require:    true,
		StringFlag: DNSSubdomainFlag,
	}

	// DNSIPFlag is used to specify host IP address(es)
	DNSIPFlag = cli.StringSliceFlag{
		Name:  IPFlagName,
		Usage: "The associated public IP address(es).",
	}

	// RequiredDNSIPFlag is the required flag to specify a control plane host IP address
	RequiredDNSIPFlag = flag.StringSliceFlag{
		Require:         true,
		Repeat:          true,
		StringSliceFlag: DNSIPFlag,
	}

	// HostFlag is used to specify a satellite host
	HostFlag = cli.StringFlag{
		Name:  HostFlagName,
		Usage: "Specify the name or ID of the host for use with your Satellite control plane or cluster.",
	}

	// RequiredHostFlag is the required flag for specifying a satellite host
	RequiredHostFlag = flag.StringFlag{
		Require:    true,
		StringFlag: HostFlag,
	}

	// HostsFlag is the flag used to provide the names of a set of Satellite hosts
	HostsFlag = cli.StringSliceFlag{
		Name:  HostFlagName,
		Usage: "Host name(s).",
	}

	// CancelFlag is used to specify whether a Satellite host should be cancelled(deleted) on removal
	CancelFlag = cli.BoolFlag{
		Name:  CancelFlagName,
		Usage: "Cancel Iaas Host.",
	}

	// ReloadFlag is used to specify whether a Satellite host should be OS reloaded on removal
	ReloadFlag = cli.BoolFlag{
		Name:  ReloadFlagName,
		Usage: "Reload host OS.",
	}

	// PublicKeyFlag is used to specify the name or identifier of a public key held in Iaas provider
	PublicKeyFlag = cli.StringFlag{
		Name:  PublicKeyFlagName,
		Usage: "Public key name or id.",
	}

	// PrivateKeyFlag is used to specify the loacaiton of a private key file
	PrivateKeyFlag = cli.StringFlag{
		Name:  PrivateKeyFlagName,
		Usage: "Private key file location.",
	}

	// AutomateFlag is used to specify whether the attachment script should be run automatically.
	AutomateFlag = cli.BoolFlag{
		Name:  AutomateFlagName,
		Usage: "Automatically run Satellite attachment script on host(s)",
	}

	// ControlFlag is used to specify whether the attachment script should be run automatically.
	ControlFlag = cli.BoolFlag{
		Name:  ControlFlagName,
		Usage: "Satellite Control Plane or Cluster hosts",
	}

	// ForceFlag is used to override preconfigured hosts cancellation prevention
	ForceFlag = cli.BoolFlag{
		Name:  ForceFlagName,
		Usage: "Force cancellation of hosts",
	}

	// ProvisionedFlag is used to reuqest procesing of hosts the have been provisioned, but not attached/assigned
	ProvisionedFlag = cli.BoolFlag{
		Name:  ProvisionedFlagName,
		Usage: "Process provisioned hosts only",
	}

	// ManagedFromFlag is used to specify the MZR from which a Satellite cluster will be managed
	ManagedFromFlag = cli.StringFlag{
		Name:  ManagedFromFlagName,
		Usage: "Specify the name of the multizone metro in IBM Cloud to manage your Satellite location from.",
	}

	// ImageFlag is used to specify the image of the provisioned host
	ImageFlag = cli.StringFlag{
		Name:  ImageFlagName,
		Usage: "Specify the name of the image of the hosts you want to attach to your location.",
	}

	// OperatingSystemFlag is used to specify the OS
	OperatingSystemFlag = cli.StringFlag{
		Name:  OperatingSystemFlagName,
		Usage: "Specify the name of the operating system of the worker nodes or Satellite location/cluster hosts.",
	}

	// RequiredManagedFromFlag is the required flag for specifying a MZR for management of a Satellite cluster
	RequiredManagedFromFlag = flag.StringFlag{
		Require:    true,
		StringFlag: ManagedFromFlag,
	}

	// CoreOSEnabledFlag is the flag used to specify whether CoreOS features should be enabled for the Satellite location
	CoreOSEnabledFlag = cli.BoolFlag{
		Name:  CoreOSEnabledFlagName,
		Usage: "Enable CoreOS features for the Satellite location.",
	}

	// VPCIDFlag is the flag used to specify the VPC identifier
	VPCIDFlag = cli.StringFlag{
		Name:  VPCIDFlagName,
		Usage: "The ID of the VPC in which to create the cluster.",
	}
	// RequiredVPCIDFlag is the required flag used to specify the VPC identifier
	RequiredVPCIDFlag = flag.StringFlag{
		Require:    true,
		StringFlag: VPCIDFlag,
	}

	// ProviderIDFlag used to set provider ID
	ProviderIDFlag = cli.StringFlag{
		Name:  ProviderIDFlagName,
		Usage: "The provider type ID of the VPC worker node.",
	}

	// ProviderForListFlag used to select the provider for list operations
	ProviderForListFlag = cli.StringFlag{
		Name: ProviderIDFlagName,
		Usage: fmt.Sprintf("Filter the list for a specific infrastructure provider. Supported values are %s, %s, %s and %s.",
			ProviderClassic, ProviderVPCClassic, ProviderVPCGen2, ProviderSatellite),
	}
	// RequiredProviderForListFlag is the required  flag used to select the provider for list operations
	RequiredProviderForListFlag = flag.StringFlag{
		Require:    true,
		StringFlag: ProviderForListFlag,
	}

	// UpdateFlag is used to attempt an update even if the change is greater than two minor versions
	UpdateFlag = cli.BoolFlag{
		Name:  UpdateFlagName,
		Usage: "Optional: Update the worker node to the same major and minor version of the master and the latest patch.",
	}

	// SubnetIDFlag used to set VPC subnet ID
	SubnetIDFlag = cli.StringFlag{
		Name:  SubnetIDFlagName,
		Usage: "The VPC subnet to assign the cluster.",
	}
	// RequiredSubnetIDFlag is the required flag for specifying a subnet
	RequiredSubnetIDFlag = flag.StringFlag{
		Require:    true,
		StringFlag: SubnetIDFlag,
	}

	// NoSubnetFlag is used to request that a portable subnet is not created
	NoSubnetFlag = cli.BoolFlag{
		Name:  NoSubnetFlagName,
		Usage: "Prevent the creation of a portable subnet when creating a cluster",
	}

	// PodSubnetFlag is an optional flag for specifying a CIDR subnet for pod IPs
	PodSubnetFlag = cli.StringFlag{
		Name:  PodSubnetFlagName,
		Usage: "Optional: Custom subnet CIDR to provide private IP addresses for pods. The subnet must be at least '/23' or larger.",
	}

	// ServiceSubnetFlag is an optional flag for specifying a CIDR subnet for service IPs
	ServiceSubnetFlag = cli.StringFlag{
		Name:  ServiceSubnetFlagName,
		Usage: "Optional: Custom subnet CIDR to provide private IP addresses for services. The subnet must be at least '/24' or larger.",
	}

	// MasterPrivateServiceEndpointFlag is used to enable the private service endpoint
	MasterPrivateServiceEndpointFlag = cli.BoolTFlag{
		Name:  MasterPrivateServiceEndpointFlagName,
		Usage: "Enable the private service endpoint to make the master privately accessible. (default: true)",
	}

	// MasterPublicServiceEndpointFlag is used to enable the public service endpoint
	MasterPublicServiceEndpointFlag = cli.BoolTFlag{
		Name:  MasterPublicServiceEndpointFlagName,
		Usage: "Enable the public service endpoint to make the master publicly accessible. (default: true)",
	}

	// PollIntervalFlag used to set resource (e.g. cluster, master or worker) polling interval
	PollIntervalFlag = cli.StringFlag{
		Name:  PollIntervalFlagName,
		Usage: "The interval to poll the status of the resource.",
	}

	// TimeoutFlag used to specify how to long to wait before aborting an operation
	TimeoutFlag = cli.StringFlag{
		Name:  TimeoutFlagName,
		Usage: "The timeout period for the operation.",
	}

	// HardwareFlag used to specify whether hardware should be shared or dedicated
	HardwareFlag = cli.StringFlag{
		Name:  HardwareFlagName,
		Usage: "The type of Hardware (dedicated or shared).",
	}

	// WorkerFailuresFlag used to specify the maximum number of deploy failures that will trigger an auto worker reload/replace
	WorkerFailuresFlag = cli.IntFlag{
		Name:  WorkerFailuresFlagName,
		Usage: "Maximum number of worker provision/deploy failure retries",
	}

	// LabelsFlag is the flag used to provide the names of a set of labels
	LabelsFlag = cli.StringSliceFlag{
		Name:  LabelFlagName,
		Usage: "Label name(s) in 'key=value' pairs.",
	}

	// RequiredLabelsFlag is the required flag to specify a set of labels
	RequiredLabelsFlag = flag.StringSliceFlag{
		Require:         true,
		Repeat:          true,
		StringSliceFlag: LabelsFlag,
	}

	// HostLabelsFlag is the flag used to provide the names of a set of host labels
	HostLabelsFlag = cli.StringSliceFlag{
		Name:  HostLabelFlagName,
		Usage: "Host Labels",
	}

	// LabelFlagFormat describes the usage of the labels flag
	LabelFlagFormat = fmt.Sprintf("--%s KEY=VALUE [--%s KEY=VALUE ...]", LabelFlagName, LabelFlagName)

	// Endpoint flags

	// EndpointFlag is used to specify a satellite endpoint
	EndpointFlag = cli.StringFlag{
		Name:  EndpointFlagName,
		Usage: "Specify the ID of the Satellite endpoint.",
	}

	// RequiredEndpointFlag is the required flag for specifying a Satellite endpoint
	RequiredEndpointFlag = flag.StringFlag{
		Require:    true,
		StringFlag: EndpointFlag,
	}

	// SourceProtocolFlag is used to specify the protocol for a Satellite endpoint source host
	SourceProtocolFlag = cli.StringFlag{
		Name:  SourceProtocolFlagName,
		Usage: "Specify the protocol the source must use to connect to the destination resource.",
	}
	// RequiredSourceProtocolFlag is the required flag for specifying a Satellite endpoint's source protocol
	RequiredSourceProtocolFlag = flag.StringFlag{
		Require:    true,
		StringFlag: SourceProtocolFlag,
	}

	// DestTypeFlag is used to specify the type of Satellite endpoint (cloud or location)
	DestTypeFlag = cli.StringFlag{
		Name:  DestTypeFlagName,
		Usage: "Specify the place where the destination resource runs, either in IBM Cloud (cloud) or your Satellite locaiton (location)",
	}

	// RequiredDestTypeFlag is the required flag for specifying the type of Satellite endpoint (cloud or location)
	RequiredDestTypeFlag = flag.StringFlag{
		Require:    true,
		StringFlag: DestTypeFlag,
	}

	// DestHostnameFlag is used to specify a Satellite endpoint's destination hostname
	DestHostnameFlag = cli.StringFlag{
		Name:  DestHostnameFlagName,
		Usage: "Specify the URL or externally accessible IP address of the destination resource.",
	}

	// RequiredDestHostnameFlag is the required flag for specifying a Satellite endpoint's destination hostname
	RequiredDestHostnameFlag = flag.StringFlag{
		Require:    true,
		StringFlag: DestHostnameFlag,
	}

	// DestPortFlag is used to specify a Satellite endpoint's destination port
	DestPortFlag = cli.UintFlag{
		Name:  DestPortFlagName,
		Usage: "Specify the port that the destination listens on for incoming requests.",
	}

	// RequiredDestPortFlag is the required flag for specifying a Satellite endpoint's destination port
	RequiredDestPortFlag = flag.UintFlag{
		Require:  true,
		UintFlag: DestPortFlag,
	}

	// DestProtocolFlag is used to specify the protocol for a Satellite endpoint destination host
	DestProtocolFlag = cli.StringFlag{
		Name:  DestProtocolFlagName,
		Usage: "Specify the protocol of the destination resource.",
	}

	// RequiredDestProtocolFlag is the required flag for specifying a Satellite endpoint's destination protocol
	RequiredDestProtocolFlag = flag.StringFlag{
		Require:    true,
		StringFlag: DestProtocolFlag,
	}

	// NLBHostFlag is the flag for specifying the NLB NDS hostname
	NLBHostFlag = cli.StringFlag{
		Name:  NLBHostFlagName,
		Usage: "Specify the hostname of the NLB resource.",
	}

	// RequiredNLBHostFlag is the required flag for specifying the NLB NDS hostname
	RequiredNLBHostFlag = flag.StringFlag{
		Require:    true,
		StringFlag: NLBHostFlag,
	}

	// LoggingKeyFlag is used to specify the Log Analysis ingestion key to be used for a logging configuration
	LoggingKeyFlag = cli.StringFlag{
		Name:  LoggingKeyFlagName,
		Usage: "Log analysis ingestion key.",
	}

	// LoggingInstanceFlag is used to specify the ID or name of the IBM Log Analysis service instance
	// to be used for a logging configuration
	LoggingInstanceFlag = cli.StringFlag{
		Name:  InstanceFlagName,
		Usage: "ID or name of the IBM Log Analysis service instance.",
	}

	// RequiredLoggingInstanceFlag is the required flag for specifying a zone
	RequiredLoggingInstanceFlag = flag.StringFlag{
		Require:    true,
		StringFlag: LoggingInstanceFlag,
	}

	// TaintFlag is used to specify a workerpool taint
	TaintFlag = cli.StringSliceFlag{
		Name:  TaintFlagName,
		Usage: "Sets taints on all the workers in the worker pool. Specify the Kubernetes taint in the format 'key=value:effect'. The 'key=value' is a pair such as 'env=prod' that you use to manage the worker node taint and matching pod tolerations. The 'effect' is a Kubernetes taint effect such as 'NoSchedule' that describes how the taint works.",
	}

	// RequiredTaintFlag is the required flag to specify a workerpool taint
	RequiredTaintFlag = flag.StringSliceFlag{
		Require:         true,
		Repeat:          true,
		StringSliceFlag: TaintFlag,
	}

	// TaintFlagFormat describes the usage of the taint flag
	TaintFlagFormat = fmt.Sprintf("--%s KEY=VALUE:EFFECT [--%s KEY=VALUE:EFFECT ...]", TaintFlagName, TaintFlagName)
)
