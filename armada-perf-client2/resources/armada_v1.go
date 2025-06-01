/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2019, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/urfave/cli"

	"github.com/IBM-Cloud/ibm-cloud-cli-sdk/common/rest"

	v2 "github.ibm.com/alchemy-containers/armada-api-model/json/v2"
	"github.ibm.com/alchemy-containers/armada-circuit/protect"
	nlbDNSModel "github.ibm.com/alchemy-containers/armada-dns-model/v2/dns"
	errModel "github.ibm.com/alchemy-containers/armada-model/errors"
	apiModelCommon "github.ibm.com/alchemy-containers/armada-model/model/api/json"
	apiModelV1 "github.ibm.com/alchemy-containers/armada-model/model/api/json/v1"

	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"

	jwt "github.com/golang-jwt/jwt/v4"
)

const (
	requestIDHeader = "X-Request-ID"
	userAgent       = " Performance Client"
)

// ArmadaEndpoint struct defining config info for the Armada API
type ArmadaEndpoint struct {
	Endpoint
	Region            string
	ResourceGroup     string
	SoftlayerUsername string
	SoftlayerAPIKey   string
	Global            bool
	Locations         []string
	Locale            string
	Version           string
}

// ListerEndpoint allows other methods to remain unaware of which endpoint implementation is in use for list calls.
type ListerEndpoint interface {
	GetAllClusters() ([]Cluster, json.RawMessage, error)
}

// APIError represents an error returned from the IKS API
type APIError struct {
	errModel.ErrorResponse
	Data string `json:"data"`
}

var debug = false
var verbose = false

func (e APIError) Error() string {
	var eString string
	if e.RecoveryCLI == "" {
		eString = fmt.Sprintf("%s (%s)\n\nIncident ID: %s", e.Err, e.ID, e.ReqID)
	} else {
		eString = fmt.Sprintf("%s (%s)\n\n%s\nIncident ID: %s", e.Err, e.ID, e.RecoveryCLI, e.ReqID)
	}
	return eString
}

// NonCriticalError is a kind of error that should be presented to the user, but not fail the command
type NonCriticalError struct {
	errs []string
}

func (e NonCriticalError) Error() string {
	return strings.Join(e.errs, "\n\n")
}

func (endpoint *ArmadaEndpoint) newRequest(method, path string, region string, body interface{}, headers map[string]string) *rest.Request {
	// Standard Armada request header
	r := rest.GetRequest(fmt.Sprintf("%s%s", endpoint.getBaseURL(), path)).Method(method)

	accessToken := endpoint.AccessToken
	if !strings.HasPrefix(strings.ToLower(accessToken), "bearer ") {
		accessToken = "Bearer " + accessToken
	}

	r.Set("Authorization", accessToken)
	r.Set("X-Auth-Refresh-Token", endpoint.RefreshToken)
	r.Set("X-Auth-Resource-Account", endpoint.AccountID)
	r.Set("User-Agent", userAgent+" "+endpoint.Version)

	// Set the region header if required, typically if using global endpoint
	if len(region) > 0 {
		r.Set("X-Region", region)
	}

	for _, location := range endpoint.Locations {
		r.Query("location", location)
	}

	r.Set("Accept", "application/json")
	r.Set("Content-Type", "application/json")

	// Optional headers
	if endpoint.SoftlayerUsername != "" {
		r.Set("X-Auth-Softlayer-Username", endpoint.SoftlayerUsername)
	}
	if endpoint.SoftlayerAPIKey != "" {
		r.Set("X-Auth-Softlayer-APIKey", endpoint.SoftlayerAPIKey)
	}
	//When the CLI is in debug mode, set a header that goes to Akamai to help debug
	if debug {
		r.Set("Pragma", "akamai-x-cache-on, akamai-x-cache-remote-on, akamai-x-check-cacheable, akamai-x-get-cache-key, akamai-x-get-extracted-values, akamai-x-get-nonces, akamai-x-get-ssl-client-session-id, akamai-x-get-true-cache-key, akamai-x-serial-no, akamai-x-get-request-id, akamai-x-feo-trace")
	}

	r.Body(body)

	return r
}

func (endpoint *ArmadaEndpoint) makeRequestWithHeaders(method, path string, region string, headers map[string]string, body, successV, errorV interface{}) (*http.Response, error) {
	err := endpoint.refreshTokensIfNeeded()
	if err != nil {
		return nil, err
	}
	req := endpoint.newRequest(method, path, region, body, headers)

	return endpoint.doRequest(req, successV, errorV)
}

func (endpoint *ArmadaEndpoint) makeRequest(method, path string, body interface{}, successV interface{}) (*http.Response, error) {
	errorV := new(APIError)

	iksConfig := cliutils.GetArmadaConfig().IKS
	return endpoint.makeRequestWithHeaders(method, path, iksConfig.Region, nil, body, successV, errorV)
}

func (endpoint *V2Endpoint) makeRequestBindMulti(method, path string, body interface{}, successResponses ...interface{}) (resp *http.Response, rawJSON json.RawMessage, err error) {
	buf := bytes.NewBuffer(nil)
	resp, err = endpoint.makeRequest(method, path, body, buf)
	if err != nil {
		return
	}
	rawJSON = buf.Bytes()
	for _, successV := range successResponses {
		switch jsonErr := json.Unmarshal(rawJSON, successV).(type) {
		case *json.UnsupportedTypeError, *json.UnmarshalTypeError:
			// this is expected when unmarshaling into multiple types
			continue
		default:
			if jsonErr != nil {
				err = jsonErr
				return
			}
		}
	}
	return
}
func (endpoint *ArmadaEndpoint) doRequest(req *rest.Request, successV, errorV interface{}) (*http.Response, error) {
	// Make the request and parse results
	start := time.Now()
	builtRequest, err := req.Build()
	if err != nil {
		fmt.Printf("An error occurred while building request:\n%s\n", err.Error())
		return nil, err
	}
	if endpoint.Context == nil {
		endpoint.Context = context.Background()
	}
	builtRequest = builtRequest.WithContext(endpoint.Context)

	dumpedRequest, err := httputil.DumpRequest(builtRequest, false)
	if err != nil {
		fmt.Printf("An error occurred while dumping request:\n%s\n", err.Error())
		return nil, err
	}

	if debug {
		fmt.Printf("\n%s\n%s\n",
			start.Format(time.RFC3339),
			string(dumpedRequest))
	}

	handleErrorFunc := endpoint.handleError

	resp, respErr := endpoint.requestWithRetries(req, successV, errorV)
	end := time.Now()

	if restClientErr, ok := respErr.(*rest.ErrorResponse); ok && resp != nil && strings.Contains(resp.Header.Get("Content-Type"), "text") {
		// do not show raw HTML in CLI error output
		if debug {
			fmt.Printf("Raw error contents from server: %v\n", restClientErr)
		}
		respErr = fmt.Errorf("Error response from server. Status code: %d", restClientErr.StatusCode)
	}

	if ctxErr := endpoint.Context.Err(); ctxErr != nil {
		return resp, respErr
	}

	err = handleErrorFunc(resp, respErr, successV, errorV)

	var dumpedResponse []byte
	var dumpErr error
	if resp != nil {
		dumpedResponse, dumpErr = httputil.DumpResponse(resp, false)
		if dumpErr != nil {
			fmt.Printf("An error occurred while dumping response:\n%s\n", dumpErr.Error())
		}
	}

	contents := fmt.Sprintf("%+v", successV)
	if resp != nil && resp.Header.Get("Content-Type") == "application/zip" {
		contents = "[Binary data]"
	}
	if contents == "" {
		errorVal := fmt.Sprintf("+%v", errorV)
		if errorVal != "" {
			contents = errorVal
		} else if err != nil {
			fmt.Printf("An error occurred while dumping response body:\n%s\n", err.Error())
		}
	}

	if debug {
		fmt.Println(string(dumpedResponse))
	}

	if verbose {
		fmt.Printf("Total Duration: %.3fs\n", end.Sub(start).Seconds())
	}
	return resp, err
}

func (endpoint *ArmadaEndpoint) handleError(resp *http.Response, respErr error, successV, errorV interface{}) error {
	if resp != nil && resp.StatusCode >= 300 && errorV != nil {
		gErr, ok := errorV.(*APIError)
		if ok && gErr.ReqID != "" {
			return gErr
		}
	}

	if respErr != nil {
		fmt.Println("Err", respErr)
		fmt.Printf("Request failed with error: %s, %v\n", respErr.Error(), resp)
		return respErr
	}

	if resp.StatusCode >= 300 {
		// Bad return code but the response didn't parse into the struct
		fmt.Printf("Request responded with unexpected return code %d: %v", resp.StatusCode, resp)
		return fmt.Errorf("Error response from server. Status code: %d", resp.StatusCode)
	}

	return nil
}

var retryStatuses = []int{
	http.StatusInternalServerError, // 500
	http.StatusRequestTimeout,      // 408
	http.StatusBadGateway,          // 502
	http.StatusServiceUnavailable,  // 503
	http.StatusGatewayTimeout,      // 504
}

// generateRequestID returns a request ID. If it fails to generate, returns an empty string.
func generateRequestID() string {
	id, idErr := uuid.NewRandom()
	if idErr != nil {
		return ""
	}
	return id.String()
}

func wrapConnectionError(requestID string, err error) error {
	if err == nil {
		return nil
	}
	if urlErr, ok := err.(*url.Error); ok && strings.Contains(urlErr.Error(), "net/http") {
		if debug {
			fmt.Println("HTTP Error:", urlErr)
		}
		err = errors.New("Unable to connect to " + urlErr.URL)
	}
	errString := fmt.Sprintf("Request failed to complete:\n%s", err.Error())
	if requestID != "" {
		errString += fmt.Sprintf("\n\nIncident ID: %s", requestID)
	}
	return errors.New(errString)
}

func (endpoint *ArmadaEndpoint) requestWithRetries(req *rest.Request, successV interface{}, errorV interface{}) (*http.Response, error) {
	var resp *http.Response
	requestID := ""
	retrier := protect.NewRetrier(nil)
	ctx := context.Background()
	retrierErr := retrier.Do(
		ctx,
		protect.Doer(func() error {
			requestID = generateRequestID()
			req.Set(requestIDHeader, requestID)
			if debug {
				fmt.Println("Running retry request.", requestIDHeader+":", requestID)
			}
			var err error
			resp, err = endpoint.client().Do(req, successV, errorV)

			if resp != nil && (resp.StatusCode >= 200 && resp.StatusCode < 300) {
				return nil
			}
			return err
		}),
		protect.DoRetry(func(err error) bool {
			httpReq, buildErr := req.Build()
			if buildErr != nil {
				return true
			}

			if httpReq.Method != http.MethodGet {
				return false
			}
			if resp != nil {
				for _, code := range retryStatuses {
					if resp.StatusCode == code {
						return true
					}
				}
				return false
			}
			return true
		}))
	if retrierErr != nil && resp == nil {
		retrierErr = wrapConnectionError(requestID, retrierErr)
	}
	return resp, retrierErr
}

// GetAPIVersion returns the default API version to use with request
func GetAPIVersion() string {
	return "v1"
}

func setAPIVersionForProvider(e *ArmadaEndpoint, provider string) ListerEndpoint {
	if IsVPCProvider(provider) || IsSatelliteProvider(provider) {
		e.APIVersion = "v2"
		return &V2Endpoint{e}
	}
	return e
}

// GetArmadaEndpoint provides an endpoint with the specified options
func GetArmadaEndpoint(c *cli.Context, opts ...EndpointOpt) *ArmadaEndpoint {
	endpoint := ArmadaEndpoint{
		Endpoint: Endpoint{
			APIVersion:   GetAPIVersion(),
			BaseURLPath:  cliutils.GetMetadataValue(c, "iksEndpoint").(string),
			AccessToken:  cliutils.GetMetadataValue(c, "accessToken").(string),
			RefreshToken: cliutils.GetMetadataValue(c, "refreshToken").(string),
			AccountID:    cliutils.GetMetadataValue(c, "accountID").(string),
			IAMPath:      cliutils.GetMetadataValue(c, "iamEndpoint").(string),
			Context:      context.Background(),
		},
		Version: c.App.Version,
	}

	return &endpoint
}

// GetEndpointForProvider returns the endpoint appropriate for the specified provider. The other return
// value is an error if an invalid value is specified, nil otherwise.
func GetEndpointForProvider(c *cli.Context, provider string, opts ...EndpointOpt) ListerEndpoint {
	e := GetArmadaEndpoint(c, opts...)
	return setAPIVersionForProvider(e, provider)
}

// GetAllClusters lists the clusters
func (endpoint *ArmadaEndpoint) GetAllClusters() ([]Cluster, json.RawMessage, error) {

	// Make the request
	var successV apiModelV1.Clusters
	var result []Cluster

	_, err := endpoint.makeRequest(http.MethodGet, "/clusters", nil, &successV)
	if len(successV) == 0 {
		// check successV instead of error to permit partial errors
		return result, []byte{}, err
	}

	result = make([]Cluster, 0, len(successV))
	for _, v1Cluster := range successV {
		result = append(
			result,
			NewClusterFromCommon(
				v2.ClusterCommon{
					ID:                v1Cluster.ID,
					Name:              v1Cluster.Name,
					State:             v1Cluster.State,
					CreatedDate:       v1Cluster.CreatedDate,
					WorkerCount:       v1Cluster.WorkerCount,
					Location:          v1Cluster.Location,
					EOS:               v1Cluster.EOS,
					MasterVersion:     v1Cluster.MasterVersion,
					TargetVersion:     v1Cluster.TargetVersion,
					ResourceGroupName: v1Cluster.ResourceGroupName,
					ProviderID:        models.ProviderClassic,
				},
			))
	}

	contents, marshalErr := json.MarshalIndent(successV, "", "    ")
	if marshalErr != nil {
		fmt.Printf("Unable to unmarshal JSON response: %s\n", marshalErr.Error())
		err = marshalErr
	}
	return result, json.RawMessage(contents), err
}

// CreateCluster makes the API call to create a cluster
func (endpoint *ArmadaEndpoint) CreateCluster(config apiModelV1.CreateClusterConfig) error {

	// make request to create cluster
	var r apiModelCommon.ClusterCreateResponse
	if _, err := endpoint.makeRequest(http.MethodPost, "/clusters", config, &r); err != nil {
		return err
	}

	// it is possible to receive a non-critical error in the response. This should be parsed
	// and returned to the user.
	if r.NonCriticalErrors != nil && len(r.NonCriticalErrors.Items) > 0 {

		// iterate the list of non-critical errors and surface as a message to display to
		// the user as a cluster create failure.
		errs := []string{"Your cluster will be created, but some operations did not complete and need your attention:"}
		for _, nce := range r.NonCriticalErrors.Items {
			errs = append(errs, fmt.Sprintf("%s: %s\n", nce.ID, nce.Err))
		}
		return NonCriticalError{errs: errs}
	}

	// cluster create successful
	return nil
}

// RemoveCluster makes the API call to remove a cluster
func (endpoint *ArmadaEndpoint) RemoveCluster(clusterNameOrID string, deleteResources bool) error {
	// Make the request
	_, err := endpoint.makeRequest(http.MethodDelete, fmt.Sprintf("/clusters/%s?deleteResources=%s", clusterNameOrID, strconv.FormatBool(deleteResources)), nil, nil)
	return err
}

// CreateWorkerPool makes the API call to create a worker pool
func (endpoint *ArmadaEndpoint) CreateWorkerPool(clusterNameOrID string, workerPoolNameOrID apiModelV1.WorkerPoolRequest) error {
	// Make the request
	_, err := endpoint.makeRequest(http.MethodPost, fmt.Sprintf("/clusters/%s/workerpools", clusterNameOrID), workerPoolNameOrID, nil)
	return err
}

// RemoveWorkerPool makes the API call to remove a worker-pool from a cluster
func (endpoint *ArmadaEndpoint) RemoveWorkerPool(clusterNameOrID string, poolNameOrID string) error {
	// Make the request
	_, err := endpoint.makeRequest(http.MethodDelete, fmt.Sprintf("/clusters/%s/workerpools/%s", clusterNameOrID, poolNameOrID), nil, nil)
	return err
}

// AddZone makes the API call to add a zone to a cluster's worker pool
func (endpoint *ArmadaEndpoint) AddZone(clusterNameOrID string, poolID string, workerPoolZone apiModelV1.WorkerPoolZone) error {
	// Make the request
	_, err := endpoint.makeRequest(http.MethodPost, fmt.Sprintf("/clusters/%s/workerpools/%s/zones", clusterNameOrID, poolID), workerPoolZone, nil)
	return err
}

// RemoveZone makes the API call to remove a zone from a worker-pool in a cluster
func (endpoint *ArmadaEndpoint) RemoveZone(clusterNameOrID string, poolNameOrID string, zone string) error {
	// Make the request
	_, err := endpoint.makeRequest(http.MethodDelete, fmt.Sprintf("/clusters/%s/workerpools/%s/zones/%s", clusterNameOrID, poolNameOrID, zone), nil, nil)
	return err
}

// AddonResponse handles both addon conflicts and general api errors
type AddonResponse struct {
	apiModelV1.AddonResponse
	APIError
}

func (endpoint *ArmadaEndpoint) patchClusterAddons(clusterNameOrID string, enable bool, update bool, addons ...apiModelV1.ClusterAddon) (AddonResponse, error) {
	req := apiModelV1.AddonRequest{
		Enable: &enable,
		Update: &update,
		Addons: addons,
	}

	var resp AddonResponse
	_, err := endpoint.makeRequest(http.MethodPatch, fmt.Sprintf("/clusters/%s/addons", clusterNameOrID), &req, &resp)
	return resp, err
}

// EnableClusterAddons enables addons for cluster
func (endpoint *ArmadaEndpoint) EnableClusterAddons(clusterNameOrID string, addons ...apiModelV1.ClusterAddon) (AddonResponse, error) {
	return endpoint.patchClusterAddons(clusterNameOrID, true, false, addons...)
}

// DisableClusterAddons disables addons for cluster
func (endpoint *ArmadaEndpoint) DisableClusterAddons(clusterNameOrID string, addons ...apiModelV1.ClusterAddon) (AddonResponse, error) {
	return endpoint.patchClusterAddons(clusterNameOrID, false, false, addons...)
}

// ListClusterAddons list addons enabled for cluster
func (endpoint *ArmadaEndpoint) ListClusterAddons(clusterNameOrID string) ([]apiModelV1.ClusterAddon, error) {
	var resp []apiModelV1.ClusterAddon
	_, err := endpoint.makeRequest(http.MethodGet, fmt.Sprintf("/clusters/%s/addons", clusterNameOrID), nil, &resp)
	return resp, err
}

// GetClusterConfig makes the API call to get cluster-specifc configuration and certificates
func (endpoint *ArmadaEndpoint) GetClusterConfig(clusterNameOrID string, admin, network bool) (*bytes.Buffer, error) {
	// Make the request
	clusterConfigQuery := fmt.Sprintf("/clusters/%s/config", clusterNameOrID)
	if admin {
		clusterConfigQuery += "/admin"
	}

	clusterConfigQuery = fmt.Sprintf("%s?createNetworkConfig=%s", clusterConfigQuery, strconv.FormatBool(network))

	var buf bytes.Buffer
	_, err := endpoint.makeRequest(http.MethodGet, clusterConfigQuery, nil, &buf)
	return &buf, err
}

// GetKubeVersions retrieves the list of supported kube versions
func (endpoint *ArmadaEndpoint) GetKubeVersions() (apiModelCommon.KubeVersions, error) {
	var kv apiModelCommon.KubeVersions
	if _, err := endpoint.makeRequest(http.MethodGet, "/kube-versions", nil, &kv); err != nil {
		return nil, err
	}
	return kv, nil
}

// GetVersions retrieves the map of supported IKS cluster versions (e.g. Kubernetes, OpenShift)
func (endpoint *ArmadaEndpoint) GetVersions() (map[string]apiModelCommon.KubeVersions, error) {
	var kv map[string]apiModelCommon.KubeVersions
	if _, err := endpoint.makeRequest(http.MethodGet, "/versions", nil, &kv); err != nil {
		return nil, err
	}
	return kv, nil
}

// GetAddonVersions retrieves the list of supported addons
func (endpoint *ArmadaEndpoint) GetAddonVersions() ([]apiModelV1.ClusterAddon, error) {
	var addons []apiModelV1.ClusterAddon
	if _, err := endpoint.makeRequest(http.MethodGet, "/addons", nil, &addons); err != nil {
		return nil, err
	}
	return addons, nil
}

// GetSubnets retrieves the list of Classic Iaas Subnets
func (endpoint *ArmadaEndpoint) GetSubnets() ([]apiModelV1.Subnet, error) {
	var subnets []apiModelV1.Subnet
	_, err := endpoint.makeRequest(http.MethodGet, "/subnets", nil, &subnets)
	return subnets, err
}

// AddNlbDNSIP add an IP to the subdomain
func (endpoint *ArmadaEndpoint) AddNlbDNSIP(clustername string, ips []string, nlbHost string) error {
	requestBody := nlbDNSModel.NlbConfig{
		ClusterID:  clustername,
		NlbIPArray: ips,
		NlbHost:    nlbHost,
	}

	_, err := endpoint.makeRequest(http.MethodPut, fmt.Sprintf("/nlb-dns/clusters/%s/add", clustername), requestBody, nil)
	return err
}

// DeleteNlbDNSIP deletes an IP from the subdomain
func (endpoint *ArmadaEndpoint) DeleteNlbDNSIP(clustername string, ip string, nlbHost string) error {
	_, err := endpoint.makeRequest(http.MethodDelete, fmt.Sprintf("/nlb-dns/clusters/%s/host/%s/ip/%s/remove", clustername, nlbHost, ip), nil, nil)
	return err
}

// parseJWTToken parses a JWT token without verifying the signature
func parseJWTToken(token string) (*jwt.StandardClaims, error) {
	// Remove "bearer " in front if there is any
	// to lowercase as the cli sends in Bearer
	if strings.HasPrefix(token, "bearer ") {
		segments := strings.SplitAfterN(token, "bearer ", 2)
		token = segments[len(segments)-1]
	} else if strings.HasPrefix(token, "Bearer ") {
		segments := strings.SplitAfterN(token, "Bearer ", 2)
		token = segments[len(segments)-1]
	}

	// Parse the token
	parsedToken, err := jwt.ParseWithClaims(token, &jwt.StandardClaims{}, nil)
	if parsedToken != nil {
		if claims, ok := parsedToken.Claims.(*jwt.StandardClaims); ok {
			return claims, nil
		} else if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return nil, errors.New("token malformed")
			}
		}
	}
	return nil, errors.New("cannot parse token")
}
