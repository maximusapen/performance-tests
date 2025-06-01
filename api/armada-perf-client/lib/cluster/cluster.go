

package cluster

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ghodss/yaml"

	"github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/config"
	"github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/request"
	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	k8sappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	clientcmd "k8s.io/client-go/tools/clientcmd"
)

// Cluster represents an armada cruiser or patrol
type Cluster struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Region            string `json:"region"`
	DataCenter        string `json:"dataCenter"`
	Location          string `json:"location"`
	ServerURL         string `json:"serverURL"`
	State             string `json:"state"`
	CreatedDate       string `json:"createDate"`
	ModifiedDate      string `json:"modifiedDate"`
	WorkerCount       int    `json:"workerCount"`
	IsPaid            bool   `json:"isPaid"`
	MasterKubeVersion string `json:"masterKubeVersion"`
	IngressHostname   string `json:"ingressHostname"`
	IngressSecretName string `json:"ingressSecretName"`
	configFile        string
	kubeClientset     *kubernetes.Clientset
	kubeAppsClient    *k8sappsv1.AppsV1Client
}

// Worker represents a cruiser or patrol worker
type Worker struct {
	ID           string `json:"id"`
	State        string `json:"state"`
	Status       string `json:"status"`
	PrivateVlan  string `json:"privateVlan"`
	PublicVlan   string `json:"publicVlan"`
	PrivateIP    string `json:"privateIP"`
	PublicIP     string `json:"publicIP"`
	MachineType  string `json:"machineType"`
	ErrorMessage string `json:"errorMessage"`
	Billing      string `json:"billing"`
	Isolation    string `json:"isolation"`
	KubeVersion  string `json:"kubeVersion"`
}

// AppPart identifies a component of an application
type AppPart struct {
	Kind string
	Name string
}

var (
	api request.Data
	//orgID        string
	//spaceID      string
	totalWorkers int
	// Debug enables verbose messages
	Debug bool
)

func isInitialized() bool {
	return true
}

// CreateCluster creates a new cluster
func CreateCluster(clusterName string, inTotalWorkers int) (Cluster, error) {
	if Debug {
		fmt.Printf("CreateCluster(): %s\n", clusterName)
	}

	var clust Cluster
	var err error

	totalWorkers = inTotalWorkers

	isInitialized()

	api = request.Data{Action: config.ActionCreateCluster, ClusterName: clusterName, TotalWorkers: totalWorkers}
	result := request.PerformRequest(api, true)

	if result.StatusCode == http.StatusCreated {
		var dat map[string]interface{}
		if err := json.Unmarshal(result.Body, &dat); err != nil {
			fmt.Println(err)
			return clust, err
		}
		clusterID := dat["id"].(string)
		result := request.PerformRequest(request.Data{Action: config.ActionGetCluster, ClusterName: clusterID, TotalWorkers: totalWorkers}, false)

		clust = initCluster(result.Body)
	} else {
		fmt.Println(result.Status)
		err = fmt.Errorf("CreateCluster action returned: %s", result.Status)
		if result.StatusCode == http.StatusConflict {
			fmt.Println("Assume its a conflict with existing cluster and delete it")
			api = request.Data{Action: config.ActionDeleteCluster, ClusterName: clusterName}
			result := request.PerformRequest(api, true)

			if result.StatusCode == http.StatusOK {
				fmt.Println("... Got OK on delete")
			}
		}
	}

	return clust, err
}

// GetClusters returns a list of clusters active within the org/space
func GetClusters() ([]Cluster, error) {
	if Debug {
		fmt.Println("GetClusters()")
	}
	isInitialized()
	var dat []Cluster
	var err error
	api := request.Data{Action: config.ActionGetClusters}
	result := request.PerformRequest(api, true)

	if result.StatusCode == http.StatusOK {
		if err = json.Unmarshal(result.Body, &dat); err != nil {
			if Debug {
				fmt.Println(err)
			}
		}
	} else if result.StatusCode == http.StatusUnauthorized {
		panic("ERROR: Unauthorized to make API requests")
	} else {
		if Debug {
			fmt.Println(result.Status)
		}
		err = fmt.Errorf("GetCluster action returned: %s", result.Status)
	}
	return dat, err
}

func initCluster(body []byte) Cluster {
	var cl Cluster
	if err := json.Unmarshal(body, &cl); err != nil {
		fmt.Println("ERROR: initCluster()")
		panic(err)
	}

	return cl
}

func (cl *Cluster) getClusterConfig() bool {
	if Debug {
		fmt.Printf("GetClusterConfig(): %s\n", cl.Name)
	}
	var success = false
	var err error

	result := request.PerformRequest(request.Data{Action: config.ActionGetClusterConfig, ClusterName: cl.Name}, false)
	if result.StatusCode == http.StatusOK && result.ContentType == "application/zip" {
		var zipFile = "./" + cl.ID + ".zip"
		var zipData *zip.ReadCloser
		if f, err := os.Create(zipFile); err == nil {
			defer f.Close()
			if _, err = f.Write(result.Body); err == nil {
				f.Sync()

				// Open a zip archive for reading.
				if zipData, err = zip.OpenReader(zipFile); err == nil {
					defer zipData.Close()
					defer os.Remove(zipFile)
				}
			}
		}

		if err != nil {
			fmt.Println("Couldn't create or open file for zip data: ", err)
		} else {

			var folder string
			var fileName string

			// Iterate through the files in the archive
			// Create config in consistently name files: <cluster name>/kube.yml
			// This makes it easy to use with kubectl
			for _, f := range zipData.File {
				if strings.HasSuffix(f.Name, "/") {
					// Create the directory if it doesn't exist
					if _, err = os.Stat(cl.Name); os.IsNotExist(err) {
						err = os.Mkdir(cl.Name, 0750)
					}
					if err != nil {
						fmt.Printf("Couldn't create directory for k8s config: %s", err)
						break
					}
					folder = strings.TrimSuffix(f.Name, "/")
				} else {
					// Create the file
					if strings.HasSuffix(f.Name, ".yml") {
						fileName = cl.Name + "/kube.yml"
						cl.configFile = fileName
						if Debug {
							fmt.Println("Configuration for ", cl.Name, " is in ", fileName)
						}
					} else {
						fileName = strings.Replace(f.Name, folder, cl.Name, 1)
					}
					if rc, err := f.Open(); err == nil {
						var c *os.File
						if c, err = os.Create(fileName); err == nil {
							c.Chmod(0600)
							// #nosec G110
							if _, err = io.Copy(c, rc); err == nil {
								c.Sync()
							}
							c.Close()
						}
						rc.Close()
					}
					if err != nil {
						fmt.Println(err)
					} else {
						success = true
					}

				}
			}
		}
	} else if result.StatusCode == http.StatusUnauthorized {
		panic("ERROR: Unauthorized to make API requests")
	} else {
		fmt.Println("Bad status code", result.Status)
		fmt.Println(result)
		fmt.Println(result.Body)
	}

	return success
}

// Created returns true if a basic k8s operation can be completed
func (cl *Cluster) Created() bool {
	if Debug {
		fmt.Printf("Created(): %s\n", cl.Name)
	}
	cl.GetKubeClientSet()
	nodes, err := cl.kubeClientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if Debug {
		fmt.Printf("Created(): There are %d nodes in the cluster\n", len(nodes.Items))
	}
	return err == nil
}

// Nodes returns a list of k8s nodes
func (cl *Cluster) Nodes() *k8sv1.NodeList {
	fmt.Printf("Nodes(): %s\n", cl.Name)
	if Debug {
		cl.GetKubeClientSet()
	}
	nodes, _ := cl.kubeClientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if Debug {
		fmt.Printf("Nodes(): There are %d nodes in the cluster\n", len(nodes.Items))
	}
	return nodes
}

// CreateApp creates an application on the cluster
func (cl *Cluster) CreateApp(configFile string) ([]AppPart, error) {
	if Debug {
		fmt.Printf("CreateApp(): %s\n", configFile)
	}
	cl.GetKubeClientSet()
	var parts = make([]AppPart, 0, 10)
	var partErr error
	var err error

	var source []byte
	// #nosec G304
	source, err = ioutil.ReadFile(configFile)
	if err != nil {
		panic(err)
	}

	for _, s := range bytes.Split(source, []byte("---")) {
		if len(s) == 0 {
			continue
		}

		var obj k8sv1.ObjectReference
		err = yaml.Unmarshal(s, &obj)
		if err != nil {
			panic(err)
		}

		if Debug {
			fmt.Println(obj)
		}

		switch obj.Kind {
		case "Pod":
			var pod k8sv1.Pod
			err = yaml.Unmarshal(s, &pod)
			if err != nil {
				panic(err)
			}
			//fmt.Printf("Creating pod: %v\n", pod)
			parts = append(parts, AppPart{Kind: pod.Kind, Name: pod.Name})
			_, err = cl.kubeClientset.CoreV1().Pods(k8sv1.NamespaceDefault).Create(context.TODO(), &pod, metav1.CreateOptions{})
			if err != nil {
				fmt.Println("FLAG: ", err)
				if partErr == nil {
					partErr = err
				}
			}
		case "Service":
			var service k8sv1.Service
			err = yaml.Unmarshal(s, &service)
			if err != nil {
				panic(err)
			}
			//fmt.Printf("Creating service: %v\n", service)
			parts = append(parts, AppPart{Kind: service.Kind, Name: service.Name})
			_, err = cl.kubeClientset.CoreV1().Services(k8sv1.NamespaceDefault).Create(context.TODO(), &service, metav1.CreateOptions{})
			if err != nil {
				fmt.Println("FLAG: ", err)
				if partErr == nil {
					partErr = err
				}
			}
		case "Secret":
			var secret k8sv1.Secret
			err = yaml.Unmarshal(s, &secret)
			if err != nil {
				panic(err)
			}
			//fmt.Printf("Creating secret: %v\n", secret)	// pragma: allowlist secret
			parts = append(parts, AppPart{Kind: secret.Kind, Name: secret.Name})
			_, err = cl.kubeClientset.CoreV1().Secrets(k8sv1.NamespaceDefault).Create(context.TODO(), &secret, metav1.CreateOptions{})
			if err != nil {
				fmt.Println("FLAG: ", err)
				if partErr == nil {
					partErr = err
				}
			}

		//TODO Support PersistentVolumeClaim

		case "StatefulSet":
			var statefulSet appsv1.StatefulSet
			err = yaml.Unmarshal(s, &statefulSet)
			if err != nil {
				panic(err)
			}
			//fmt.Printf("Creating statefulSet: %v\n", statefulSet)
			parts = append(parts, AppPart{Kind: statefulSet.Kind, Name: statefulSet.Name})
			_, err = cl.kubeAppsClient.StatefulSets(k8sv1.NamespaceDefault).Create(context.TODO(), &statefulSet, metav1.CreateOptions{})
			if err != nil {
				fmt.Println("FLAG: ", err)
				if partErr == nil {
					partErr = err
				}
			}
		default:
			fmt.Println("AppPart type is unknown: ", obj.Kind)
			panic("Object type not supported in pod configuration")
		}
	}

	return parts, partErr
}

// DeleteApp deletes an application running on the cluster
func (cl *Cluster) DeleteApp(parts []AppPart) error {
	if Debug {
		fmt.Printf("DeleteApp(): %s\n", parts)
	}
	cl.GetKubeClientSet()
	var firstErr error
	var err error

	for _, part := range parts {
		var delOpts metav1.DeleteOptions
		switch part.Kind {
		case "Pod":
			err = cl.kubeClientset.CoreV1().Pods(k8sv1.NamespaceDefault).Delete(context.TODO(), part.Name, *&delOpts)
		case "Service":
			err = cl.kubeClientset.CoreV1().Services(k8sv1.NamespaceDefault).Delete(context.TODO(), part.Name, *&delOpts)
		case "Secret":
			err = cl.kubeClientset.CoreV1().Secrets(k8sv1.NamespaceDefault).Delete(context.TODO(), part.Name, *&delOpts)
		case "StatefulSet":
			//TODO This doesn't delete the pods managed by the service. Weird. Delete service then its pods
			//     It may be the wait method has to wait on the pods to delete, not just the statefulset
			err = cl.kubeAppsClient.StatefulSets(k8sv1.NamespaceDefault).Delete(context.TODO(), part.Name, *&delOpts)
		}
		if err != nil {
			fmt.Println("FAILED: DeleteApp", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// WaitDeletedApp waits until each component of the application deleted
func (cl *Cluster) WaitDeletedApp(parts []AppPart) {
	if Debug {
		fmt.Printf("WaitDeletedApp(): %s\n", parts)
	}

	// Assumes if get returns error then the part has been deleted
	for _, part := range parts {
		var err error
		// TODO shouldn't be infinite
		for {
			if Debug {
				fmt.Println("Check status of ", part.Kind)
			}
			switch part.Kind {
			case "Pod":
				_, err = cl.kubeClientset.CoreV1().Pods(k8sv1.NamespaceDefault).Get(context.TODO(), part.Name, metav1.GetOptions{})
			case "Service":
				_, err = cl.kubeClientset.CoreV1().Services(k8sv1.NamespaceDefault).Get(context.TODO(), part.Name, metav1.GetOptions{})
			case "Secret":
				_, err = cl.kubeClientset.CoreV1().Secrets(k8sv1.NamespaceDefault).Get(context.TODO(), part.Name, metav1.GetOptions{})
			case "StatefulSet":
				_, err = cl.kubeAppsClient.StatefulSets(k8sv1.NamespaceDefault).Get(context.TODO(), part.Name, metav1.GetOptions{})
			}
			if err != nil {
				if Debug {
					fmt.Println(err)
				}
				break
			}
			time.Sleep(1 * time.Second)
		}
	}
}

// GetAppStatus returns true if all components of the application are running
func (cl *Cluster) GetAppStatus(parts []AppPart) (bool, []string) {
	if Debug {
		fmt.Printf("GetAppStatus(): %s\n", parts)
	}
	cl.GetKubeClientSet()
	var status = make([]string, len(parts))
	var succeed = true

	for key, part := range parts {
	nextPart:
		switch part.Kind {
		case "Pod":
			pod, err := cl.kubeClientset.CoreV1().Pods(k8sv1.NamespaceDefault).Get(context.TODO(), part.Name, metav1.GetOptions{})
			if err != nil {
				status[key] = "Failed"
				succeed = false
			} else {
				//status[key] = pod.Status.Phase
				switch pod.Status.Phase {
				case "Pending":
					status[key] = "Pending"
					succeed = false
				case "Running":
					status[key] = "Succeeded"
				default:
					if Debug {
						fmt.Println(status)
					}
					status[key] = "Failed"
					succeed = false
					break nextPart
				}
			}
		case "Service":
			_, err := cl.kubeClientset.CoreV1().Services(k8sv1.NamespaceDefault).Get(context.TODO(), part.Name, metav1.GetOptions{})
			if err != nil {
				status[key] = "Failed"
				succeed = false
			} else {
				// Service status is based on LB Ingress and not necissarily a status
				//     service.Status.LoadBalancer.Ingress
				// Return Succeeded if we can ge the service
				status[key] = "Succeeded"
			}
		case "Secret":
			_, err := cl.kubeClientset.CoreV1().Secrets(k8sv1.NamespaceDefault).Get(context.TODO(), part.Name, metav1.GetOptions{})
			if err != nil {
				status[key] = "Failed"
				succeed = false
			} else {
				status[key] = "Succeeded"
			}
		case "StatefulSet":
			statefulset, err := cl.kubeAppsClient.StatefulSets(k8sv1.NamespaceDefault).Get(context.TODO(), part.Name, metav1.GetOptions{})
			if err != nil {
				status[key] = "Failed"
				succeed = false
			} else {
				if Debug {
					fmt.Printf("statefulsets replicas: requested %d, actual: %d\n", *statefulset.Spec.Replicas, statefulset.Status.Replicas)
				}
				if *statefulset.Spec.Replicas == statefulset.Status.Replicas {
					/* Attempt at smarts for detecting statefulset status. Way beyond what is needed now.
					   //TODO: Basically return "Pending" if real status isn't know. Need to fix otherwise caller is
					   //      given responsibility for exiting loop after checking for interval
							fmt.Println("SS Selector", statefulset.Spec.Selector)
							var selector k8sv1.ListOptions
							//TODO a real stretch to just pick the name of the first container spec
							selector.LabelSelector = k8sv1.ListOptions{TypeMeta{Kind: "Pod"}, LabelSelector: statefulset.Spec.Template.Spec.Containers[0].Name}
							podList, err := cl.kubeClientset.Core().Pods(k8sv1.NamespaceDefault).List(selector)
							if err != nil {
								status[key] = "Failed"
								succeed = false
							}
							if int32(len(podList.Items)) == *statefulset.Spec.Replicas {
								for _, pod := range podList.Items {
									if pod.Status.Phase != "Running" {
										status[key] = "Pending"
										succeed = false
										break nextPart
									}
								}
								status[key] = "Succeeded"
							} else {
								status[key] = "Pending"
								succeed = false
								break
							}
					*/
					status[key] = "Succeeded"
				} else {
					status[key] = "Pending"
					succeed = false
				}
			}
		}
	}
	return succeed, status
}

// Delete remove the cluster from the carrier
func (cl *Cluster) Delete() error {
	if Debug {
		fmt.Printf("Delete(): %s\n", cl.Name)
	}

	var err error
	api = request.Data{Action: config.ActionDeleteCluster, ClusterName: cl.Name}
	result := request.PerformRequest(api, true)

	if result.StatusCode == http.StatusOK {
		if Debug {
			fmt.Println(result.Body)
		}
		var dat map[string]interface{}
		if err = json.Unmarshal(result.Body, &dat); err != nil {
			fmt.Println(err)
		}
		if Debug {
			fmt.Println("in Delete")
			fmt.Println(dat)
		}
	} else if result.StatusCode == http.StatusUnauthorized {
		panic("ERROR: Unauthorized to make API requests")
	}
	return err
}

// WaitForDeleted waits for up to 1 hour for the cluster to be deleted
func (cl *Cluster) WaitForDeleted() bool {
	if Debug {
		fmt.Printf("WaitForDeleted(): %s\n", cl.Name)
	}
	var success = false

	api = request.Data{Action: config.ActionGetCluster, ClusterName: cl.Name}

	// Keep checking for 60 minutes
	for i := 0; i < 180; i++ {
		if Debug {
			fmt.Printf("Testing for deleted: %s\n", cl.Name)
		}
		r := request.PerformRequest(api, false)

		if r.StatusCode == http.StatusNotFound {
			break
		} else if r.StatusCode == http.StatusUnauthorized {
			panic("ERROR: Unauthorized to make API requests")
		} else if r.StatusCode/100 != 2 {
			fmt.Println(r.Status)
			fmt.Println(api)
			fmt.Println("Unexpected status while checking for deleted, not sure how to proceed")
			break
		}

		time.Sleep(20 * time.Second)
	}
	return success
}

// WaitForDeployed waits for up to 1 hour for the cluster to be deployed
func (cl *Cluster) WaitForDeployed() bool {
	if Debug {
		fmt.Printf("WaitForDeployed(): %s\n", cl.Name)
	}
	var success = false

	api = request.Data{Action: config.ActionGetCluster, ClusterName: cl.Name}

	// Keep checking for 60 minutes: It used to be 20 minutes, but now cluster seems to go active after workers active.
outerLoop:
	for i := 0; i < 360; i++ {
		if Debug {
			fmt.Printf("Testing for deployed: %s\n", cl.Name)
		}
		r := request.PerformRequest(api, false)

		if r.StatusCode == http.StatusOK {
			var dat map[string]interface{}
			if err := json.Unmarshal(r.Body, &dat); err != nil {
				fmt.Println(err)
			}
			switch dat["state"] {
			case "deployed", "normal":
				success = true
				if Debug {
					fmt.Printf("Workers: %d\n", dat["workerCount"])
				}
				break outerLoop
			case "deploying", "pending":
			default:
				fmt.Printf("WaitForDeployed(): state = %s, %v\n", dat["state"], time.Now())
			}
		} else if r.StatusCode == http.StatusUnauthorized {
			panic("ERROR: Unauthorized to make API requests")
		} else {
			fmt.Println(r.Status)
			fmt.Println(api)
			fmt.Println("Unexpected status, not sure how to proceed")
			break
		}

		time.Sleep(10 * time.Second)
	}
	return success
}

// WaitForWorkers waits up to waitMinutes minutes for workers to be deployed
func (cl *Cluster) WaitForWorkers(waitMinutes int) bool {
	if Debug {
		fmt.Printf("WaitForWorkers(): %s\n", cl.Name)
	}
	var success = false

	api = request.Data{Action: config.ActionGetClusterWorkers, ClusterName: cl.Name}

	var waited float32
	for {
		fmt.Printf("Testing for deployed workers: %s\n", cl.Name)
		r := request.PerformRequest(api, false)

		if r.StatusCode == http.StatusOK {
			//fmt.Println(r.Body)
			var dat []Worker
			if err := json.Unmarshal(r.Body, &dat); err != nil {
				fmt.Println(err)
			}
			if len(dat) < totalWorkers {
				fmt.Printf("Number of workers (%d) is < expected workers (%d)\n", len(dat), totalWorkers)
				continue
			}
			var workersReady = true
			for j, w := range dat {
				//fmt.Println(w)
				switch w.State {
				case "deployed", "normal":
					if Debug {
						fmt.Printf("Worker [%d] is deployed\n", j)
					}
				default:
					workersReady = false
					if Debug {
						fmt.Printf("WaitForWorkers(): [%d].state = %s\n", j, w.State)
					}
					break
				}
			}

			if workersReady {
				success = true
				break
			}
		} else if r.StatusCode == http.StatusUnauthorized {
			panic("ERROR: Unauthorized to make API requests")
		} else {
			fmt.Println(r.Status)
			fmt.Println(api)
			fmt.Println("Unexpected status, not sure how to proceed.")
			break
		}

		if waitMinutes >= int(waited) {
			break
		}

		time.Sleep(30 * time.Second)
		fmt.Printf(".")
		waited += 0.5
	}
	fmt.Print(" done")
	return success
}

// GetKubeClientSet loads the k8s config for the cluster
func (cl *Cluster) GetKubeClientSet() *kubernetes.Clientset {
	if Debug {
		fmt.Println("GetKubeClientSet")
	}
	if cl.kubeClientset == nil {
		if cl.configFile == "" {
			cl.getClusterConfig()
		}

		if Debug {
			fmt.Println("Getting kube config")
		}
		config, err := clientcmd.BuildConfigFromFlags("", cl.configFile)
		if err != nil {
			panic(err.Error())
		}

		clientset, _ := kubernetes.NewForConfig(config)
		cl.kubeClientset = clientset

		if Debug {
			nodes, _ := cl.kubeClientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
			fmt.Printf("There are %d nodes in the cluster\n", len(nodes.Items))
		}

		v1cleint, _ := k8sappsv1.NewForConfig(config)
		cl.kubeAppsClient = v1cleint
	}

	return cl.kubeClientset
}
