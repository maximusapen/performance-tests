/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2019, 2020 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package flag

import (
	"github.com/urfave/cli"
)

// StringFlag an IKS CLI flavor of urfave's flag
type StringFlag struct {
	cli.StringFlag

	Require bool
	GroupID string
}

// StringSliceFlag an IKS CLI flavor of urfave's flag
type StringSliceFlag struct {
	cli.StringSliceFlag

	Repeat     bool
	Require    bool
	GroupID    string
	TargetName string
}

// IntFlag an IKS CLI flavor of urfave's flag
type IntFlag struct {
	cli.IntFlag

	Repeat     bool
	Require    bool
	GroupID    string
	TargetName string
}

// UintFlag an IKS CLI flavor of urfave's flag
type UintFlag struct {
	cli.UintFlag

	Repeat     bool
	Require    bool
	GroupID    string
	TargetName string
}
