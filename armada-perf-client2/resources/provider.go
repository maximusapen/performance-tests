/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2019, 2020 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package resources

import (
	"strings"

	"github.ibm.com/alchemy-containers/armada-model/model"
)

// IsClassicProvider is a helper function that sets up the provider string correctly before calling armada-model's
// IsClassicProvider function
func IsClassicProvider(provider string) bool {
	internalName := model.MapProviderToInternal(strings.ToLower(provider))
	return model.IsClassicProvider(&internalName)
}

// IsVPCProvider is a helper function that sets up the provider string correctly before calling armada-model's
// IsVPCProvider function
func IsVPCProvider(provider string) bool {
	internalName := model.MapProviderToInternal(strings.ToLower(provider))
	return model.IsVPCProvider(&internalName)
}

// IsSatelliteProvider is a helper function that sets up the provider string correctly before calling armada-model's
// IsSatelliteProvider function
func IsSatelliteProvider(provider string) bool {
	internalName := model.MapProviderToInternal(strings.ToLower(provider))
	return model.IsMultishiftProvider(&internalName)
}
