/*******************************************************************************
 * I
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2020, 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdlocation

import "github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"

func getLocationConfig(name string) cliutils.TomlSatelliteInfrastructureLocation {
	satelliteConfig := cliutils.GetInfrastructureConfig().Satellite
	if satelliteConfig != nil {
		locationConfig := cliutils.GetInfrastructureConfig().Satellite.Locations

		lc, ok := locationConfig[name]
		if !ok {
			return cliutils.GetDefaultLocation()
		}
		return lc
	}

	return cliutils.TomlSatelliteInfrastructureLocation{}
}
