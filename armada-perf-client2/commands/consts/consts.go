/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2019, 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package consts

const (
	// DefaultWorkerNum defines the default number of workers
	DefaultWorkerNum = 1

	// DefaultClusterQuantity define the default number of clusters to create
	DefaultClusterQuantity = 1

	// DefaultThreads defines the default number of threads to be used for parallel requests
	DefaultThreads = 1

	// TimeoutEnvVar is an environment variable used to override the default HTTP request timeout
	TimeoutEnvVar = "APC2_REQUEST_TIMEOUT"
)
