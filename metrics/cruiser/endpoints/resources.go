/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2018, 2021 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

/* Requires Metrics Server to be deployed. Metrics Server is available on 1.12 and above clusters.
 For simplicity, this utility currently uses anonymous user to access metrics.
Thus, the following ClusterRole and ClusterRoleBinding should be deployed on cluster */

/*
 apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: view-metrics
rules:
- apiGroups:
    - metrics.k8s.io
  resources:
    - pods
    - nodes
  verbs:
    - get
    - list
    - watch

---

apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: view-metrics
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: view-metrics
subjects:
  - apiGroup: rbac.authorization.k8s.io
    kind: User
	name: system:anonymous
*/

package endpoints

import (
	"context"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	metricsservice "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"
	"k8s.io/client-go/kubernetes"
)

type metadata struct {
	Name         string `json:"name"`
	CreationTime string `json:"creationTimestamp"`
}

type usage struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

// Resource defines operations on a kubernetes resurce tpye, e.g. node, pod
type Resource interface {
	Name() string
	Metrics() Resource
	BMMetrics() []metricsservice.BluemixMetric
	Unmarshal(data []byte) error
}

var (
	// Filter is a Regular Expression pattern for filtering on matching resources
	Filter string

	// Level at which metrics are to be collected. Higher levels will be aggregated.
	Level string

	// Testname is the (optional) name of a running test to be include in published metrics name
	Testname string

	// KubeClientset is the Kubernetes Clientset associated with the user supplied config
	KubeClientset *kubernetes.Clientset

	re       *regexp.Regexp
	fatallog = log.New(os.Stderr, log.Prefix(), log.LstdFlags|log.Lshortfile)
)

// Returns cpu in nano cores, and memory in bytes
func parseMetric(m, s string) int64 {
	var idx, mult = 1, 1

	if s == "" {
		// Some containers may have activities so low that cpu usage is not recorded.
		// Not seen in mem so far.
		// Setting to 0 if s is empty string.
		s = "0"
	}

	switch m {
	// Store cpu in nano cores
	case "cpu":
		switch s[len(s)-1:] {
		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx = 0
			mult = 1e9
			break
		case "n":
			break
		case "u":
			mult = 1e3
		case "m":
			mult = 1e6
		default:
			fatallog.Fatalf("Unrecognized unit : %s\n", s)
		}
	case "mem":
		// Store memory in bytes
		switch s[len(s)-1:] {
		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx = 0
			break
		case "i":
			idx = 2
			switch s[len(s)-2 : len(s)-1] {
			case "K":
				mult = 1024
			case "M":
				mult = 1024 * 1024
			case "G":
				mult = 1024 * 1024 * 1024
			default:
				fatallog.Fatalf("Unrecognized unit : %s\n", s)
			}
		default:
			fatallog.Fatalf("Unrecognized unit : %s\n", s)
		}
	default:
		fatallog.Fatalf("Unrecognized metric type '%s'. Expected 'cpu' or 'mem'\n", m)
	}

	val, err := strconv.ParseInt(s[:len(s)-idx], 10, 64)
	if err != nil {
		fatallog.Fatalf("Cannot parse cpu value: '%s'\n", s)
	}

	return val * int64(mult)
}

// Metrics returns resource data via Kuberrnetes metrics api
func Metrics(r Resource) Resource {
	const retries = 2
	var err error

	// Process --filter regexp if supplied
	if len(Filter) > 0 {
		re, err = regexp.Compile(Filter)
		if err != nil {
			fatallog.Fatalf("Invalid filter %s - %s\n", Filter, err.Error())
		}
	}

	// Get metrics from Kubernetes metrics server
	for i := 0; i < retries; i++ {
		var data []byte
		data, err = KubeClientset.
			RESTClient().
			Get().
			AbsPath(
				strings.Join([]string{
					"apis/metrics.k8s.io/v1beta1",
					r.Name()},
					"/")).
			DoRaw(context.TODO())

		// Successful call to metrics server ?
		if err == nil {
			// Store JSON data in our struct
			err = r.Unmarshal(data)
			if err != nil {
				fatallog.Fatalf("Failure parsing %s data metrics response - %s\n", r.Name(), err)
			}

			return r
		}

		// Maybe transitory problem, retry in 5s
		time.Sleep(5 * time.Second)
	}

	log.Printf("WARNING: Failure getting cruiser %s metrics - %s\n", r.Name(), err)
	return nil
}
