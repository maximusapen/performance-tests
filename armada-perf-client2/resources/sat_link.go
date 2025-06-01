/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2021 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package resources

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models/link"
)

// SatLinkEndpoint communicates with the Satellite Link APIs
type SatLinkEndpoint struct {
	Base *ArmadaEndpoint
}

// GetSatLinkEndpoint returns a new SatLinkEndpoint
func GetSatLinkEndpoint(c *cli.Context) *SatLinkEndpoint {
	endpoint := GetArmadaEndpoint(c)
	endpoint.BaseURLPath = cliutils.GetMetadataValue(c, "satlinkEndpoint").(string)
	endpoint.APIVersion = "v1"
	return &SatLinkEndpoint{endpoint}
}

func (s *SatLinkEndpoint) makeRequest(method, path string, body interface{}, successV interface{}) (*http.Response, error) {
	return s.Base.makeRequest(method, path, body, successV)
}

// GetEndpoint retrieves a Satellite Link endpoint
func (s *SatLinkEndpoint) GetEndpoint(location, endpointID string) (raw json.RawMessage, endpoint link.Endpoint, notFound bool, err error) {
	var hr *http.Response
	hr, err = s.makeRequest(http.MethodGet, fmt.Sprintf("/locations/%s/endpoints/%s", location, endpointID), nil, &raw)
	if err == nil {
		err = json.Unmarshal(raw, &endpoint)
	} else {
		notFound = (hr.StatusCode == http.StatusNotFound)
	}
	return raw, endpoint, notFound, err
}

// ListEndpoints retrieves a list of Satellite Link endpoints
func (s *SatLinkEndpoint) ListEndpoints(location string) (json.RawMessage, []link.Endpoint, error) {
	response := link.ResponseListEndpoints{
		Endpoints: make([]link.Endpoint, 0),
	}
	var raw json.RawMessage
	_, err := s.makeRequest(http.MethodGet, fmt.Sprintf("/locations/%s/endpoints", location), nil, &raw)
	if err == nil {
		err = json.Unmarshal(raw, &response)
	}
	return raw, response.Endpoints, err
}

// CreateEndpoint creates a Satellite Link endpoint
func (s *SatLinkEndpoint) CreateEndpoint(location string, endpoint link.Endpoint) (id string, err error) {
	var response link.Endpoint
	_, err = s.makeRequest(http.MethodPost, fmt.Sprintf("/locations/%s/endpoints", location), endpoint, &response)
	return response.ID, err
}

// DeleteEndpoint removes a Satellite Link endpoint
func (s *SatLinkEndpoint) DeleteEndpoint(location, endpointID string) error {
	_, err := s.makeRequest(http.MethodDelete, fmt.Sprintf("/locations/%s/endpoints/%s", location, endpointID), nil, nil)
	return err
}
