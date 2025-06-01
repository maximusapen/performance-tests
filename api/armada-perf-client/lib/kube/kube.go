
package kube

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"time"

	"github.com/ghodss/yaml"

	//v1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	k8sappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	clientcmd "k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/connrotation"
)

// Kube ...
type Kube struct {
	configFile     string
	kubeClientset  *kubernetes.Clientset
	kubeAppsClient *k8sappsv1.AppsV1Client
	ipPort         string
	dialer         *connrotation.Dialer
}

// AppPart identifies a component of an application
type AppPart struct {
	Kind string
	Name string
}

var (
	// Debug enables detailed logging
	Debug bool
	// PortActiveTimeout the timeout for calls to test whether a port is active
	PortActiveTimeout = time.Duration(10 * time.Second)
)

// CreateKube ...
func CreateKube(kubeConfig string, timeout time.Duration) (Kube, error) {
	if Debug {
		fmt.Printf("CreateKube(): %s\n", kubeConfig)
	}

	kube := Kube{configFile: kubeConfig}

	_, err := kube.GetKubeClientSet(timeout)

	return kube, err
}

// GetKubeClientSet loads the k8s config
func (kube *Kube) GetKubeClientSet(timeout time.Duration) (*kubernetes.Clientset, error) {
	if Debug {
		fmt.Println("GetKubeClientSet")
	}

	var err error

	if kube.kubeClientset == nil {
		if kube.configFile == "" {
			panic("Kube config file not defined")
		}

		if Debug {
			fmt.Println("Getting kube config")
		}
		config, err := clientcmd.BuildConfigFromFlags("", kube.configFile)
		if err != nil {
			fmt.Printf("ERROR: Building client configuration from %s, %v\n", kube.configFile, err.Error())
			return kube.kubeClientset, err
		}

		config.Timeout = timeout

		// TODO should do more checks like trailing '/'
		host := config.Host
		if strings.HasPrefix(host, "https://") {
			kube.ipPort = strings.TrimPrefix(host, "https://")
		}

		kube.dialer = connrotation.NewDialer((&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext)
		config.Dial = kube.dialer.DialContext

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			fmt.Printf("ERROR: Getting client set: %s - %v\n", kube.configFile, err.Error())
			panic(err)
		}
		kube.kubeClientset = clientset

		if Debug {
			nodes, _ := kube.kubeClientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
			fmt.Printf("There are %d nodes in the cluster\n", len(nodes.Items))
		}

		v1client, _ := k8sappsv1.NewForConfig(config)
		kube.kubeAppsClient = v1client
	}

	return kube.kubeClientset, err
}

// ResetConnections reinitializes connections to Kubernetes
func (kube *Kube) ResetConnections(timeout time.Duration) {
	kube.dialer.CloseAll()
	return
}

// GetDeploymentReplicas ...
func (kube *Kube) GetDeploymentReplicas(namespace string, name string) (int32, error) {
	deployment, err := kube.kubeClientset.AppsV1beta1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if Debug {
			fmt.Printf("ERROR: Getting replicas (namespace=%s, deployment=%s): %v\n", namespace, name, err)
		}
		return 0, err
	}
	return *deployment.Spec.Replicas, err
}

// PortActive check whether kube access port is active
func (kube *Kube) PortActive() (bool, error) {
	if len(kube.ipPort) > 0 {
		conn, err := net.DialTimeout("tcp", kube.ipPort, PortActiveTimeout)
		if err != nil {
			return false, err
		}
		conn.Close()
	}

	return true, nil
}

// Nodes returns a list of k8s nodes
func (kube *Kube) Nodes() *k8sv1.NodeList {
	//fmt.Printf("Nodes(): %s\n", kube.Name)
	nodes, _ := kube.kubeClientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if Debug {
		fmt.Printf("Nodes(): There are %d nodes in the cluster\n", len(nodes.Items))
	}
	return nodes
}

// GetServices ...
func (kube *Kube) GetServices() (*k8sv1.ServiceList, error) {
	svcs, err := kube.kubeClientset.CoreV1().Services(k8sv1.NamespaceDefault).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println("ERROR: Couldn't get services: ", err)
	}
	return svcs, err
}

// GetConfigMap ...
func (kube *Kube) GetConfigMap(namespace string, name string) (*k8sv1.ConfigMap, error) {
	configMap, err := kube.kubeClientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		fmt.Println("ERROR: Couldn't get config map in namespace of : ", namespace, ", name : ", name, err)
	}
	return configMap, err
}

// GetConfigMaps ...
func (kube *Kube) GetConfigMaps() (*k8sv1.ConfigMapList, error) {
	cfgmaps, err := kube.kubeClientset.CoreV1().ConfigMaps(k8sv1.NamespaceDefault).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println("ERROR: Couldn't get config maps: ", err)
	}
	return cfgmaps, err
}

// GetConfigMapsByNamespace ...
func (kube *Kube) GetConfigMapsByNamespace(namespace string) (*k8sv1.ConfigMapList, error) {
	// TODO consider specifying an override for k8sv1.NamespaceDefault in context of all *Kube calls
	//      so don't have to specify namespace each time
	cfgmaps, err := kube.kubeClientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println("ERROR: Couldn't get config maps with namespace of : ", namespace, err)
	}
	return cfgmaps, err
}

// GetSecret ...
func (kube *Kube) GetSecret(namespace string, name string) (*k8sv1.Secret, error) {
	secret, err := kube.kubeClientset.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		fmt.Println("ERROR: Couldn't get secret in namespace of : ", namespace, ", name : ", name, err)
	}
	return secret, err
}

// GetSecretsByNamespace ...
func (kube *Kube) GetSecretsByNamespace(namespace string) (*k8sv1.SecretList, error) {
	secrets, err := kube.kubeClientset.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println("ERROR: Couldn't get secrets with namespace of : ", namespace, err)
	}
	return secrets, err
}

// GetNamespaces ...
func (kube *Kube) GetNamespaces() (*k8sv1.NamespaceList, error) {
	namespaces, err := kube.kubeClientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println("ERROR: Couldn't get namespaces: ", err)
	}
	return namespaces, err
}

// CreateApp creates an application on the cluster
func (kube *Kube) CreateApp(appConfigFile string) ([]AppPart, error) {
	if Debug {
		fmt.Printf("CreateApp(): %s\n", appConfigFile)
	}
	var parts = make([]AppPart, 0, 10)
	var partErr error
	var err error

	var source []byte
	// #nosec G304
	source, err = ioutil.ReadFile(appConfigFile)
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
			_, err = kube.kubeClientset.CoreV1().Pods(k8sv1.NamespaceDefault).Create(context.TODO(), &pod, metav1.CreateOptions{})
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
			_, err = kube.kubeClientset.CoreV1().Services(k8sv1.NamespaceDefault).Create(context.TODO(), &service, metav1.CreateOptions{})
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
			_, err = kube.kubeClientset.CoreV1().Secrets(k8sv1.NamespaceDefault).Create(context.TODO(), &secret, metav1.CreateOptions{})
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
			_, err = kube.kubeAppsClient.StatefulSets(k8sv1.NamespaceDefault).Create(context.TODO(), &statefulSet, metav1.CreateOptions{})
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

// DeleteApp deletes an application running on kube
func (kube *Kube) DeleteApp(parts []AppPart) error {
	if Debug {
		fmt.Printf("DeleteApp(): %s\n", parts)
	}
	var firstErr error
	var err error

	for _, part := range parts {
		var delOpts metav1.DeleteOptions
		switch part.Kind {
		case "Pod":
			err = kube.kubeClientset.CoreV1().Pods(k8sv1.NamespaceDefault).Delete(context.TODO(), part.Name, *&delOpts)
		case "Service":
			err = kube.kubeClientset.CoreV1().Services(k8sv1.NamespaceDefault).Delete(context.TODO(), part.Name, *&delOpts)
		case "Secret":
			err = kube.kubeClientset.CoreV1().Secrets(k8sv1.NamespaceDefault).Delete(context.TODO(), part.Name, *&delOpts)
		case "StatefulSet":
			//TODO This doesn't delete the pods managed by the service. Weird. Delete service then its pods
			//     It may be the wait method has to wait on the pods to delete, not just the statefulset
			err = kube.kubeAppsClient.StatefulSets(k8sv1.NamespaceDefault).Delete(context.TODO(), part.Name, *&delOpts)
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
func (kube *Kube) WaitDeletedApp(parts []AppPart) {
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
				_, err = kube.kubeClientset.CoreV1().Pods(k8sv1.NamespaceDefault).Get(context.TODO(), part.Name, metav1.GetOptions{})
			case "Service":
				_, err = kube.kubeClientset.CoreV1().Services(k8sv1.NamespaceDefault).Get(context.TODO(), part.Name, metav1.GetOptions{})
			case "Secret":
				_, err = kube.kubeClientset.CoreV1().Secrets(k8sv1.NamespaceDefault).Get(context.TODO(), part.Name, metav1.GetOptions{})
			case "StatefulSet":
				_, err = kube.kubeAppsClient.StatefulSets(k8sv1.NamespaceDefault).Get(context.TODO(), part.Name, metav1.GetOptions{})
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
func (kube *Kube) GetAppStatus(parts []AppPart) (bool, []string) {
	if Debug {
		fmt.Printf("GetAppStatus(): %s\n", parts)
	}
	var status = make([]string, len(parts))
	var succeed = true

	for key, part := range parts {
	nextPart:
		switch part.Kind {
		case "Pod":
			pod, err := kube.kubeClientset.CoreV1().Pods(k8sv1.NamespaceDefault).Get(context.TODO(), part.Name, metav1.GetOptions{})
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
			_, err := kube.kubeClientset.CoreV1().Services(k8sv1.NamespaceDefault).Get(context.TODO(), part.Name, metav1.GetOptions{})
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
			_, err := kube.kubeClientset.CoreV1().Secrets(k8sv1.NamespaceDefault).Get(context.TODO(), part.Name, metav1.GetOptions{})
			if err != nil {
				status[key] = "Failed"
				succeed = false
			} else {
				status[key] = "Succeeded"
			}
		case "StatefulSet":
			statefulset, err := kube.kubeAppsClient.StatefulSets(k8sv1.NamespaceDefault).Get(context.TODO(), part.Name, metav1.GetOptions{})
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
							var selector metav1.ListOptions
							//TODO a real stretch to just pick the name of the first container spec
							selector.LabelSelector = metav1.ListOptions{TypeMeta{Kind: "Pod"}, LabelSelector: statefulset.Spec.Template.Spec.Containers[0].Name}
							podList, err := kube.kubeClientset.Core().Pods(k8sv1.NamespaceDefault).List(selector)
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

// GetPodStatus retrieves a list of pods and their status
func (kube *Kube) GetPodStatus(namespace string) (*k8sv1.PodList, error) {
	pods, err := kube.kubeClientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println(err)
	}
	return pods, err
}
