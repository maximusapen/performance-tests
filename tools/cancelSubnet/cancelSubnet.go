/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2019, 2021 All Rights Reserved.
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

	"github.com/softlayer/softlayer-go/filter"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
)

// encyptionKey is the key used to decrypt sensitive data from the configuration file(s).
// It's value is baked in the executable at build time
var encryptionKey string

func main() {
	const slVlanMask = "id,name,vlanNumber,secondarySubnets,secondarySubnets.displayLabel,secondarySubnets.billingItem.id"

	const iaasAccount = 1186049
	const privateVlanID = 2263901
	const publicVlanID = 2263903

	var ourPerformanceVlanID int

	var slTrue = true
	var slReason = "No longer needed"

	subnetDisplayLabel := flag.String("subnet", "", "Display label of subnet to be cancelled")
	privateVLAN := flag.Bool("private", false, "Private Performance VLAN")
	publicVLAN := flag.Bool("public", false, "Public Performance VLAN")

	flag.Parse()

	if *subnetDisplayLabel == "" {
		log.Fatalln("Please specify subnet display label to be cancelled")
	}

	// Ensure one of --private --public has been specified
	if *privateVLAN == *publicVLAN {
		log.Fatalln("Please specify --private (for private VLAN) or --public (for public VLAN")
	}

	if *publicVLAN {
		ourPerformanceVlanID = publicVlanID
	} else {
		ourPerformanceVlanID = privateVlanID
	}

	// Get Softlayer credentials from environment
	iaasUsername := os.Getenv(fmt.Sprintf("PROD_GLOBAL_ARMPERF_SOFTLAYER_%d_USERID", iaasAccount))
	if len(iaasUsername) == 0 {
		log.Fatalf("IAAS username not provided. Check environment.")
	}
	iaasAPIKey := os.Getenv(fmt.Sprintf("PROD_GLOBAL_ARMPERF_SOFTLAYER_%d_APIKEY", iaasAccount)) // pragma: allowlist secret
	if len(iaasAPIKey) == 0 {
		log.Fatalf("IAAS API Key not provided. Check environment.")
	}

	log.Println("Opening session with Softlayer")
	sess := session.New(
		iaasUsername,
		iaasAPIKey,
		"https://api.softlayer.com/rest/v3",
		"30s")

	log.Println("Getting data from Softlayer")

	// Filter on our VLAN id
	filter := filter.New(
		filter.Path("networkVlans.id").Eq(ourPerformanceVlanID),
	).Build()

	vlans, err := services.GetAccountService(sess).Mask(slVlanMask).Filter(filter).GetNetworkVlans()
	if err != nil {
		log.Fatalf("Unable to get VLANs from Softlayer - %s\n", err.Error())
	}

	for _, v := range vlans {
		// Should only have one VLAN, and it should be ours (we used a filter), but let's be sure
		if *v.Id == ourPerformanceVlanID {
			log.Printf("Processing VLAN: \"%s\" - %d\n", *v.Name, *v.VlanNumber)
			for _, s := range v.SecondarySubnets {
				if *s.DisplayLabel == *subnetDisplayLabel {
					if s.BillingItem != nil {
						log.Printf("About to cancel subnet \"%s\". Are you sure ? (Enter yes to confirm)", *s.DisplayLabel)
						var input string
						fmt.Scanln(&input)

						if input == "yes" {
							bi := services.GetBillingItemService(sess).Id(*s.BillingItem.Id)

							log.Printf("Cancelling \"%s\"\n", *s.DisplayLabel)
							success, err := bi.CancelItem(&slTrue, &slTrue, &slReason, nil)
							if err != nil {
								log.Fatalf("Unable to cancel subnet - %s\n", err.Error())
							} else if !success {
								log.Fatalln("Cancel failed without error")
							} else {
								log.Printf("\"%s\" successfully cancelled.\n", *s.DisplayLabel)
							}
						}
					} else {
						log.Printf("No billing item found for \"%s\"; skipping.\n", *s.DisplayLabel)
					}
					os.Exit(0)
				}
			}

			log.Printf("Subnet \"%s\" not found.\n", *subnetDisplayLabel)
		} else {
			log.Fatalln("Wrong VLAN detected. Its a bug. Shouldn't happen !")
		}
	}
}
