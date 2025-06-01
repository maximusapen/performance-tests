/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2019, 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package resources

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/urfave/cli"
	v2 "github.ibm.com/alchemy-containers/armada-api-model/json/v2"
	"github.ibm.com/alchemy-containers/armada-api-model/json/v2/classic"
	"github.ibm.com/alchemy-containers/armada-api-model/json/v2/requests"
	"github.ibm.com/alchemy-containers/armada-api-model/json/v2/responses"
	"github.ibm.com/alchemy-containers/armada-api-model/json/v2/vpc"
	nlbDNSModel "github.ibm.com/alchemy-containers/armada-dns-model/v2/dns"
	ingressModel "github.ibm.com/alchemy-containers/armada-dns-model/v2/ingress"
	ingressVPC "github.ibm.com/alchemy-containers/armada-dns-model/v2/vpc"
	obmodel "github.ibm.com/alchemy-containers/armada-kedge-model/model"

	"github.ibm.com/alchemy-containers/armada-model/model/ingress"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
)

// V2Endpoint an armada endpoint with provider-agnostic operations
type V2Endpoint struct {
	*ArmadaEndpoint
}

// GetArmadaV2Endpoint an armada endpoint with /v2 in the path
func GetArmadaV2Endpoint(c *cli.Context, opts ...EndpointOpt) *V2Endpoint {
	e := GetArmadaEndpoint(c, opts...)
	e.APIVersion = "v2"
	return &V2Endpoint{e}
}

func (endpoint *V2Endpoint) getVPCClusters() ([]vpc.GetClustersResponse, error) {
	var successV []vpc.GetClustersResponse
	_, err := endpoint.makeRequest(http.MethodGet, "/vpc/getClusters", nil, &successV)
	return successV, err
}

func (endpoint *V2Endpoint) getSatelliteClusters() ([]vpc.GetClustersResponse, error) {
	var successV []vpc.GetClustersResponse
	_, err := endpoint.makeRequest(http.MethodGet, "/satellite/getClusters", nil, &successV)
	return successV, err
}

// GetAllClusters ...
func (endpoint *V2Endpoint) GetAllClusters() (result []Cluster, rawMessage json.RawMessage, err error) {
	var successVV []vpc.GetClustersResponse
	var successSV []vpc.GetClustersResponse

	successVV, _ = endpoint.getVPCClusters()
	successSV, _ = endpoint.getSatelliteClusters()
	successV := append(successVV, successSV...)

	for _, vpcCluster := range successV {
		result = append(
			result,
			NewClusterFromCommon(
				v2.ClusterCommon{
					ID:                vpcCluster.ID,
					Name:              vpcCluster.Name,
					State:             vpcCluster.State,
					CreatedDate:       vpcCluster.CreatedDate,
					WorkerCount:       vpcCluster.WorkerCount,
					Location:          vpcCluster.Location,
					EOS:               vpcCluster.EOS,
					MasterVersion:     vpcCluster.MasterVersion,
					TargetVersion:     vpcCluster.TargetVersion,
					ResourceGroupName: vpcCluster.ResourceGroupName,
					ProviderID:        vpcCluster.ProviderID,
				},
			),
		)
	}

	jsonBytes, marshalErr := json.MarshalIndent(successV, "", "    ")
	if marshalErr != nil {
		fmt.Printf("Unable to unmarshal JSON response: %s\n", marshalErr.Error())
		err = marshalErr
	}
	rawMessage = json.RawMessage(jsonBytes)

	return
}

// GetCluster retrieves a specified cluster
func (endpoint *V2Endpoint) GetCluster(cluster string) (rawJSON json.RawMessage, clusterDetails Cluster, err error) {
	path := fmt.Sprintf("/getCluster?cluster=%s", cluster)

	buf := bytes.NewBuffer(nil)
	_, err = endpoint.makeRequest(http.MethodGet, path, nil, buf)
	if err != nil {
		return
	}
	rawJSON = buf.Bytes()

	var apiV2Response map[string]interface{}
	err = json.Unmarshal(rawJSON, &apiV2Response)
	if err != nil {
		clusterDetails = nil
		err = fmt.Errorf("Error unmarshalling getCluster resposne. Error: %w", err)
	} else {
		provider := apiV2Response["provider"].(string)

		if IsClassicProvider(provider) {
			classicClusterResponse := classic.GetClusterResponse{}
			err = json.Unmarshal(rawJSON, &classicClusterResponse)
			clusterDetails = NewClusterFromCommons(classicClusterResponse.ClusterCommon, classicClusterResponse.SingleClusterCommon)
		} else if IsVPCProvider(provider) {
			vpcClusterResponse := vpc.GetClusterResponse{}
			err = json.Unmarshal(rawJSON, &vpcClusterResponse)
			clusterDetails = NewClusterFromCommons(vpcClusterResponse.ClusterCommon, vpcClusterResponse.SingleClusterCommon)
		} else if IsSatelliteProvider(provider) {
			satelliteClusterResponse := vpc.GetClusterResponse{}
			err = json.Unmarshal(rawJSON, &satelliteClusterResponse)
			clusterDetails = NewClusterFromCommons(satelliteClusterResponse.ClusterCommon, satelliteClusterResponse.SingleClusterCommon)
		} else {
			clusterDetails = nil
			err = errors.New("can't determine cluster provider type. Use --json flag")
		}
	}

	return rawJSON, clusterDetails, err
}

// GetWorkers lists workers for a cluster.
// It will return v2 properties that are common between classic and vpc clusters.
// This method will require updating to determine the cluster type and then use the armada-api-model specific response (rather than WorkerCommon)
// if some of the specific properties are required.
func (endpoint *V2Endpoint) GetWorkers(cluster string) (rawJSON json.RawMessage, v2CommonResponse []v2.WorkerCommon, err error) {
	path := fmt.Sprintf("/getWorkers?cluster=%s", cluster)

	buf := bytes.NewBuffer(nil)
	_, err = endpoint.makeRequest(http.MethodGet, path, nil, buf)
	if err != nil {
		return
	}
	rawJSON = buf.Bytes()

	err = json.Unmarshal(rawJSON, &v2CommonResponse)

	return
}

// GetWorkerPools lists worker pools for a cluster.
// It will return v2 properties that are common between classic and vpc clusters.
// This method will require updating to determine the cluster type and then use the armada-api-model specific response (rather than WorkerPoolCommon)
// if some of the specific properties are required.
func (endpoint *V2Endpoint) GetWorkerPools(cluster string) (rawJSON json.RawMessage, v2CommonResponse []v2.WorkerPoolCommon, err error) {
	path := fmt.Sprintf("/getWorkerPools?cluster=%s", cluster)

	buf := bytes.NewBuffer(nil)
	_, err = endpoint.makeRequest(http.MethodGet, path, nil, buf)
	if err != nil {
		return
	}
	rawJSON = buf.Bytes()
	err = json.Unmarshal(rawJSON, &v2CommonResponse)

	return
}

// CreateWorkerPool makes the API call to create a VPC workerpool for given cluster
func (endpoint *V2Endpoint) CreateWorkerPool(config requests.VPCCreateWorkerPool) error {
	_, err := endpoint.makeRequest(http.MethodPost, "/vpc/createWorkerPool", config, nil)
	return err
}

// CreateVPCCluster makes the API call to create a VPC cluster
func (endpoint *ArmadaEndpoint) CreateVPCCluster(config requests.VPCCreateCluster) error {
	_, err := endpoint.makeRequest(http.MethodPost, "/vpc/createCluster", config, nil)
	return err
}

// GetVPCs get VPC list
func (endpoint *V2Endpoint) GetVPCs(provider string) (vpc.VirtualPrivateClouds, error) {
	var vpcs vpc.VirtualPrivateClouds
	_, err := endpoint.makeRequest(http.MethodGet, fmt.Sprintf("/vpc/getVPCs?provider=%s", provider), nil, &vpcs)
	return vpcs, err
}

// GetWorker gets details of the specified worker of a cluster
// This method will require updating to determine the cluster type and then use the armada-api-model specific response (rather than WorkerCommon)
// if some of the specific properties are required.
func (endpoint *V2Endpoint) GetWorker(cluster, worker string) (rawJSON json.RawMessage, v2CommonResponse v2.WorkerCommon, err error) {
	path := fmt.Sprintf("/getWorker?cluster=%s&worker=%s", cluster, worker)
	buf := bytes.NewBuffer(nil)
	_, err = endpoint.makeRequest(http.MethodGet, path, nil, buf)
	if err != nil {
		return
	}
	rawJSON = buf.Bytes()

	err = json.Unmarshal(rawJSON, &v2CommonResponse)
	return
}

// GetWorkerPool gets details of the specified worker pool of a cluster
func (endpoint *V2Endpoint) GetWorkerPool(cluster, workerPool string) (rawJSON json.RawMessage, v2Response vpc.GetWorkerPoolResponse, err error) {
	path := fmt.Sprintf("/getWorkerPool?cluster=%s&workerpool=%s", cluster, workerPool)
	buf := bytes.NewBuffer(nil)
	_, err = endpoint.makeRequest(http.MethodGet, path, nil, buf)
	if err != nil {
		return
	}
	rawJSON = buf.Bytes()

	err = json.Unmarshal(rawJSON, &v2Response)
	return
}

// AddZone adds a VPC zone to a workerpool
func (endpoint *V2Endpoint) AddZone(zone vpc.CreateWorkerpoolZoneRequest) (err error) {
	_, err = endpoint.makeRequest(http.MethodPost, "/vpc/createWorkerPoolZone", zone, nil)
	return err
}

// AddSatelliteZone adds a satellite zone to a workerpool
func (endpoint *V2Endpoint) AddSatelliteZone(zone requests.SatelliteWorkerPoolZoneAdd) (err error) {
	_, err = endpoint.makeRequest(http.MethodPost, "/satellite/createWorkerPoolZone", zone, nil)
	return err
}

// ReplaceWorker performs a worker replace on the given worker ID
func (endpoint *V2Endpoint) ReplaceWorker(cluster string, workerID string, update bool) error {
	req := vpc.ReplaceWorkerRequest{
		Cluster:  cluster,
		WorkerID: workerID,
		Update:   update,
	}
	_, err := endpoint.makeRequest(http.MethodPost, "/replaceWorker", req, nil)
	return err
}

// IBM Cloud Satellite

// CreateSatelliteCluster creates a Satellite cruiser
func (endpoint *V2Endpoint) CreateSatelliteCluster(config requests.MultishiftCreateCluster) (string, error) {
	var resp responses.MultishiftCreateCluster
	_, err := endpoint.makeRequest(http.MethodPost, "/satellite/createCluster", config, &resp)
	return resp.ID, err
}

// CreateSatelliteWorkerPool makes the API call to create a Satelliute workerpool for given cluster
func (endpoint *V2Endpoint) CreateSatelliteWorkerPool(config requests.SatelliteCreateWorkerPool) error {
	_, err := endpoint.makeRequest(http.MethodPost, "/satellite/createWorkerPool", config, nil)
	return err
}

// CreateLocation makes the API call to create a Satellite location
func (endpoint *V2Endpoint) CreateLocation(config requests.MultishiftCreateController) (string, error) {
	var resp responses.MultishiftCreateController
	_, err := endpoint.makeRequest(http.MethodPost, "/satellite/createController", config, &resp)
	return resp.ID, err
}

// RemoveLocation makes the API call to remove a Satellite location
func (endpoint *V2Endpoint) RemoveLocation(locationID string) error {
	_, err := endpoint.makeRequest(http.MethodPost, "/satellite/removeController", requests.MultishiftCreateClusterFragment{Controller: locationID}, nil)
	return err
}

// GetLocation returns details of the specified IBM Cloud Satellite location
func (endpoint *V2Endpoint) GetLocation(location string) (rawJSON json.RawMessage, v2SatResponse responses.MultishiftGetController, notFound bool, err error) {
	var buf bytes.Buffer
	var hr *http.Response
	path := fmt.Sprintf("/satellite/getController?%s=%s", requests.ControllerQueryParamKey, location)
	hr, err = endpoint.makeRequest(http.MethodGet, path, nil, &buf)
	if err != nil {
		if hr != nil {
			notFound = (hr.StatusCode == http.StatusNotFound)
		}
		return
	}
	rawJSON = buf.Bytes()

	err = json.Unmarshal(rawJSON, &v2SatResponse)

	return
}

// GetLocations lists the IBM Cloud Satellite locations that a user has access to
func (endpoint *V2Endpoint) GetLocations() (rawJSON json.RawMessage, v2SatResponse []responses.MultishiftController, err error) {
	var buf bytes.Buffer
	_, err = endpoint.makeRequest(http.MethodGet, "/satellite/getControllers", nil, &buf)
	if err != nil {
		return
	}
	rawJSON = buf.Bytes()

	err = json.Unmarshal(rawJSON, &v2SatResponse)

	return
}

// GetLocationSubdomains will list the nlb dns subdomains corresponding to a Satellite location
func (endpoint *V2Endpoint) GetLocationSubdomains(location string) (rawJSON json.RawMessage, v2SatResponse []nlbDNSModel.NlbConfig, err error) {
	var buf bytes.Buffer
	path := fmt.Sprintf("/nlb-dns/getSatLocationSubdomains?controller=%s", location)
	_, err = endpoint.makeRequest(http.MethodGet, path, nil, &buf)
	if err != nil {
		return
	}
	rawJSON = buf.Bytes()

	err = json.Unmarshal(rawJSON, &v2SatResponse)

	return
}

// GetLocationSubdomain will return the details of the given nlb dns corresponding to a Satellite location
func (endpoint *V2Endpoint) GetLocationSubdomain(location string, nlbSubdomain string) (rawJSON json.RawMessage, v2SatResponse nlbDNSModel.NlbConfig, err error) {
	var buf bytes.Buffer
	path := fmt.Sprintf("/nlb-dns/getSatLocationSubdomain?controller=%s&subdomain=%s", location, nlbSubdomain)
	_, err = endpoint.makeRequest(http.MethodGet, path, nil, &buf)
	if err != nil {
		return
	}
	rawJSON = buf.Bytes()

	err = json.Unmarshal(rawJSON, &v2SatResponse)

	return
}

// RegisterLocationSubdomains will register the provided IPs with the domains for the location
func (endpoint *V2Endpoint) RegisterLocationSubdomains(controller string, ips []string) (nlbDNSModel.MSCRegisterResp, error) {
	var successV nlbDNSModel.MSCRegisterResp
	var reqBody = nlbDNSModel.MSCRegistration{
		Controller: controller,
		IPs:        ips,
	}
	_, err := endpoint.makeRequest(http.MethodPost, "/nlb-dns/registerMSCDomains", reqBody, &successV)
	return successV, err
}

// CreateHostScript creates a script to bootstrap a host
func (endpoint *V2Endpoint) CreateHostScript(config requests.MultishiftCreateScript) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	_, err := endpoint.makeRequest(http.MethodPost, "/satellite/hostqueue/createRegistrationScript", config, &buf)
	return &buf, err
}

// CreateHostAssignment creates an assignment as requested by the user
func (endpoint *V2Endpoint) CreateHostAssignment(config requests.MultishiftCreateAssignment) (string, error) {
	var resp responses.MultishiftCreateAssignment

	_, err := endpoint.makeRequest(http.MethodPost, "/satellite/hostqueue/createAssignment", config, &resp)
	return resp.HostID, err
}

// GetHosts retrieves all Satellite hosts for a given location
func (endpoint *V2Endpoint) GetHosts(location string) (rawJSON json.RawMessage, v2SatResponse []responses.MultishiftQueueNode, err error) {
	var buf bytes.Buffer
	path := fmt.Sprintf("/satellite/hostqueue/getHosts?controller=%s", location)
	_, err = endpoint.makeRequest(http.MethodGet, path, nil, &buf)
	if err != nil {
		return
	}
	rawJSON = buf.Bytes()

	err = json.Unmarshal(rawJSON, &v2SatResponse)

	return
}

// GetHost retrieves the specified Satellite host for a given location
func (endpoint *V2Endpoint) GetHost(location, host string) (rawJSON json.RawMessage, hostResponse responses.MultishiftQueueNode, err error) {
	var v2SatResponse []responses.MultishiftQueueNode

	path := fmt.Sprintf("/satellite/hostqueue/getHosts?controller=%s", location)
	_, err = endpoint.makeRequest(http.MethodGet, path, nil, &v2SatResponse)
	if err != nil {
		return
	}

	for _, h := range v2SatResponse {
		if h.ID == host || h.Name == host {

			rawJSON, err = json.Marshal(h)
			hostResponse = h
			break
		}
	}
	return
}

// RemoveHost removes an existing host from a location
func (endpoint *V2Endpoint) RemoveHost(location, host string) error {
	_, err := endpoint.makeRequest(http.MethodPost, "/satellite/hostqueue/removeHost", requests.MultishiftRemoveNode{Controller: location, NodeID: host}, nil)
	return err
}

// UpdateHost updates an existing Satellite host
func (endpoint *V2Endpoint) UpdateHost(config requests.MultishiftUpdateNode) error {
	_, err := endpoint.makeRequest(http.MethodPost, "/satellite/hostqueue/updateHost", config, nil)
	return err
}

// GetSubnets gets VPC-Gen2 subnets
func (endpoint *V2Endpoint) GetSubnets(vpcID string, zone string) (vpc.Subnets, error) {
	var subnets vpc.Subnets
	_, err := endpoint.makeRequest(http.MethodGet, fmt.Sprintf("/vpc/getSubnets?provider=%s&vpc=%s&zone=%s", models.ProviderVPCGen2, vpcID, zone), nil, &subnets)
	return subnets, err
}

// RemoveWorkerPool calls the v2 API to remove a worker pool
func (endpoint *V2Endpoint) RemoveWorkerPool(config requests.RemoveWorkerPool) error {
	_, err := endpoint.makeRequest(http.MethodPost, "/removeWorkerPool", config, nil)
	return err
}

// RemoveWorkerPoolZone calls the v2 API to remove a zone from a worker pool
func (endpoint *V2Endpoint) RemoveWorkerPoolZone(config requests.RemoveWorkerPoolZoneReq) error {
	_, err := endpoint.makeRequest(http.MethodPost, "/removeWorkerPoolZone", config, nil)
	return err
}

// ResizeWorkerPool calls the v2 API to resize a worker pool
func (endpoint *V2Endpoint) ResizeWorkerPool(config requests.ResizeWorkerPool) error {
	_, err := endpoint.makeRequest(http.MethodPost, "/resizeWorkerPool", config, nil)
	return err
}

// RebalanceWorkerPool calls the v2 API to rebalance a worker pool
func (endpoint *V2Endpoint) RebalanceWorkerPool(config requests.RebalanceWorkerPool) error {
	_, err := endpoint.makeRequest(http.MethodPost, "/rebalanceWorkerPool", config, nil)
	return err
}

// SetWorkerPoolTaints sets the taints on a worker pool
func (endpoint *V2Endpoint) SetWorkerPoolTaints(cluster, workerpool string, taints map[string]string) error {
	wpt := &requests.SetWorkerPoolTaints{
		Cluster:    cluster,
		WorkerPool: workerpool,
		Taints:     taints,
	}
	_, err := endpoint.makeRequest(http.MethodPost, "/setWorkerPoolTaints", wpt, nil)
	return err
}

// SetWorkerPoolLabels adds the labels to a worker pool
func (endpoint *V2Endpoint) SetWorkerPoolLabels(cluster, workerpool string, labels map[string]string) error {
	wpl := &requests.SetWorkerPoolLabels{
		Cluster:    cluster,
		WorkerPool: workerpool,
		Labels:     labels,
	}
	_, err := endpoint.makeRequest(http.MethodPost, "/setWorkerPoolLabels", wpl, nil)
	return err
}

// GetClusterALBs returns the list of albs available for any cluster
func (endpoint *V2Endpoint) GetClusterALBs(clusterNameOrID string) (rawJSON json.RawMessage, classicResponse ingress.ClusterALB, vpcResponse ingressVPC.ClusterALB, err error) {
	_, rawJSON, err = endpoint.makeRequestBindMulti(http.MethodGet, fmt.Sprintf("/alb/getClusterAlbs?cluster=%s", clusterNameOrID), nil, &classicResponse, &vpcResponse)
	return
}

// GetClusterALB returns details about particular alb for any cluster
func (endpoint *V2Endpoint) GetClusterALB(albID string) (rawJSON json.RawMessage, classicResponse ingress.ALBConfig, vpcResponse ingressVPC.ALBConfig, err error) {
	_, rawJSON, err = endpoint.makeRequestBindMulti(http.MethodGet, fmt.Sprintf("/alb/getAlb?albID=%s", albID), nil, &classicResponse, &vpcResponse)
	return
}

// GetALBImages returns list of supported alb versions
func (endpoint *V2Endpoint) GetALBImages() (versions ingressModel.Images, err error) {
	_, err = endpoint.makeRequest(http.MethodGet, "/alb/getAlbImages", nil, &versions)
	return
}

// UpdateALB updates the ALB image versions in the cluster
func (endpoint *V2Endpoint) UpdateALB(config ingressModel.V2UpdateALB) error {
	_, err := endpoint.makeRequest(http.MethodPost, "/alb/updateAlb", config, nil)
	return err
}

// GetNlbDNSList returns the list of nlb dns available for any cluster
func (endpoint *V2Endpoint) GetNlbDNSList(clusterNameOrID string) (rawJSON json.RawMessage, classicResponse []nlbDNSModel.NlbConfig, vpcResponse ingressVPC.V2NlbList, err error) {
	_, rawJSON, err = endpoint.makeRequestBindMulti(http.MethodGet, fmt.Sprintf("/nlb-dns/getNlbDNSList?cluster=%s", clusterNameOrID), nil, &classicResponse, &vpcResponse)
	return
}

// GetNlbDNSDetails returns the details of the gven nlb dns available for any cluster
func (endpoint *V2Endpoint) GetNlbDNSDetails(clusterNameOrID string, nlbHost string) (rawJSON json.RawMessage, classicResponse nlbDNSModel.NlbConfig, vpcResponse ingressVPC.NlbVPCListConfig, err error) {
	_, rawJSON, err = endpoint.makeRequestBindMulti(http.MethodGet, fmt.Sprintf("/nlb-dns/getNlbDetails?cluster=%s&nlbSubdomain=%s", clusterNameOrID, nlbHost), nil, &classicResponse, &vpcResponse)
	return
}

// RemoveLBHostname removes load balancer from nlb dns
func (endpoint *V2Endpoint) RemoveLBHostname(config ingressVPC.NlbVPCConfig) error {
	_, err := endpoint.makeRequest(http.MethodPost, "/nlb-dns/vpc/removeLBHostname", config, nil)
	return err
}

// Observability
// #############

// OBListConfig makes the API call to list the configurations associated with a cluster.
// serviceType -- logging or monitoring
// cluster -- cluster name or cluster ID
func (endpoint *V2Endpoint) OBListConfig(serviceType, cluster string) (obmodel.ObsConfigs, error) {
	var instances obmodel.ObsConfigs
	if _, err := endpoint.makeRequest(http.MethodGet, fmt.Sprintf("/observe/%s/getConfigs?cluster=%s", serviceType, url.QueryEscape(cluster)), nil, &instances); err != nil {
		return nil, err
	}
	return instances, nil
}

// OBCreateConfig makes the API call to create a new configuration
// serviceType -- logging or monitoring
// cluster -- cluster name or cluster ID
// ingestionKey -- logdna or sysdig ingestion key
// instance -- logdna or sysdig instance name or ID
// privateEndpoint -- boolean indicating if private endpoints should be used
func (endpoint *V2Endpoint) OBCreateConfig(serviceType, cluster, ingestionKey, instance string, privateEndpoint bool) error {

	// Create the config parameters
	configBody := obmodel.ConfigBody{
		Cluster:         cluster,
		Instance:        instance,
		IngestionKey:    ingestionKey,
		PrivateEndpoint: privateEndpoint,
	}

	var successV obmodel.ConfigResponse

	// make request to create cluster
	if _, err := endpoint.makeRequest(http.MethodPost, fmt.Sprintf("/observe/%s/createConfig", serviceType), configBody, &successV); err != nil {
		return err
	}

	return nil
}
