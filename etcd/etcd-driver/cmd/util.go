/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2021 All Rights Reserved.
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
	"crypto/rand"
	cryptoTLS "crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// Endpoints are distributed via round-robin to connections.
// Connections are distributed via round-robin to clients.
// Generally a test will create a thread per client to pull from the request queue.
// - In the case of 'pattern' there are multiple tests, each of which creates
//   a thread per client.

var (
	// dialTotal counts the number of mustCreateConn calls so that endpoint
	// connections can be handed out in round-robin order
	dialTotal int
)

func mustCreateConn(allEndpoints bool) *clientv3.Client {
	var theEndpoints []string
	if !allEndpoints {
		endpoint := endpoints[dialTotal%len(endpoints)]
		dialTotal++
		theEndpoints = []string{endpoint}
	} else {
		theEndpoints = endpoints
	}
	cfg := clientv3.Config{Endpoints: theEndpoints}

	if !tls.Empty() && len(userNamePassword) == 0 {
		cfgtls, err := tls.ClientConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "bad tls config: %v\n", err)
			os.Exit(1)
		}
		cfg.TLS = cfgtls
	} else if len(userNamePassword) > 0 {
		split := strings.Split(userNamePassword, ":")
		if len(split) != 2 {
			fmt.Fprintf(os.Stderr, "bad username:password parameter\n")
			os.Exit(1)
		}
		cfg.Username = split[0]
		cfg.Password = split[1] // pragma: allowlist secret

		caPath := tls.TrustedCAFile
		if caPath != "" {
			// #nosec G304
			caCert, err := ioutil.ReadFile(caPath)
			if err != nil {
				log.Fatalf("Error occurred reading file: %s", err.Error())
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			tlsConfig := &cryptoTLS.Config{
				RootCAs: caCertPool,
			}
			cfg.TLS = tlsConfig
		}
	}

	client, err := clientv3.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial error: %v\n", err)
		os.Exit(1)
	}
	return client
}

func mustCreateClients(totalClients, totalConns uint) []*clientv3.Client {
	conns := make([]*clientv3.Client, totalConns)
	for i := range conns {
		conns[i] = mustCreateConn(false)
	}

	clients := make([]*clientv3.Client, totalClients)
	for i := range clients {
		clients[i] = conns[i%int(totalConns)]
	}
	return clients
}

func mustRandBytes(n int) []byte {
	rb := make([]byte, n)
	_, err := rand.Read(rb)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate value: %v\n", err)
		os.Exit(1)
	}
	return rb
}
