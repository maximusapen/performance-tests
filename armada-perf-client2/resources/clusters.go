/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2019, 2021 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package resources

import (
	"fmt"
	"strings"

	v2 "github.ibm.com/alchemy-containers/armada-api-model/json/v2"
)

// Cluster represents a cluster to render in the terminal
type Cluster interface {
	ID() string
	Name() string
	ClusterType() string
	State() string
	CreatedDate() string
	WorkerCount() int
	Location() string
	EOS() string
	MasterVersion() string
	MasterState() string
	MasterStatus() string
	MasterHealth() string
	MasterURL() string
	TargetVersion() string
	ResourceGroupName() string
	Provider() string
	IngressHostname() string
	IngressSecret() string
	IngressStatus() string
	IngressMessage() string
}

type cluster struct {
	id                string
	name              string
	state             string
	createdDate       string
	workerCount       int
	location          string
	eos               string
	masterVersion     string
	targetVersion     string
	resourceGroupName string
	provider          string
	masterState       string
	masterStatus      string
	masterHealth      string
	masterURL         string
	ingressHostname   string
	ingressSecret     string
	ingressStatus     string
	ingressMessage    string
	clusterType       string
}

// GetVersionString retrieve version for display in cluster output
func GetVersionString(masterVersion, targetVersion string) (versionString string) {
	versionString = masterVersion
	if !strings.Contains(masterVersion, targetVersion) {
		versionString = fmt.Sprintf("%s* (%s latest)", versionString, targetVersion)
	}
	return
}

// NewClusterFromCommon returns a cluster that may be rendered in the terminal
func NewClusterFromCommon(c v2.ClusterCommon) Cluster {
	return &cluster{
		id:                c.ID,
		name:              c.Name,
		state:             c.State,
		createdDate:       c.CreatedDate,
		workerCount:       c.WorkerCount,
		location:          c.Location,
		eos:               c.EOS,
		masterVersion:     c.MasterVersion,
		targetVersion:     c.TargetVersion,
		resourceGroupName: c.ResourceGroupName,
		provider:          c.ProviderID,
	}
}

// NewClusterFromCommons returns a cluster that may be rendered in the terminal
func NewClusterFromCommons(c v2.ClusterCommon, sc v2.SingleClusterCommon) Cluster {
	return &cluster{
		id:                c.ID,
		name:              c.Name,
		clusterType:       c.Type,
		state:             c.State,
		createdDate:       c.CreatedDate,
		workerCount:       c.WorkerCount,
		location:          c.Location,
		eos:               c.EOS,
		masterVersion:     c.MasterVersion,
		targetVersion:     c.TargetVersion,
		resourceGroupName: c.ResourceGroupName,
		provider:          c.ProviderID,

		masterURL:    c.MasterURL,
		masterState:  sc.Lifecycle.MasterState,
		masterStatus: sc.Lifecycle.MasterStatus,
		masterHealth: sc.Lifecycle.MasterHealth,

		ingressHostname: c.Ingress.Hostname,
		ingressSecret:   c.Ingress.SecretName, // pragma: allowlist secret
		ingressStatus:   c.Ingress.Status,
		ingressMessage:  c.Ingress.Message,
	}
}

func (c *cluster) ID() string {
	return c.id
}
func (c *cluster) Name() string {
	return c.name
}
func (c *cluster) ClusterType() string {
	return c.clusterType
}
func (c *cluster) State() string {
	return c.state
}
func (c *cluster) CreatedDate() string {
	return c.createdDate
}
func (c *cluster) WorkerCount() int {
	return c.workerCount
}
func (c *cluster) Location() string {
	return c.location
}
func (c *cluster) EOS() string {
	return c.eos
}
func (c *cluster) MasterVersion() string {
	return c.masterVersion
}
func (c *cluster) TargetVersion() string {
	return c.targetVersion
}
func (c *cluster) ResourceGroupName() string {
	return c.resourceGroupName
}

func (c *cluster) Provider() string {
	return c.provider
}

func (c *cluster) MasterState() string {
	return c.masterState
}

func (c *cluster) MasterStatus() string {
	return c.masterStatus
}

func (c *cluster) MasterHealth() string {
	return c.masterHealth
}

func (c *cluster) MasterURL() string {
	return c.masterURL
}

func (c *cluster) IngressHostname() string {
	return c.ingressHostname
}

func (c *cluster) IngressSecret() string {
	return c.ingressSecret
}

func (c *cluster) IngressStatus() string {
	return c.ingressStatus
}

func (c *cluster) IngressMessage() string {
	return c.ingressMessage
}
