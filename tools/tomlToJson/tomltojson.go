/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2021 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/BurntSushi/toml"
)

func main() {
	flag.Parse()
	for _, f := range flag.Args() {
		// Parse toml file
		var dt interface{}
		_, err := toml.DecodeFile(f, &dt)
		if err != nil {
			log.Fatalf("Error decoding toml file : %s", err.Error())
		}

		// Convert to json
		j := json.NewEncoder(os.Stdout)
		err = j.Encode(dt)
		if err != nil {
			log.Fatalf("Error encoding json : %s", err.Error())
		}
	}
}
