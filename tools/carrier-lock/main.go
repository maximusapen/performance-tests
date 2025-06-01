/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2021, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 *
 * Utility that uses a ConfigMap to enable a locking system
 * It has 5 functions:
 * acquire - Try to acquire the lock, and wait if someone else owns it
 * release - Release the lock - will only work if the parent PID and hostname are the same as the process that acquired the lock
 * force-release - Release the lock, but don't check for hostname or parent PID
 * query - Get the current lock status (outputs json)
 * cleanup - Check if the lock was taken from this host AND the PID is still running. If the PID no longer exists then release the lock
 ******************************************************************************/

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// Namespace defines the namespace to use for the lock
	Namespace string = "default"
	// LockName defines the name of the ConfigMap to use as a lock
	LockName string = "armada-perf-lock"
	// HostFlagName the name of the flag to set in the lock that specifies the host it was created from
	HostFlagName string = "host"
	// StartFlagName the name of the flag to set in the lock that specifies the time the lock started
	StartFlagName string = "start-time"
	// PidFlagName the name of the flag to set in the lock that specifies the PID that owns the lock
	PidFlagName string = "pid"
	// SleepTime - the time to sleep between retries when retrying the lock
	SleepTime time.Duration = time.Duration(60 * time.Second)
)

var kubeconfig string
var maxWaitTime time.Duration
var action string

var kubeClient *kubernetes.Clientset

// initialize - Create the Kube client from the KUBECONFIG
func initialize() {

	ckc, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Failed to build carrier kubeconfig: %v\n", err)
	}

	// create the clientset
	kubeClient, err = kubernetes.NewForConfig(ckc)
	if err != nil {
		log.Fatalf("Failed to create clientset for carrier kubeconfig: %v\n", err)
	}
}

// acquireLock - take the lock, or wait if it is already taken
func acquireLock() {
	acquireExpiryTime := time.Now().Add(maxWaitTime)
	completed := false
	for completed != true {
		cm, err := kubeClient.CoreV1().ConfigMaps(Namespace).Get(context.TODO(), LockName, metav1.GetOptions{})
		// ConfigMap doesn't exist so we can safely create it and take the lock
		if errors.IsNotFound(err) {
			fmt.Printf("%s ConfigMap %s in namespace %s not found, it will be created, and the lock taken\n", time.Now().Format(time.Stamp), LockName, Namespace)
			cm := buildConfigMapData()
			_, err2 := kubeClient.CoreV1().ConfigMaps(Namespace).Create(context.TODO(), &cm, metav1.CreateOptions{})
			// TODO - Retries??
			if err2 != nil {
				log.Fatalf("Unexpected Error occurred Creating the ConfigMap: %v . Lock status is unknown  \n", err)
			}
			completed = true
		} else if err != nil {
			log.Fatalf("Unexpected Error occurred attempting to get the ConfigMap: %v . Will exit without taking lock \n", err)
		} else {
			fmt.Printf("%s ConfigMap %s in namespace %s already exists, owner: %v:%v waiting...\n", time.Now().Format(time.Stamp), LockName, Namespace, cm.Data[HostFlagName], cm.Data[PidFlagName])
			if time.Now().After(acquireExpiryTime) {
				log.Fatalf("Unable to acquire the lock after maxWaitTime: %s . Will exit without taking lock \n", maxWaitTime)
			} else {
				time.Sleep(SleepTime)
			}
		}
	}
}

// releaseLock - Release the lock
func releaseLock(force bool) {
	cm, err := kubeClient.CoreV1().ConfigMaps(Namespace).Get(context.TODO(), LockName, metav1.GetOptions{})
	// ConfigMap doesn't exist so nothing to do
	if errors.IsNotFound(err) {
		fmt.Printf("%s ConfigMap %s in namespace %s not found, nothing to do\n", time.Now().Format(time.Stamp), LockName, Namespace)
	} else if err != nil {
		log.Fatalf("Unexpected Error occurred attempting to get the ConfigMap: %v . Will exit without releasing the lock \n", err)
	} else {
		if !force {
			configMapData := cm.Data
			lockHostName := configMapData[HostFlagName]
			lockOwnerPid, err := strconv.Atoi(configMapData[PidFlagName])
			if err != nil {
				log.Fatalf("Unexpected Error occurred getting owner PID from ConfigMap: %v . Will exit without cleaning up the lock \n", err)
			}
			// Check that the hostname and the PID of the releaser are the same as the acquirer of the lock
			thisHost, _ := os.Hostname()
			if lockHostName != thisHost {
				log.Fatalf("Owner hostname (%s) does not match this hostname (%s), will not release the lock\n", lockHostName, thisHost)
			} else {
				// Check this pid matches the pid that took the lock
				thisPid := os.Getppid()
				if lockOwnerPid != thisPid {
					log.Fatalf("Calling PID (%v) does not match lock owner PID (%v) so will not clean up the lock\n", thisPid, lockOwnerPid)
				}
			}

		}
		err2 := kubeClient.CoreV1().ConfigMaps(Namespace).Delete(context.TODO(), LockName, metav1.DeleteOptions{})
		fmt.Printf("%s Lock Released", time.Now().Format(time.Stamp))
		if err2 != nil {
			log.Fatalf("Unexpected Error occurred attempting to Delete the ConfigMap: %v .\n", err)
		}

	}
}

// queryLock - Query the current lock status
func queryLock() {
	cm, err := kubeClient.CoreV1().ConfigMaps(Namespace).Get(context.TODO(), LockName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		// Doesn't exist, so return empty json
		fmt.Println(string(""))
	} else if err != nil {
		log.Fatalf("Unexpected Error occurred attempting to get the ConfigMap: %v \n", err)
	} else {
		// Print the result as json
		b, err := json.Marshal(cm.Data)
		fmt.Println(string(b))
		if err != nil {
			fmt.Printf("Error occurred marshalling results to json : %v", err)
			return
		}
	}
}

// cleanupLock - Check if the lock was taken on this hostname and if the PID is still running. If it was on this host
// AND the PID is no longer running then release the lock.
func cleanupLock() {
	cm, err := kubeClient.CoreV1().ConfigMaps(Namespace).Get(context.TODO(), LockName, metav1.GetOptions{})
	// ConfigMap doesn't exist so nothing to do
	if errors.IsNotFound(err) {
		fmt.Printf("%s ConfigMap %s in namespace %s not found, nothing to do\n", time.Now().Format(time.Stamp), LockName, Namespace)
	} else if err != nil {
		log.Fatalf("Unexpected Error occurred attempting to get the ConfigMap: %v . Will exit without cleaning up the lock \n", err)
	} else {
		configMapData := cm.Data
		lockHostName := configMapData[HostFlagName]
		lockOwnerPid, err := strconv.Atoi(configMapData[PidFlagName])
		if err != nil {
			log.Fatalf("Unexpected Error occurred getting owner PID from ConfigMap: %v . Will exit without cleaning up the lock \n", err)
		}
		thisHost, _ := os.Hostname()
		if lockHostName == thisHost {
			// Check if Pid is running still
			stillRunning, err := pidExists(lockOwnerPid)
			if err != nil {
				log.Fatalf("Error occurred when checking if PID was running, will not clean up: %v .\n", err)
			}
			if !stillRunning {
				fmt.Printf("%s PID %d on host %s is no longer running so will clean up the lock\n", time.Now().Format(time.Stamp), lockOwnerPid, thisHost)
				releaseLock(true)
			} else {
				fmt.Printf("%s PID %d on host %s is still running so will not clean up the lock\n", time.Now().Format(time.Stamp), lockOwnerPid, thisHost)
			}
		} else {
			fmt.Printf("%s Owner hostname (%s) does not match this hostname (%s), will not cleanup\n", time.Now().Format(time.Stamp), lockHostName, thisHost)
		}
	}
}

// pidExists - Check if the PID that took the lock is still running
func pidExists(pid int) (bool, error) {
	if pid <= 0 {
		return false, fmt.Errorf("invalid pid %v", pid)
	}
	proc, err := os.FindProcess(int(pid))
	if err != nil {
		return false, err
	}
	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true, nil
	}
	if err.Error() == "os: process already finished" {
		return false, nil
	}
	errno, ok := err.(syscall.Errno)
	if !ok {
		return false, err
	}
	switch errno {
	case syscall.ESRCH:
		return false, nil
	case syscall.EPERM:
		return true, nil
	}
	return false, err
}

// buildConfigMapData - Build the data that will be stored in the ConfigMap
func buildConfigMapData() corev1.ConfigMap {
	configMapData := make(map[string]string, 0)

	configMapData[HostFlagName], _ = os.Hostname()
	configMapData[StartFlagName] = time.Now().String()
	configMapData[PidFlagName] = strconv.Itoa(os.Getppid())
	configMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      LockName,
			Namespace: Namespace,
		},
		Data: configMapData,
	}
	return configMap
}

func main() {

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Kubeconfig for the cluster to create the lock")
	flag.DurationVar(&maxWaitTime, "max-wait-time", (120 * time.Minute), "The maximum duration to wait for the lock")
	flag.StringVar(&action, "action", "", "The Action to take - either acquire, release, force-release, query or cleanup")

	flag.Parse()
	if len(kubeconfig) == 0 {
		log.Fatalln("Please specify location of the Kubeconfig file")
	}

	initialize()

	if action == "query" {
		queryLock()
	} else if action == "acquire" {
		acquireLock()
	} else if action == "release" {
		releaseLock(false)
	} else if action == "force-release" {
		releaseLock(true)
	} else if action == "cleanup" {
		cleanupLock()
	} else {
		log.Fatalf("Unknown action flag specified: %v . Value must be acquire, query, release, force-release or cleanup.\n", action)
	}
}
