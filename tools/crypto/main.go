/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2020 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.ibm.com/alchemy-containers/armada-performance/tools/crypto/utils"
)

func main() {
	var generate, encrypt, decrypt bool

	var key string
	var data string

	flag.BoolVar(&generate, "generate", false, "Generate and use a new encryption key")
	flag.BoolVar(&encrypt, "encrypt", false, "Encrypt input data")
	flag.BoolVar(&decrypt, "decrypt", false, "Decrypt input data")
	flag.StringVar(&key, "key", "", "Existing encryption key to be used")

	flag.Parse()

	// Data to encrypt/decrypt
	if len(flag.Args()) > 0 {
		data = flag.Args()[0]
	}

	if len(key) != 0 {
		os.Setenv(utils.KeyEnvVar, key)
	}

	if generate {
		key, err := utils.GenerateKey()
		if err != nil {
			log.Fatalf("%s\n", err.Error())
		}

		fmt.Printf("Cipher Key: %s\n", key)
	}

	if encrypt {
		ciphertext, err := utils.Encrypt(data)
		if err != nil {
			log.Fatalf("%s\n", err.Error())
		}

		fmt.Printf("%s\n", ciphertext)

		data = ciphertext
	}

	if decrypt {
		plaintext, err := utils.Decrypt(data)
		if err != nil {
			log.Fatalf("%s\n", err.Error())
		}

		fmt.Printf("%s\n", plaintext)
	}
}
