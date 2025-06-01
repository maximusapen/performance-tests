
package request

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	softlayer_datatypes "github.com/softlayer/softlayer-go/datatypes"
	bootstrap "github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/bootstrap"
	config "github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/config"
	deploy "github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/deploy"
	"github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/metrics"
	token "github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/tokens"
	"github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/util"
)

// Data defines information required for each API request
type Data struct {
	Action            config.ActionType
	ResponseTime      time.Duration
	ClusterName       string
	RequestNum        int
	WorkerID          string
	WorkerPoolName    string
	ZoneID            string
	TotalWorkers      int
	PoolSize          int
	Status            string
	StatusCode        int
	ContentType       string
	Body              []byte
	Metrics           metrics.RequestMetric
	ActionFailed      bool
	ActionTime        time.Duration
	ClusterID         string
	KubeUpdateVersion string
	Failure           FailureType
}

// FailureType defines an failure 'enum' for supported Aramda API requests
type FailureType int

// Failure enumerations
const (
	FailureUnspecified FailureType = iota
	FailureCloudflare
	FailureBackendIssue
)

var armadaURL url.URL
var armadaClustersURL url.URL

var debug bool
var verbose bool
var monitor bool
var mockDeploy *deploy.MockDeploy
var mockBootstrap *bootstrap.MockBootstrap
var blockingRequest bool
var postDataSized string
var machineType string
var kubeVersion string
var workerPoolName string
var (
	addClusterWorkersTemplate,
	workerPoolConfigTemplate,
	workerPoolSizeTemplate,
	workerZoneConfigTemplate []byte
)
var iamToken, refreshToken string
var requestCompleted *chan Data
var clusterCreateTemplateLines []string

var reqConf *config.Config

// InitRequests returns a request instance
func InitRequests(inmachineType string, inKubeVersion string, inWorkerPoolName string, conf *config.Config, inVerbose bool, inDebug bool, inMonitor bool, inMockDeploy *deploy.MockDeploy, inMockBootstrap *bootstrap.MockBootstrap, inNumThreads int, inRequestJobs *chan Data, inClusterComplete *chan Data) {
	debug = inDebug
	verbose = inVerbose
	monitor = inMonitor
	machineType = inmachineType
	kubeVersion = inKubeVersion
	workerPoolName = inWorkerPoolName
	mockDeploy = inMockDeploy
	mockBootstrap = inMockBootstrap
	requestCompleted = inClusterComplete

	blockingRequest = inMockBootstrap != nil

	reqConf = conf

	configPath := config.GetConfigPath()

	var err error

	Authenticate()

	// Read request body template for add worker requests
	// #nosec G304
	addClusterWorkersTemplate, err = ioutil.ReadFile(filepath.Join(configPath, config.GetConfigString("armada_add_workers_template", conf.Request.AddWorkers)))
	if err != nil {
		panic(err)
	}

	// Read request body template for worker pool creation requests
	// #nosec G304
	workerPoolConfigTemplate, err = ioutil.ReadFile(filepath.Join(configPath, config.GetConfigString("armada_worker_pool_config_template", conf.Request.CreateWorkerPool)))
	if err != nil {
		panic(err)
	}

	// #nosec G304
	workerPoolSizeTemplate, err = ioutil.ReadFile(filepath.Join(configPath, config.GetConfigString("armada_worker_pool_size_template", conf.Request.ResizeWorkerPool)))
	if err != nil {
		panic(err)
	}

	// #nosec G304
	workerZoneConfigTemplate, err = ioutil.ReadFile(filepath.Join(configPath, config.GetConfigString("armada_worker_pool_config_template", conf.Request.AddWorkerPoolZone)))
	if err != nil {
		panic(err)
	}

	// Read request body template for cluster creation requests
	// These requests should provide private/public VLAN data for cruisers only.
	// Patrols (free accounts) do not need this information
	// #nosec G304
	createTemplateFile, err := os.Open(filepath.Join(configPath, config.GetConfigString("armada_create_cluster_template", conf.Request.CreateCluster)))
	if err != nil {
		panic(err)
	}
	defer createTemplateFile.Close()

	// Read the file line by lline, ignoring VLAN data for free accounts
	scanner := bufio.NewScanner(createTemplateFile)
	var templateLines []string
	for scanner.Scan() {
		templateFileLine := scanner.Text()

		if strings.Contains(templateFileLine, "KUBEVERSION%") {
			if len(kubeVersion) > 0 {
				templateLines = append(templateLines, templateFileLine)
			}
			continue
		}

		if !(machineType == config.FreeAccountStr && strings.Contains(templateFileLine, "VLAN%")) {
			templateLines = append(templateLines, templateFileLine)
		}
	}
	if err = scanner.Err(); err != nil {
		panic(err)
	}

	clusterCreateTemplateLines = templateLines

	setPostDataSized()

	if conf.Softlayer != nil {
		// If we're running Armada Cluster with a dummy SL provider, we need to provide it with VLAN data
		if conf.Softlayer.SoftlayerDummy {
			etcdClientV3 := config.InitEtcdV3Client(conf.Etcd)

			var vlans []softlayer_datatypes.Network_Vlan

			vlanIds := []int{101, 100}
			vlanNetworks := []string{"PUBLIC", "PRIVATE"}

			dummyHost := "sl_dummy_hostname"
			dummyDatacenter := conf.Location.Datacenter
			theType := "STANDARD"

			var primaryRouter [2]softlayer_datatypes.Hardware_Router
			var dc softlayer_datatypes.Location
			var typ softlayer_datatypes.Network_Vlan_Type

			dc.Name = &dummyDatacenter

			primaryRouter[0].Hardware_Switch.Hardware.Hostname = &dummyHost
			primaryRouter[0].Hardware_Switch.Hardware.Datacenter = &dc
			primaryRouter[1].Hardware_Switch.Hardware.Hostname = &dummyHost
			primaryRouter[1].Hardware_Switch.Hardware.Datacenter = &dc

			typ.KeyName = &theType

			//datatypes.Network_Vlan.Prim
			vlans = []softlayer_datatypes.Network_Vlan{
				{Id: &vlanIds[0], Type: &typ, NetworkSpace: &vlanNetworks[0], PrimaryRouter: &primaryRouter[0]},
				{Id: &vlanIds[1], Type: &typ, NetworkSpace: &vlanNetworks[1], PrimaryRouter: &primaryRouter[1]},
			}
			for _, vlan := range vlans {
				// Serialise our object to JSON for storing
				vlanKey := strings.Join([]string{"/armada-cluster-test", "vlans", strconv.Itoa(*vlan.Id), "object"}, "/")

				vlanJSON, err := json.Marshal(vlan)
				if err != nil {
					log.Fatal(err)
				}
				vlanValue := string(vlanJSON)

				if _, err := util.V3PutWithRetry(etcdClientV3, *conf, vlanKey, vlanValue); err != nil {
					panic(err)
				}
			}
		}
	}

	armadaURL = url.URL{Scheme: conf.API.APIServerScheme, Host: config.GetConfigString("armada_api_server_ip", conf.API.APIServerIP) + ":" +
		config.GetConfigString("armada_api_server_port", conf.API.APIServerPort), Path: config.GetConfigString("armada_api_version", conf.API.APIVersion)}
	armadaClustersURL = url.URL{Scheme: conf.API.APIServerScheme, Host: config.GetConfigString("armada_api_server_ip", conf.API.APIServerIP) + ":" +
		config.GetConfigString("armada_api_server_port", conf.API.APIServerPort), Path: config.GetConfigString("armada_api_version", conf.API.APIVersion) +
		"/" + "clusters"}

	for j := 1; j <= inNumThreads; j++ {
		go worker(j, *inRequestJobs)
	}
}

// IsBlocking returns whether the request waits for the action to complete
func IsBlocking() bool {
	return blockingRequest
}

func setPostDataSized() {
	// Update template for create requests with machine size
	postDataSized = strings.Replace(strings.Join(clusterCreateTemplateLines, "\n"), "%MACHINETYPE%", machineType, 1)

	if len(kubeVersion) > 0 {
		postDataSized = strings.Replace(postDataSized, "%KUBEVERSION%", kubeVersion, 1)
	}
	postDataSized = strings.Replace(postDataSized, "%DATACENTER%", reqConf.Location.Environment+"-"+reqConf.Location.Datacenter, 1)

	if reqConf.Softlayer != nil {
		postDataSized = strings.Replace(postDataSized, "%PRIVATEVLAN%", reqConf.Softlayer.SoftlayerPrivateVLAN, 1)
		postDataSized = strings.Replace(postDataSized, "%PUBLICVLAN%", reqConf.Softlayer.SoftlayerPublicVLAN, 1)
		postDataSized = strings.Replace(postDataSized, "%BILLING%", reqConf.Softlayer.SoftlayerBilling, 1)
		postDataSized = strings.Replace(postDataSized, "%ISOLATION%", reqConf.Softlayer.SoftlayerIsolation, 1)
		postDataSized = strings.Replace(postDataSized, "%NOSUBNET%", strconv.FormatBool(!reqConf.Softlayer.SoftlayerPortableSubnet), 1)
		postDataSized = strings.Replace(postDataSized, "%DISKENCRYPTION%", strconv.FormatBool(reqConf.Softlayer.SoftlayerDiskEncryption), 1)
	}
}

func setCommonRequestHeaders(req *http.Request) {
	// Set headers that are mandatory for all requests
	// (Strictly speaking, not all requests require the iamToken, but does no harm to set it)
	req.Header.Set("Authorization", iamToken)
	req.Header.Set("X-Auth-Refresh-Token", refreshToken)

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
}

// Authenticate initializes the tokens necessary to talk to armada
func Authenticate() {
	if debug {
		fmt.Printf("%s\tAuthenticating via ", time.Now().Format(time.StampMilli))
	}
	configPath := config.GetConfigPath()

	// Get the authentication tokens for our user from Bluemix
	if !reqConf.Bluemix.BluemixDummy {
		// Get UAA and IAM tokens from Bluemix
		iamToken, refreshToken, _ = token.GetTokens(reqConf.Bluemix, debug)
	} else {
		if debug {
			fmt.Println("dummy IAM token")
		}
		// Get Dummy tokens from file
		var contents []byte
		var err error

		// #nosec G304
		contents, err = ioutil.ReadFile(filepath.Join(configPath, config.GetConfigString("armada_iam_token", reqConf.Bluemix.IAMToken)))
		iamToken = string(contents)
		if err != nil {
			panic(err)
		}
	}
}

// worker provides a pool of workers to submit API requests
func worker(id int, jobs <-chan Data) {
	for r := range jobs {
		if debug {
			fmt.Printf("%s\tRW: %d - %s for cluster %s\n", time.Now().Format(time.StampMilli), id, r.Action, r.ClusterName)
		}
		*requestCompleted <- PerformRequest(r, true)
	}
}

// SetKubeVersion sets the default kube version for new clusters
func SetKubeVersion(inKubeVersion string) {
	kubeVersion = inKubeVersion
	setPostDataSized()
}

// CreateClusterRequest creates a request that triggers a cluster create
// See https://containers.cloud.ibm.com/swagger-api/#/clusters/CreateCluster
func CreateClusterRequest(body string) *http.Request {
	if req, err := http.NewRequest("POST", armadaClustersURL.String(), bytes.NewBufferString(body)); err != nil {
		panic(err)
	} else {
		return req
	}
}

// GetClustersRequest creates a request that retrieves the list of clusters a user has access to
// See https://containers.cloud.ibm.com/swagger-api/#/clusters/GetClusters
func GetClustersRequest() *http.Request {
	if req, err := http.NewRequest("GET", armadaClustersURL.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// GetClusterRequest creates a request that retrieves the specified cluster's details
// See https://containers.cloud.ibm.com/swagger-api/#/clusters/GetClusters
func GetClusterRequest(clusterName string) *http.Request {
	buffer := bytes.NewBufferString(armadaClustersURL.String())
	buffer.WriteString("/")
	buffer.WriteString(clusterName)

	if req, err := http.NewRequest("GET", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		values := req.URL.Query()
		values.Add("showResources", strconv.FormatBool(reqConf.Request.ShowResources))
		req.URL.RawQuery = values.Encode()
		return req
	}
}

// GetClusterConfigRequest creates a request that retrieves the specified cluster's kube config
// See https://containers.cloud.ibm.com/swagger-api/#/clusters/GetClusterConfig
func GetClusterConfigRequest(clusterName string) *http.Request {
	buffer := bytes.NewBufferString(armadaClustersURL.String())
	buffer.WriteString("/")
	buffer.WriteString(clusterName)
	buffer.WriteString("/config")
	if reqConf.Request.AdminConfig {
		buffer.WriteString("/admin")
	}

	if req, err := http.NewRequest("GET", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// DeleteClusterRequest creates a request that deletes the specified cluster
// See https://containers.cloud.ibm.com/swagger-api/#/clusters/RemoveCluster
func DeleteClusterRequest(clusterName string) *http.Request {
	buffer := bytes.NewBufferString(armadaClustersURL.String())
	buffer.WriteString("/")
	buffer.WriteString(clusterName)
	if req, err := http.NewRequest("DELETE", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		// Always delete additional resources linked to the cluster
		values := req.URL.Query()
		values.Add("deleteResources", strconv.FormatBool(reqConf.Request.DeleteResources))
		req.URL.RawQuery = values.Encode()

		return req
	}
}

// GetWorkerPoolsRequest creates a request that lists the worker pools in a cluster
// See https://containers.cloud.ibm.com/swagger-api/#/clusters/GetWorkerPools
func GetWorkerPoolsRequest(clusterName string) *http.Request {
	buffer := bytes.NewBufferString(armadaClustersURL.String())
	buffer.WriteString("/")
	buffer.WriteString(clusterName)
	buffer.WriteString("/workerpools")

	if req, err := http.NewRequest("GET", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// CreateWorkerPoolRequest creates a request that creates a worker pool for a cluster
// See https://containers.cloud.ibm.com/swagger-api/#/clusters/CreateWorkerPool
func CreateWorkerPoolRequest(body string, clusterName string) *http.Request {
	buffer := bytes.NewBufferString(armadaClustersURL.String())
	buffer.WriteString("/")
	buffer.WriteString(clusterName)
	buffer.WriteString("/workerpools")

	if req, err := http.NewRequest("POST", buffer.String(), bytes.NewBufferString(body)); err != nil {
		panic(err)
	} else {
		return req
	}
}

// RemoveWorkerPoolRequest creates a request that removes aworker pool from a cluster
// See https://containers.cloud.ibm.com/swagger-api/#/clusters/RemoveWorkerPool
func RemoveWorkerPoolRequest(clusterName string, workerPoolName string) *http.Request {
	if len(workerPoolName) == 0 {
		log.Fatalln("Please specify worker pool using -workerPoolName option")
	}
	buffer := bytes.NewBufferString(armadaClustersURL.String())
	buffer.WriteString("/")
	buffer.WriteString(clusterName)
	buffer.WriteString("/workerpools/")
	buffer.WriteString(workerPoolName)
	if req, err := http.NewRequest("DELETE", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// GetWorkerPoolRequest creates a request that views the details of a cluster's worker pool
// See https://containers.cloud.ibm.com/swagger-api/#/clusters/GetWorkerPool
func GetWorkerPoolRequest(clusterName string, workerPoolID string) *http.Request {
	if len(workerPoolID) == 0 {
		log.Fatalln("Please specify worker pool using -workerPoolId option")
	}
	buffer := bytes.NewBufferString(armadaClustersURL.String())
	buffer.WriteString("/")
	buffer.WriteString(clusterName)
	buffer.WriteString("/workerpools/")
	buffer.WriteString(workerPoolID)
	if req, err := http.NewRequest("GET", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// ResizeWorkerPoolRequest creates a request that resizes a worker pool.
// It will change the number of worker nodes that an existing worker pool deploys in each zone (datacenter
// by resizing the worker pool.
// See https://containers.cloud.ibm.com/swagger-api/#/clusters/PatchWorkerPool
func ResizeWorkerPoolRequest(body string, clusterName string, workerPoolName string) *http.Request {
	buffer := bytes.NewBufferString(armadaClustersURL.String())
	buffer.WriteString("/")
	buffer.WriteString(clusterName)
	buffer.WriteString("/workerpools/")
	buffer.WriteString(workerPoolName)
	if req, err := http.NewRequest("PATCH", buffer.String(), bytes.NewBufferString(body)); err != nil {
		panic(err)
	} else {
		return req
	}
}

// AddWorkerPoolZoneRequest creates a request that adds a zone to the specified worker pool for a cluster
// See https://containers.cloud.ibm.com/swagger-api/#/clusters/AddWorkerPoolZone
func AddWorkerPoolZoneRequest(body string, clusterName string, workerPoolName string) *http.Request {
	buffer := bytes.NewBufferString(armadaClustersURL.String())
	buffer.WriteString("/")
	buffer.WriteString(clusterName)
	buffer.WriteString("/workerpools/")
	buffer.WriteString(workerPoolName)
	buffer.WriteString("/zones")
	if req, err := http.NewRequest("POST", buffer.String(), bytes.NewBufferString(body)); err != nil {
		panic(err)
	} else {
		return req
	}
}

// RemoveWorkerPoolZoneRequest creates a request that removes a zone from a worker pool
// See https://containers.cloud.ibm.com/swagger-api/#/clusters/RemoveWorkerPoolZone
func RemoveWorkerPoolZoneRequest(clusterName string, workerPoolName string, zoneID string) *http.Request {
	if len(workerPoolName) == 0 {
		log.Fatalln("Please specify worker pool using -workerPoolName option")
	}

	buffer := bytes.NewBufferString(armadaClustersURL.String())
	buffer.WriteString("/")
	buffer.WriteString(clusterName)
	buffer.WriteString("/workerpools/")
	buffer.WriteString(workerPoolName)
	buffer.WriteString("/zones/")
	buffer.WriteString(zoneID)

	if req, err := http.NewRequest("DELETE", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// GetClusterWorkersRequest creates a request that retrieves the list of workers for the specified cluster
// See https://containers.cloud.ibm.com/swagger-api/#/clusters/GetClusterWorkers
func GetClusterWorkersRequest(clusterName string) *http.Request {
	buffer := bytes.NewBufferString(armadaClustersURL.String())
	buffer.WriteString("/")
	buffer.WriteString(clusterName)
	buffer.WriteString("/workers")

	if req, err := http.NewRequest("GET", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// AddClusterWorkersRequest creates a request that adds additional workers to the specified cluster
// See https://containers.cloud.ibm.com/swagger-api/#/clusters/AddClusterWorkers
func AddClusterWorkersRequest(body string, clusterName string) *http.Request {
	buffer := bytes.NewBufferString(armadaClustersURL.String())
	buffer.WriteString("/")
	buffer.WriteString(clusterName)
	buffer.WriteString("/workers")

	if req, err := http.NewRequest("POST", buffer.String(), bytes.NewBufferString(body)); err != nil {
		panic(err)
	} else {
		return req
	}
}

// GetWorkerRequest creates a request that retrieves a worker's details for the specified cluster
// See https://containers.cloud.ibm.com/swagger-api/#/clusters/GetWorkers
func GetWorkerRequest(workerID string) *http.Request {
	if len(workerID) == 0 {
		log.Fatalln("Please specify worker using -workerId option")
	}
	buffer := bytes.NewBufferString(armadaURL.String())
	buffer.WriteString("/workers/")
	buffer.WriteString(workerID)
	if req, err := http.NewRequest("GET", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// DeleteWorkerRequest creates a request that deletes a worker for the specified cluster
// See https://containers.cloud.ibm.com/swagger-api/#/clusters/RemoveClusterWorker
func DeleteWorkerRequest(workerID string) *http.Request {
	if len(workerID) == 0 {
		log.Fatalln("Please specify worker using -workerId option")
	}
	buffer := bytes.NewBufferString(armadaURL.String())
	buffer.WriteString("/workers/")
	buffer.WriteString(workerID)
	if req, err := http.NewRequest("DELETE", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// UpdateWorkerRequest creates a request that reboots or reloads a worker for the specified cluster
// See https://containers.cloud.ibm.com/swagger-api/#/clusters/UpdateClusterWorker
func UpdateWorkerRequest(clusterName, workerID, command string) *http.Request {
	if len(workerID) == 0 {
		log.Fatalln("Please specify worker using -workerId option")
	}
	buffer := bytes.NewBufferString(armadaClustersURL.String())
	buffer.WriteString("/")
	buffer.WriteString(clusterName)
	buffer.WriteString("/workers/")
	buffer.WriteString(workerID)

	body := "{\"action\": \"" + command + "\"}"
	if req, err := http.NewRequest("PUT", buffer.String(), bytes.NewBufferString(body)); err != nil {
		panic(err)
	} else {
		return req
	}
}

// GetDatacentersRequest creates a request that retrieves the list of datacenters
// See https://containers.cloud.ibm.com/swagger-api/#/properties/getDataCenters
// Historical request, superseded by GetZones
func GetDatacentersRequest() *http.Request {
	buffer := bytes.NewBufferString(armadaURL.String())
	buffer.WriteString("/datacenters")

	if req, err := http.NewRequest("GET", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// GetRegionsRequest creates a request that retrieves the list of available zones(datacenters) in a region (Deprecated)
// See https://containers.cloud.ibm.com/swagger-api/#/util/GetRegions
func GetRegionsRequest() *http.Request {
	buffer := bytes.NewBufferString(armadaURL.String())
	buffer.WriteString("/regions")

	if req, err := http.NewRequest("GET", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// GetZonesRequest creates a request that retrieves the list of available zones(datacenters) in a region
// See https://containers.cloud.ibm.com/swagger-api/#/util/GetZones
func GetZonesRequest() *http.Request {
	buffer := bytes.NewBufferString(armadaURL.String())
	buffer.WriteString("/zones")

	if req, err := http.NewRequest("GET", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// GetMachineTypesRequest creates a request that retrieves the list of datacenter machine types
// See https://containers.cloud.ibm.com/swagger-api/#/properties/getMachineTypes
func GetMachineTypesRequest() *http.Request {
	buffer := bytes.NewBufferString(armadaURL.String())
	buffer.WriteString("/datacenters/")
	buffer.WriteString(reqConf.Location.Datacenter)
	buffer.WriteString("/machine-types")

	if req, err := http.NewRequest("GET", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// GetVLANsRequest creates a request that retrieves the list of valid datacenter VLANs
// See https://containers.cloud.ibm.com/swagger-api/#/properties/getDatacenterVLANs
func GetVLANsRequest() *http.Request {
	buffer := bytes.NewBufferString(armadaURL.String())
	buffer.WriteString("/datacenters/")
	buffer.WriteString(reqConf.Location.Datacenter)
	buffer.WriteString("/vlans")

	if req, err := http.NewRequest("GET", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// GetKubeVersionsRequest creates a request that retrieves the list of supported Kubernetes versions
// See https://containers.cloud.ibm.com/swagger-api/#/util/GetKubeVersions (Deprecated)
func GetKubeVersionsRequest() *http.Request {
	buffer := bytes.NewBufferString(armadaURL.String())
	buffer.WriteString("/kube-versions")

	if req, err := http.NewRequest("GET", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// GetVersionsRequest creates a request that retrieves the list of supported Kubernetes versions
// See https://containers.cloud.ibm.com/swagger-api/#/util/GetVersions
func GetVersionsRequest() *http.Request {
	buffer := bytes.NewBufferString(armadaURL.String())
	buffer.WriteString("/versions")

	if req, err := http.NewRequest("GET", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// GetSubnetsRequest creates a request that retrieves the list of availbale portable subnets
// in the user's Bluemix Infrastructure (Softlayer) account
// See https://containers.cloud.ibm.com/swagger-api/#/properties/ListSubnets
func GetSubnetsRequest() *http.Request {
	buffer := bytes.NewBufferString(armadaURL.String())
	buffer.WriteString("/subnets")

	if req, err := http.NewRequest("GET", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// CreateSubnetRequest creates a request that creates a portable subnet under your public or private vlan
// in the user's Bluemix Infrastructure (Softlayer) account. and make it available to a cluster
// See https://containers.cloud.ibm.com/swagger-api/#/properties/ListSubnets
func CreateSubnetRequest(clusterName, vlanID string) *http.Request {
	buffer := bytes.NewBufferString(armadaClustersURL.String())
	buffer.WriteString("/")
	buffer.WriteString(clusterName)
	buffer.WriteString("/vlans/")
	buffer.WriteString(vlanID)

	if req, err := http.NewRequest("POST", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		subnetSize := reqConf.Softlayer.SoftlayerPortableSubnetSize
		if subnetSize == 0 {
			subnetSize = 16
		}
		values := req.URL.Query()
		values.Add("size", strconv.Itoa(subnetSize))
		req.URL.RawQuery = values.Encode()

		return req
	}
}

// GetCredentialsRequest creates a request that sets Bluemix Infrastructure (Softlayer) account credentials.
// See https://containers.cloud.ibm.com/swagger-api/#/accounts/GetUserCredentials
func GetCredentialsRequest() *http.Request {
	buffer := bytes.NewBufferString(armadaURL.String())
	buffer.WriteString("/credentials")
	if req, err := http.NewRequest("GET", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// SetCredentialsRequest creates a request that sets Bluemix Infrastructure (Softlayer) account credentials.
// See https://containers.cloud.ibm.com/swagger-api/#/accounts/storeUserCredentials
func SetCredentialsRequest() *http.Request {
	buffer := bytes.NewBufferString(armadaURL.String())
	buffer.WriteString("/credentials")
	if req, err := http.NewRequest("POST", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		req.Header.Set("X-Auth-Softlayer-Username", config.GetConfigString("armada_softlayer_username", reqConf.Softlayer.SoftlayerUsername))
		req.Header.Set("X-Auth-Softlayer-APIKey", config.GetConfigString("armada_softlayer_api_key", reqConf.Softlayer.SoftlayerAPIKey))

		return req
	}
}

// DeleteCredentialsRequest creates a request that removes Bluemix Infrastructure (Softlayer) account credentials.
// See https://containers.cloud.ibm.com/swagger-api/#/accounts/removeUserCredentials
func DeleteCredentialsRequest() *http.Request {
	buffer := bytes.NewBufferString(armadaURL.String())
	buffer.WriteString("/credentials")
	if req, err := http.NewRequest("DELETE", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// GetVLANSpanningRequest creates a request that retrieves the vlan spanning status for an infrastructure account
// See https://containers.cloud.ibm.com/swagger-api/#/accounts/GetVlanSpanning
func GetVLANSpanningRequest() *http.Request {
	buffer := bytes.NewBufferString(armadaURL.String())
	buffer.WriteString("/subnets/vlan-spanning")

	if req, err := http.NewRequest("GET", buffer.String(), nil); err != nil {
		panic(err)
	} else {
		return req
	}
}

// UpdateClusterRequest updates the version of the Kubernetes cluster master node
// See https://containers.cloud.ibm.com/swagger-api/#/clusters/UpdateCluster
func UpdateClusterRequest(clusterName string, kubeVersion string, action string) *http.Request {
	buffer := bytes.NewBufferString(armadaClustersURL.String())
	buffer.WriteString("/")
	buffer.WriteString(clusterName)

	var body string
	if len(kubeVersion) == 0 {
		body = fmt.Sprintf("{\"action\": \"%s\"}", action)
	} else {
		body = fmt.Sprintf("{\"action\": \"%s\", \"version\": \""+kubeVersion+"\"}", action)
	}
	if req, err := http.NewRequest("PUT", buffer.String(), bytes.NewBufferString(body)); err != nil {
		panic(err)
	} else {
		return req
	}
}

// PerformRequest makes the specified request to armada
func PerformRequest(request Data, singleAttempt bool) Data {
	var req *http.Request
	var localKubeUpdateVersion string

	// We're not a blocking request unless the action is associated with worker creation
	blockingRequest = blockingRequest && request.Action.WorkerCreation()

	totalWorkersStr := strconv.Itoa(request.TotalWorkers)
	switch request.Action {
	case config.ActionCreateCluster:
		// Generate http request from template, substituting the cluster name and worker count
		postData := strings.Replace(postDataSized, "%CLUSTERNAME%", request.ClusterName, 1)
		postData = strings.Replace(postData, "\"%WORKERNUM%\"", totalWorkersStr, 1)

		req = CreateClusterRequest(postData)

	case config.ActionGetClusters:
		req = GetClustersRequest()

	case config.ActionGetCluster:
		req = GetClusterRequest(request.ClusterName)

	case config.ActionGetClusterConfig:
		req = GetClusterConfigRequest(request.ClusterName)

	case config.ActionDeleteCluster:
		req = DeleteClusterRequest(request.ClusterName)

	case config.ActionGetWorkerPools:
		req = GetWorkerPoolsRequest(request.ClusterName)

	case config.ActionCreateWorkerPool:
		postData := strings.Replace(string(workerPoolConfigTemplate), "%DISKENCRYPTION%", strconv.FormatBool(reqConf.Softlayer.SoftlayerDiskEncryption), 1)
		postData = strings.Replace(postData, "%ISOLATION%", reqConf.Softlayer.SoftlayerIsolation, 1)
		postData = strings.Replace(postData, "%MACHINETYPE%", machineType, 1)
		postData = strings.Replace(postData, "%WORKERPOOLNAME%", workerPoolName, 1)
		postData = strings.Replace(postData, "%LABELNAME%", "poolLabel", 1)
		postData = strings.Replace(postData, "%LABEL%", workerPoolName, 1) // use workerpool name as the label
		postData = strings.Replace(postData, "\"%SIZEPERZONE%\"", strconv.FormatInt(int64(request.PoolSize), 10), 1)

		req = CreateWorkerPoolRequest(postData, request.ClusterName)

	case config.ActionRemoveWorkerPool:
		req = RemoveWorkerPoolRequest(request.ClusterName, request.WorkerPoolName)

	case config.ActionGetWorkerPool:
		req = GetWorkerPoolRequest(request.ClusterName, request.WorkerPoolName)

	case config.ActionResizeWorkerPool:
		postData := strings.Replace(string(workerPoolSizeTemplate), "\"%SIZEPERZONE%\"", strconv.FormatInt(int64(request.PoolSize), 10), 1)
		req = ResizeWorkerPoolRequest(postData, request.ClusterName, request.WorkerPoolName)

	case config.ActionAddWorkerPoolZone:
		postData := strings.Replace(string(workerZoneConfigTemplate), "%ZONEID%", request.ZoneID, 1)
		postData = strings.Replace(postData, "%PRIVATEVLAN%", reqConf.Softlayer.SoftlayerPrivateVLAN, 1)
		postData = strings.Replace(postData, "%PUBLICVLAN%", reqConf.Softlayer.SoftlayerPublicVLAN, 1)
		req = AddWorkerPoolZoneRequest(postData, request.ClusterName, request.WorkerPoolName)

	case config.ActionRemoveWorkerPoolZone:
		req = RemoveWorkerPoolZoneRequest(request.ClusterName, request.WorkerPoolName, request.ZoneID)

	case config.ActionGetClusterWorkers:
		req = GetClusterWorkersRequest(request.ClusterName)

	case config.ActionAddClusterWorkers:
		postData := strings.Replace(string(addClusterWorkersTemplate), "\"%WORKERNUM%\"", totalWorkersStr, 1)
		postData = strings.Replace(postData, "%MACHINETYPE%", machineType, 1)
		postData = strings.Replace(postData, "%DISKENCRYPTION%", strconv.FormatBool(reqConf.Softlayer.SoftlayerDiskEncryption), 1)
		postData = strings.Replace(postData, "%PRIVATEVLAN%", reqConf.Softlayer.SoftlayerPrivateVLAN, 1)
		postData = strings.Replace(postData, "%PUBLICVLAN%", reqConf.Softlayer.SoftlayerPublicVLAN, 1)

		req = AddClusterWorkersRequest(postData, request.ClusterName)

	case config.ActionGetWorker:
		req = GetWorkerRequest(request.WorkerID)

	case config.ActionDeleteWorker:
		req = DeleteWorkerRequest(request.WorkerID)

	case config.ActionRebootWorker:
		// For now we'll support a soft reboot only.
		// If we want to add support for hard reboot in the future, action would be "power_cycle"
		req = UpdateWorkerRequest(request.ClusterName, request.WorkerID, "os_reboot")

	case config.ActionReloadWorker:
		req = UpdateWorkerRequest(request.ClusterName, request.WorkerID, "reload")

	case config.ActionGetDatacenters:
		req = GetDatacentersRequest()

	case config.ActionGetRegions:
		req = GetRegionsRequest()

	case config.ActionGetZones:
		req = GetZonesRequest()

	case config.ActionGetMachineTypes:
		req = GetMachineTypesRequest()

	case config.ActionGetVLANs:
		req = GetVLANsRequest()

	case config.ActionGetKubeVersions:
		req = GetKubeVersionsRequest()

	case config.ActionGetSubnets:
		req = GetSubnetsRequest()

	case config.ActionCreateSubnet:
		if reqConf.Request.PrivateVLAN {
			req = CreateSubnetRequest(request.ClusterName, reqConf.Softlayer.SoftlayerPrivateVLAN)
		}
		if reqConf.Request.PublicVLAN {
			req = CreateSubnetRequest(request.ClusterName, reqConf.Softlayer.SoftlayerPublicVLAN)
		}

	case config.ActionGetCredentials:
		req = GetCredentialsRequest()

	case config.ActionSetCredentials:
		req = SetCredentialsRequest()

	case config.ActionDeleteCredentials:
		req = DeleteCredentialsRequest()

	case config.ActionGetVLANSpanning:
		req = GetVLANSpanningRequest()

	case config.ActionUpdateCluster:
		localKubeUpdateVersion = kubeVersion
		if len(request.KubeUpdateVersion) > 0 {
			localKubeUpdateVersion = request.KubeUpdateVersion
		}
		req = UpdateClusterRequest(request.ClusterName, localKubeUpdateVersion, "update")

	case config.ActionApplyPullSecret:
		req = UpdateClusterRequest(request.ClusterName, "", "enablePullSecrets")

	case config.ActionGetVersions:
		req = GetVersionsRequest()

	default:
		fmt.Println("Action not defined: ", request.Action)
	}

	// Set headers that are mandatory for all requests
	setCommonRequestHeaders(req)

	client := &http.Client{}

	var actionStr = request.Action.String()
	if request.Action.HasCluster() {
		actionStr = strings.Join([]string{actionStr, request.ClusterName}, " - ")
	}
	apiRequestTime := time.Now()
	if verbose {
		fmt.Printf("%s\tAPI request: %s : \"%s\"\n", apiRequestTime.Format(time.StampMilli), req.URL.String(), actionStr)
	}
	if debug {
		if req.Body != nil {
			fmt.Println(req.Body)
		}
	}
	if monitor {
		fmt.Printf("%s: monitor %s %+v requested\n", time.Now().Format(time.StampMilli), request.ClusterName, request.Action)
	}

	var resp *http.Response
	var err error
	if singleAttempt {
		// Only retry if there is an authentication error, and only once
		resp, err = client.Do(req)
		if err != nil {
			fmt.Println(err)
			request.ActionFailed = true
			return request
		}

		request.Status = resp.Status
		request.StatusCode = resp.StatusCode

		// During long tests token may become stale, give 1 try at resolving issue
		//Add temporary check for registry token issue giving occasional StatusInternalServerError errors.
		if resp.StatusCode == http.StatusUnauthorized || (resp.StatusCode == http.StatusInternalServerError && request.Action == config.ActionCreateCluster) {
			fmt.Printf("Request failed with %v error code. Retrying request. Cluster: %s\n", resp.StatusCode, request.ClusterName)
			resp.Body.Close()
			Authenticate()
			setCommonRequestHeaders(req)

			// Sleep before retry
			time.Sleep(time.Second * 10)

			resp, err = client.Do(req)

			if err != nil {
				if resp != nil {
					fmt.Printf("Second request failed with %v error code, Status: %v. Cluster: %s\n", resp.StatusCode, resp.Status, request.ClusterName)
				} else {
					fmt.Printf("Second request failed with %v error, Cluster: %s\n", err, request.ClusterName)
				}
				fmt.Println(err)
				request.ActionFailed = true
				return request
			}
			fmt.Printf("Second request succeeded with %v response code. Cluster: %s\n", resp.StatusCode, request.ClusterName)
			request.Status = resp.Status
			request.StatusCode = resp.StatusCode
			defer resp.Body.Close()

		} else {
			defer resp.Body.Close()
		}
	} else {

		var sleep = time.Millisecond
		for {
			resp, err = client.Do(req)
			if err != nil {
				fmt.Printf("%s\tAPI request: %s : \"%s\"\n", apiRequestTime.Format(time.StampMilli), req.URL.String(), request.Action)
				fmt.Println(err)
				//panic(err)
			} else {
				request.Status = resp.Status
				request.StatusCode = resp.StatusCode

				if resp.StatusCode/100 == 2 || resp.StatusCode == http.StatusNotFound {
					defer resp.Body.Close()
					break
				} else if resp.StatusCode == http.StatusUnauthorized {
					fmt.Printf("Retrying request after re-authenticating after receiving StatusUnauthorized: \"%s\" %s\n", req.URL.String(), request.Action)
				} else {
					fmt.Printf("%s : Retrying request after response of %s : \"%s\" %s\n", apiRequestTime.Format(time.StampMilli), resp.Status, req.URL.String(), request.Action)
				}
				resp.Body.Close()
			}
			if err != nil || resp.StatusCode == http.StatusUnauthorized {
				// During long tests token may become stale, give 1 try at resolving issue
				Authenticate()
				setCommonRequestHeaders(req)
			}
			time.Sleep(sleep)
			if sleep < time.Second*10 {
				sleep = sleep * 10
			}
			apiRequestTime = time.Now()
		}
	}

	var body []byte
	if resp.Body != nil {
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
	}

	apiResponseTime := time.Now()
	request.ResponseTime = apiResponseTime.Sub(apiRequestTime)

	successResp := resp.StatusCode/100 == 2

	if verbose {
		fmt.Printf("%s\tAPI response: %s\n", apiResponseTime.Format(time.StampMilli), resp.Status)
	}
	if debug {
		fmt.Println(resp.Header)
	}
	if resp.StatusCode != http.StatusNoContent {
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			var prettyJSON bytes.Buffer
			if err = json.Indent(&prettyJSON, body, "", "\t"); err != nil {
				panic(err)
			}
			if verbose || (!successResp && !(request.Action == config.ActionGetCluster && resp.StatusCode == http.StatusNotFound)) {
				fmt.Println(string(prettyJSON.Bytes()))
			}
		} else if !strings.Contains(contentType, "application/zip") {
			// Don't print out zip contents
			fmt.Println("Dumping unexpected content:", request.ClusterName, resp.Status)
			fmt.Println(string(body))
			if strings.Contains(string(body), "cloudflare") {
				fmt.Println("ERROR: cloudflare returned error")
				request.Failure = FailureCloudflare
			} else if strings.Contains(string(body), "A backend issue occurred") {
				fmt.Println("ERROR: A backend issue occurred")
				request.Failure = FailureBackendIssue
			} else if strings.Contains(string(body), "Could not connect to a backend service") {
				fmt.Println("ERROR: Could not connect to a backend service")
				request.Failure = FailureBackendIssue
			}
		}
	}

	if resp.StatusCode != http.StatusNoContent {
		request.ContentType = resp.Header.Get("Content-Type")
		request.Body = body
	}

	var pollForWorkersEligible = false
	var pollForMastersEligible = false
	// If a successful return code
	if successResp {
		// Post processing of response
		switch request.Action {
		case config.ActionCreateCluster:
			if request.TotalWorkers > 0 {
				pollForWorkersEligible = true
			}
			pollForMastersEligible = true
			// Get ClusterID from the API http response
			var dat map[string]interface{}
			if err = json.Unmarshal(body, &dat); err != nil {
				panic(err)
			}
			request.ClusterID = dat["id"].(string)

			// When creating a Cruiser (cluster) we need to inform Armada cluster that we're ready to provision workers.
			// This is normally done by Armada deploy once the cruiser master is ready.
			if mockDeploy != nil || mockBootstrap != nil {
				if mockDeploy != nil {
					mockDeploy.DeployCruiserMaster(request.ClusterID)
				}
				if mockBootstrap != nil {
					mockBootstrap.PerformBootstrap(request.Action, reqConf.Bluemix.AccountID, request.ClusterID, machineType == config.FreeAccountStr)
				}
			}

		case config.ActionDeleteCluster:
			pollForMastersEligible = true
		case config.ActionAddClusterWorkers:
			pollForWorkersEligible = true
			if mockBootstrap != nil {
				mockBootstrap.PerformBootstrap(request.Action, reqConf.Bluemix.AccountID, request.ClusterName, machineType == config.FreeAccountStr)
			}
		case config.ActionGetClusterWorkers:
			pollForWorkersEligible = true
		case config.ActionGetClusterConfig:
			// Create zip file containing cluster config
			clusterConfigPrefix := "kubeConfig"
			if reqConf.Request.AdminConfig {
				clusterConfigPrefix = clusterConfigPrefix + "Admin"
			}
			clusterConfigFilename := clusterConfigPrefix + "-" + request.ClusterName + ".zip"
			err = ioutil.WriteFile(clusterConfigFilename, body, 0644)
			if err != nil {
				panic(err)
			}
			fmt.Println("Configuration for \"" + request.ClusterName + "\" written to " + clusterConfigFilename)

		case config.ActionUpdateCluster:
			pollForMastersEligible = true
		default:
			// Nothing to do.
		}

		// Synchronous request? If so, we'll block until the master is ready to rock 'n' roll
		if pollForMastersEligible && reqConf.Request.MasterPollInterval.Duration > 0 {
			var clusterComplete bool

			// Mark this request as blocking. Used for generating sensible metrics.
			blockingRequest = true

			request.Metrics.ClusterName = request.ClusterName

			for !clusterComplete {
				clusterComplete = true

				statusReq := GetClusterRequest(request.ClusterName)

				setCommonRequestHeaders(statusReq)

				statusResp, err := client.Do(statusReq)
				if err != nil {
					fmt.Println(err)
					time.Sleep(reqConf.Request.MasterPollInterval.Duration)
					clusterComplete = false
					continue
				}

				// For long running tests our authentication tokens can expire. (Think they are valid for one hour after issue)
				// We should really add some code to get the expiry time, and then use the refresh token
				// to grant a new authentication token. But for now, lets just hack it...
				if statusResp.StatusCode == http.StatusUnauthorized {
					// Token has probably expired. Let's try once more with new tokens.
					statusResp.Body.Close()
					Authenticate()
					setCommonRequestHeaders(statusReq)
					statusResp, err = client.Do(statusReq)
					if err != nil {
						panic(err)
					}
				}

				// API sometimes kicks back a request. Mostly due to availability issues with resource group manager
				// Causes master[] to be filled with error message, not the status of the cluster
				if statusResp.StatusCode == http.StatusBadRequest {
					clusterComplete = false
					continue
				}

				if request.Action == config.ActionDeleteCluster && statusResp.StatusCode == http.StatusNotFound {
					request.ActionTime = time.Since(apiRequestTime)
				} else {

					body, err := ioutil.ReadAll(statusResp.Body)
					if err != nil {
						panic(err)
					}
					statusResp.Body.Close()
					var cgResp interface{}
					err = json.Unmarshal(body, &cgResp)
					if err != nil {
						// Some hokey response that we weren't expecting, let's ignore and carry on polling
						if debug && err != nil {
							fmt.Println(err.Error())
							fmt.Println(string(body))
						}
						time.Sleep(reqConf.Request.WorkerPollInterval.Duration)
						clusterComplete = false
						continue
					}

					master := cgResp.(map[string]interface{})

					if master["masterStatus"] != nil {
						if request.Action == config.ActionUpdateCluster &&
							master["masterKubeVersion"] != nil && !strings.HasPrefix(master["masterKubeVersion"].(string), strings.Trim(localKubeUpdateVersion, "_openshift")) {
							if strings.HasPrefix(master["masterStatus"].(string), "Version update failed.") || strings.HasPrefix(master["masterStatus"].(string), "Version update cancelled.") {
								fmt.Println("Version update failed", request.ClusterName, master["masterStatus"].(string))
							} else {
								clusterComplete = false
							}
						} else if request.Action == config.ActionUpdateCluster &&
							strings.HasPrefix(master["masterStatus"].(string), "The master is already up to date for the specified version") {
							request.KubeUpdateVersion = master["masterStatus"].(string)
							continue
						} else {

							switch master["masterStatus"] {
							case "Ready":
								// protect against first check not catching delete request
								if request.Action == config.ActionDeleteCluster {
									clusterComplete = false
									// Cluster masterStatus can be "Ready" before operation starts. Even for creates
								} else if time.Since(apiRequestTime) > time.Duration(1)*time.Minute {
									request.ActionTime = time.Since(apiRequestTime)
								}

							case "Deploy requested.", "Deploy in progress.",
								"Version update requested.", "Version update in progress.",
								"Delete requested.", "Delete in progress.",
								"VPN server configuration update in progress.",
								"VPN server configuration update requested.",
								"CAE011: The domain name service for the network load balancer was not created in the allotted time (2 hours). For troubleshooting steps, see the docs: http://ibm.biz/rhoks_ts_vpn_subnet",
								"":

								clusterComplete = false

							default:
								// If update failed then status will reflect that when delete is done.
								if request.Action == config.ActionDeleteCluster &&
									(strings.Contains(master["masterStatus"].(string), "Version update failed.") || strings.Contains(master["masterStatus"].(string), "Version update cancelled.")) {
									clusterComplete = false
								} else {

									fmt.Println("Unexpected masterStatus: ", request.ClusterName, " = ", master["masterStatus"])
								}
							}
						}
					} else {
						clusterComplete = false
						fmt.Println("masterStatus is nil")
						fmt.Println("statusResp.StatusCode: ", statusResp.StatusCode)
						fmt.Println(master)
					}
				}
				if !clusterComplete {
					time.Sleep(reqConf.Request.MasterPollInterval.Duration)
				}
			}

			if request.ActionTime == 0 {
				request.ActionFailed = true
			}
		}
	} else {
		request.ActionFailed = true
	}

	// Synchronous request? If so, we'll block until the cluster is ready to rock 'n' roll
	if pollForWorkersEligible && reqConf.Request.WorkerPollInterval.Duration > 0 && !request.ActionFailed {
		var clusterComplete bool
		var prevTime time.Time

		// Mark this request as blocking. Used for generating sensible metrics.
		blockingRequest = true

		request.Metrics.ClusterName = request.ClusterName
		request.Metrics.WorkerCreationTimes = make(map[string]float64)
		request.Metrics.ClusterWorkerStates = make(metrics.ClusterWorkerStateMetrics)

		for !clusterComplete {
			clusterComplete = true

			statusReq := GetClusterWorkersRequest(request.ClusterName)
			setCommonRequestHeaders(statusReq)

			pollRequestTime := time.Now()
			statusResp, err := client.Do(statusReq)
			if err != nil {
				// Error occurred, let's ignore and carry on polling
				fmt.Printf("Error occurred getting workers, will continue polling: %s", err.Error())

				time.Sleep(reqConf.Request.WorkerPollInterval.Duration)
				clusterComplete = false
				continue
			}

			// For long running tests our authentication tokens can expire. (Think they are valid for one hour after issue)
			// We should really add some code to get the expiry time, and then use the refresh token
			// to grant a new authentication token. But for now, lets just hack it...
			if statusResp.StatusCode == http.StatusUnauthorized {
				// Token has probably expired. Let's try once more with new tokens.
				statusResp.Body.Close()
				Authenticate()
				setCommonRequestHeaders(statusReq)
				pollRequestTime = time.Now()
				statusResp, err = client.Do(statusReq)
				if err != nil {
					// Error occurred, let's ignore and carry on polling
					fmt.Printf("Error occurred getting workers, will continue polling: %s", err.Error())

					time.Sleep(reqConf.Request.WorkerPollInterval.Duration)
					clusterComplete = false
					continue
				}
			}

			body, err := ioutil.ReadAll(statusResp.Body)
			if err != nil {
				panic(err)
			}
			statusResp.Body.Close()
			var cgResp []interface{}
			err = json.Unmarshal(body, &cgResp)
			if err != nil || len(cgResp) == 0 {
				// Some hokey response that we weren't expecting, let's ignore and carry on polling
				if debug && err != nil {
					fmt.Println(err.Error())
					fmt.Println(string(body))
				}
				time.Sleep(reqConf.Request.WorkerPollInterval.Duration)
				clusterComplete = false
				continue
			}

			var headerWritten = false

			var workersCompletedCount = 0
			var workerStateCounts = make(map[string]int)
			for _, worker := range cgResp {
				reportChange := false

				workerDetails := worker.(map[string]interface{})

				workerID := workerDetails["id"].(string)
				workerState := workerDetails["state"].(string)
				workerStatus := workerDetails["status"].(string)

				workerStateCounts[workerState]++

				// Dirty, nasty, horrible code alert which I'm disowning
				// We want to include "waiting for master to be deployed" in our state metrics.
				// Alas, this information is a worker status field (rather than a worker state) so we have to check for a specific string
				// (or send all status transitions too, which given the fairly large number of them, we don't really want to do)
				const wfmd = "waiting_for_master_deployment"
				if workerStatus == "Waiting for master to be deployed" {
					workerState = wfmd
				}

				// First time we've encountered this worker ?, if so, initialize it
				if ws, ok := request.Metrics.ClusterWorkerStates[workerID]; !ok {
					reportChange = true
					request.Metrics.ClusterWorkerStates[workerID] = metrics.WorkerStates{
						CurState:  workerState,
						CurStatus: workerStatus,
						Metrics:   make(map[string]time.Duration)}
					ws = request.Metrics.ClusterWorkerStates[workerID]
					ws.Metrics[workerState] = 0
					request.Metrics.ClusterWorkerStates[workerID] = ws
				} else {
					// Worker moving to a new state ?
					if ws.CurState != workerState {
						reportChange = true

						// Record the duration of the state we've just completed
						ws.Metrics[ws.CurState] += pollRequestTime.Sub(prevTime)

						// Now update the state and status fields
						ws.CurState = workerState
						ws.CurStatus = workerStatus
					} else {
						// Still in same state, update its total duration
						ws.Metrics[ws.CurState] += pollRequestTime.Sub(prevTime)

						// Check the status too (for reporting updates to the user purposes only)
						if ws.CurStatus != workerStatus {
							ws.CurStatus = workerStatus
							reportChange = true
						}

					}
					request.Metrics.ClusterWorkerStates[workerID] = ws
				}

				// Keep a record of total creation time for each worker, together with a running record of
				// how many workers have been successfully created (can use to plot successful worker creations over time)
				workerCompleted := strings.EqualFold(workerState, "normal") || strings.EqualFold(workerState, "deleted")
				if workerCompleted {
					workersCompletedCount++

					if request.Metrics.WorkerCreationTimes[workerID] == 0 {
						request.Metrics.WorkerCreationTimes[workerID] = time.Since(apiRequestTime).Seconds()
					}
				}

				// If this workers state has changed since last polling interval, then display update
				// (For display purposes, ensure we group all worker state changes under a single timestamp update)
				if reportChange {
					if !headerWritten {
						fmt.Println(pollRequestTime.Format(time.Stamp))
						fmt.Printf("\t%s\n", request.ClusterName)
						headerWritten = true
					}
					fmt.Printf("\t\t%s, %s, \"%s\"\n", workerID, workerState, workerStatus)
				}
				clusterComplete = clusterComplete && workerCompleted
			}

			// If we've had a worker state/status update, then update the number of workers currently in each state
			if headerWritten {
				for k, v := range workerStateCounts {
					fmt.Printf("\t%s : %d/%d\n", k, v, len(cgResp))
				}
				fmt.Println()
			}

			prevTime = pollRequestTime

			request.Metrics.Workers = append(
				request.Metrics.Workers,
				metrics.WorkerMetrics{
					MetricTime:     time.Now().Unix(),
					Duration:       time.Since(apiRequestTime),
					WorkersCreated: workersCompletedCount,
				})
			if !clusterComplete {
				time.Sleep(reqConf.Request.WorkerPollInterval.Duration)
			}
		}

		if debug {
			fmt.Println()
			prevWorkersCreated := 0
			for _, m := range request.Metrics.Workers {
				if m.WorkersCreated != prevWorkersCreated {
					log.Printf("Cluster: %s, Duration: %ds, Workers Created: %d\n", request.ClusterName, int64(m.Duration.Seconds()), m.WorkersCreated)
					prevWorkersCreated = m.WorkersCreated
				}
			}
			fmt.Println()
		}
		request.ActionTime = time.Since(apiRequestTime)
	}
	return request
}
