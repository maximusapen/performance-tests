/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

// Copyright 2015 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"sync"

	"github.com/spf13/cobra"
	"go.etcd.io/etcd/client/pkg/v3/transport"
)

// RootCmd ...
var RootCmd = &cobra.Command{
	Use:   "etcd-pattern",
	Short: "Etcd key/value generator",
	Long:  "A tool for generating etcd key/value pairs based on a pattern",
}

var (
	endpoints []string

	totalConns   uint
	totalClients uint
	results      chan result

	clientTimeout int

	wg               sync.WaitGroup
	tls              transport.TLSInfo
	userNamePassword string

	csvFile      string
	csvDir       string
	fileComments string

	verbose bool

	cpuProfPath string
	memProfPath string
)

func init() {
	RootCmd.PersistentFlags().StringSliceVar(&endpoints, "endpoints", []string{"127.0.0.1:2379"}, "gRPC endpoints")

	RootCmd.PersistentFlags().UintVar(&totalConns, "conns", 1, "Total number of gRPC connections")
	RootCmd.PersistentFlags().UintVar(&totalClients, "clients", 1, "Total number of gRPC clients")

	RootCmd.PersistentFlags().IntVar(&clientTimeout, "client-timeout", 10, "The timeout (in seconds) used for etcd interactions (default 10 seconds)")

	RootCmd.PersistentFlags().StringVar(&tls.CertFile, "cert", "", "identify HTTPS client using this SSL certificate file")
	RootCmd.PersistentFlags().StringVar(&tls.KeyFile, "key", "", "identify HTTPS client using this SSL key file")
	RootCmd.PersistentFlags().StringVar(&tls.TrustedCAFile, "cacert", "", "verify certificates of HTTPS-enabled servers using this CA bundle")

	RootCmd.PersistentFlags().StringVar(&csvFile, "csv-file", "", "File to write csv results")
	RootCmd.PersistentFlags().StringVar(&csvDir, "csv-dir", "", "Directory where csv file will be placed. $HOSTNAME is appended as an additional directory")
	RootCmd.PersistentFlags().StringVar(&fileComments, "file-comment", "", "Comment to add to results written to file")

	RootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output")
	RootCmd.PersistentFlags().StringVar(&cpuProfPath, "cpuprofile", "", "the path of file for storing cpu profile result")
	RootCmd.PersistentFlags().StringVar(&memProfPath, "memprofile", "", "the path of file for storing heap profile result")
	RootCmd.PersistentFlags().StringVar(&userNamePassword, "user", "", "username:password for authentication")

	// If a new parmeter isn't added to the exclude list then the output csv file column layout will change
	addToFileExclude("cacert")
	addToFileExclude("cert")
	addToFileExclude("csv-dir")
	addToFileExclude("csv-file")
	addToFileExclude("help")
	addToFileExclude("key")
	addToFileExclude("cpuprofile")
	addToFileExclude("memprofile")
	addToFileExclude("user")
}

func setupProfiling() {
	if cpuProfPath != "" {
		log.Print("Starting profile")
		f, err := os.Create(cpuProfPath)
		if err != nil {
			log.Fatalf("Failed to create a file for storing cpu profile result: %v", err)
		}

		err = pprof.StartCPUProfile(f)
		if err != nil {
			log.Fatalf("Failed to start cpu profile: %v", err)
		}
		defer pprof.StopCPUProfile()
	}

	if memProfPath != "" {
		f, err := os.Create(memProfPath)
		if err != nil {
			log.Fatalf("Failed to create a file for storing heap profile result: %v", err)
		}

		defer func() {
			err := pprof.WriteHeapProfile(f)
			if err != nil {
				log.Printf("Failed to write heap profile result: %v", err)
				// can do nothing for handling the error
			}
		}()
	}
}

func setupCsvFile() {
	if len(csvDir) > 0 {
		hostname := os.Getenv("HOSTNAME")
		fmt.Println("Hostname ", hostname)
		if len(hostname) > 0 {
			csvDir = csvDir + "/" + hostname
		}
		fmt.Println("mkdir ", csvDir)
		err := os.MkdirAll(csvDir, 0755)
		if err != nil {
			fmt.Println("Failed to make directory for csv file", csvDir, err)
			os.Exit(1)
		}
		csvFile = csvDir + "/" + csvFile
	}
}
