/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2021 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package link

// NOTE: The API uses "client" to refer to a link's source. "server" refers to a destination.

// ResponseListEndpoints contains multiple Endpoint objects
type ResponseListEndpoints struct {
	Endpoints []Endpoint `json:"endpoints"`
}

// Endpoint is a Satellite Link endpoint object
type Endpoint struct {
	ID string `json:"endpoint_id,omitempty"`

	Certificates          CertificateSet `json:"certs"`
	CompressData          *bool          `json:"compress_data,omitempty"`
	DestinationHost       string         `json:"server_host,omitempty"`
	DestinationMutualAuth *bool          `json:"server_mutual_auth,omitempty"`
	DestinationPort       *uint          `json:"server_port,omitempty"`
	DestinationProtocol   string         `json:"server_protocol,omitempty"` // default auto-selected by API
	DestinationType       string         `json:"conn_type,omitempty"`
	IdleTimeoutSeconds    *int           `json:"timeout,omitempty"` // default, no idle timeout (0)
	Name                  string         `json:"display_name,omitempty"`
	SNIHostname           string         `json:"sni,omitempty"` // same as destination host
	SatelliteLocation     string         `json:"location_id,omitempty"`
	SourceHost            string         `json:"client_host,omitempty"`
	SourceMutualAuth      *bool          `json:"client_mutual_auth,omitempty"`
	SourcePort            *uint          `json:"client_port,omitempty"`
	SourceProtocol        string         `json:"client_protocol,omitempty"`
	Status                string         `json:"status,omitempty"`
	VerifyDestination     *bool          `json:"reject_unauth,omitempty"` // verify destination CA
}

// CertificateSet is effectively a map of each possible certificate type
type CertificateSet struct {
	Connector   *Certificate `json:"connector,omitempty"`
	Destination *Certificate `json:"server,omitempty"`
	Source      *Certificate `json:"client,omitempty"`
}

// Certificate represents a certificate file and it's key file
type Certificate struct {
	Certificate File `json:"cert,omitempty"`
	Key         File `json:"key,omitempty"`
}

// File represents a certificate or key file, with its name and file contents
type File struct {
	Name     string `json:"filename,omitempty"`
	Contents string `json:"file_contents,omitempty"`
}
