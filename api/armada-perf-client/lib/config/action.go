

package config

import (
	"errors"
	"strings"
)

// ActionType defines an action 'enum' for supported Aramda API actions
type ActionType int

// Action enumerations
const (
	ActionUnspecified ActionType = iota
	ActionCreateCluster
	ActionGetClusters
	ActionGetCluster
	ActionGetClusterConfig
	ActionDeleteCluster
	ActionGetWorkerPools
	ActionCreateWorkerPool
	ActionRemoveWorkerPool
	ActionGetWorkerPool
	ActionResizeWorkerPool
	ActionAddWorkerPoolZone
	ActionRemoveWorkerPoolZone
	ActionGetClusterWorkers
	ActionAddClusterWorkers
	ActionGetWorker
	ActionDeleteWorker
	ActionRebootWorker
	ActionReloadWorker
	ActionGetDatacenters
	ActionGetRegions
	ActionGetZones
	ActionGetMachineTypes
	ActionGetVLANs
	ActionGetKubeVersions
	ActionCreateSubnet
	ActionGetSubnets
	ActionGetCredentials
	ActionSetCredentials
	ActionDeleteCredentials
	ActionGetVLANSpanning
	ActionUpdateCluster
	ActionAlignClusters
	ActionChurnClusters
	ActionApplyPullSecret
	ActionGetVersions
)

type action struct {
	name           string
	clusterAction  bool
	workerCreation bool
}

type actions []action

// Actions provides an enumeration to String mapping.
// Additionally, identifies whether the action operates on cluster(s)and/or worker(s)
var Actions = actions{
	action{
		name:           "Unspecified",
		clusterAction:  false,
		workerCreation: false,
	},
	action{
		name:           "CreateCluster",
		clusterAction:  true,
		workerCreation: true,
	},
	action{
		name:           "GetClusters",
		clusterAction:  false,
		workerCreation: false,
	},
	action{
		name:           "GetCluster",
		clusterAction:  true,
		workerCreation: false,
	},
	action{
		name:           "GetClusterConfig",
		clusterAction:  true,
		workerCreation: false,
	},
	action{
		name:           "DeleteCluster",
		clusterAction:  true,
		workerCreation: false,
	},
	action{
		name:           "GetWorkerPools",
		clusterAction:  true,
		workerCreation: false,
	},
	action{
		name:           "CreateWorkerPool",
		clusterAction:  true,
		workerCreation: true,
	},
	action{
		name:           "RemoveWorkerPool",
		clusterAction:  true,
		workerCreation: false,
	},
	action{
		name:           "GetWorkerPool",
		clusterAction:  true,
		workerCreation: false,
	},
	action{
		name:           "ResizeWorkerPool",
		clusterAction:  true,
		workerCreation: true,
	},
	action{
		name:           "AddWorkerPoolZone",
		clusterAction:  true,
		workerCreation: true,
	},
	action{
		name:           "RemoveWorkerPoolZone",
		clusterAction:  true,
		workerCreation: false,
	},
	action{
		name:           "GetClusterWorkers",
		clusterAction:  true,
		workerCreation: false,
	},
	action{
		name:           "AddClusterWorkers",
		clusterAction:  true,
		workerCreation: true,
	},
	action{
		name:           "GetWorker",
		clusterAction:  false,
		workerCreation: false,
	},
	action{
		name:           "DeleteWorker",
		clusterAction:  false,
		workerCreation: false,
	},
	action{
		name:           "RebootWorker",
		clusterAction:  true,
		workerCreation: false,
	},
	action{
		name:           "ReloadWorker",
		clusterAction:  true,
		workerCreation: false,
	},
	action{
		name:           "GetDatacenters",
		clusterAction:  false,
		workerCreation: false,
	},
	action{
		name:           "GetRegions",
		clusterAction:  false,
		workerCreation: false,
	},
	action{
		name:           "GetZones",
		clusterAction:  false,
		workerCreation: false,
	},
	action{
		name:           "GetMachineTypes",
		clusterAction:  false,
		workerCreation: false,
	},
	action{
		name:           "GetVLANs",
		clusterAction:  false,
		workerCreation: false,
	},
	action{
		name:           "GetKubeVersions",
		clusterAction:  false,
		workerCreation: false,
	},
	action{
		name:           "CreateSubnet",
		clusterAction:  true,
		workerCreation: false,
	},
	action{
		name:           "GetSubnets",
		clusterAction:  false,
		workerCreation: false,
	},
	action{
		name:           "GetCredentials",
		clusterAction:  false,
		workerCreation: false,
	},
	action{
		name:           "SetCredentials",
		clusterAction:  false,
		workerCreation: false,
	},
	action{
		name:           "DeleteCredentials",
		clusterAction:  false,
		workerCreation: false,
	},
	action{
		name:           "GetVLANSpanning",
		clusterAction:  false,
		workerCreation: false,
	},
	action{
		name:           "UpdateCluster",
		clusterAction:  true,
		workerCreation: false,
	},
	action{
		name:           "AlignClusters",
		clusterAction:  true,
		workerCreation: false,
	},
	action{
		name:           "ChurnClusters",
		clusterAction:  true,
		workerCreation: false,
	},
	action{
		name:           "ApplyPullSecret",
		clusterAction:  true,
		workerCreation: false,
	},
	action{
		name:           "GetVersions",
		clusterAction:  false,
		workerCreation: false,
	},
}

func (act actions) Strings() []string {
	var s []string
	for _, v := range act {
		s = append(s, v.name)
	}
	return s
}

func (act ActionType) String() string {
	return Actions[act].name
}

// HasCluster identifies whether this action(request) is associated with a cluster
func (act ActionType) HasCluster() bool {
	return Actions[act].clusterAction
}

// WorkerCreation identifies whether this action(request) is associated with worker(s)
func (act ActionType) WorkerCreation() bool {
	return Actions[act].workerCreation
}

// Set returns the string to enumeration mapping
func (act *ActionType) Set(value string) error {
	for i, v := range Actions {
		if strings.EqualFold(v.name, value) {
			*act = ActionType(i)
			return nil
		}
	}

	return errors.New("Invalid action")
}
