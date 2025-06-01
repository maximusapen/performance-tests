/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package config

// ScalingUpName holds the name for tests that increase deployment replicas
const ScalingUpName = "pod_scaling_up"

// ScalingDownName holds the name for tests that decrease deployment replicas
const ScalingDownName = "pod_scaling_down"

// Metrics holds command line option to indicate whether metrics should be sent to IBM Cloud monitoring service
var Metrics bool

// Verbose holds command line option to indicate whether additional logging is required
var Verbose bool

// Debug holds command line option to indicate whether detailed logging is required
var Debug bool
