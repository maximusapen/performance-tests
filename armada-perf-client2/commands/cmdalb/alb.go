
// Package cmdalb registers and executes 'apc2 alb ...' commands.
package cmdalb

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/urfave/cli"
	ingressModel "github.ibm.com/alchemy-containers/armada-dns-model/v2/ingress"

	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

const (
	albIDFlagName      = "alb-id"
	albVersionFlagName = "version"

	ingressBuildName     = "ingress"
	ingressAuthBuildName = "ingress-auth"
)

// RegisterCommands registers the "versions" command
func RegisterCommands() []cli.Command {
	albVersionFlag := cli.StringFlag{
		Name:  albVersionFlagName,
		Usage: "Specify the ALB image version. To see supported image versions, run 'apc2 alb versions'.",
	}

	return []cli.Command{
		{
			Name:        models.NamespaceALB,
			Description: "View and configure an Ingress application load balancer (ALB).",
			Category:    models.ClusterComponentsCategory,
			Subcommands: []cli.Command{
				{
					Name:        models.CmdVersions,
					Description: "List Ingress ALB image versions that are available.",
					Flags: []cli.Flag{
						models.JSONOutFlag,
					},
					Action: albVersions,
				},
				{
					Name:        models.CmdList,
					Description: "List all Ingress ALB IDs in a cluster and whether ALB pods are at the latest version.",
					Flags: []cli.Flag{
						models.RequiredClusterFlag,
						models.JSONOutFlag,
					},
					Action: albList,
				},
				{
					Name:        models.CmdUpdate,
					Description: "Force a one-time update of the pods for individual or all ALBs in the cluster.",
					Flags: []cli.Flag{
						models.RequiredClusterFlag,
						albVersionFlag,

						cli.StringSliceFlag{
							Name:  albIDFlagName,
							Usage: "To update a specific ALB, specify the ALB ID. To update more than one ALB, specify one ALB ID in each flag, such as '--alb-id ID_1 --alb-id ID_2'. To update all ALBs, do not include this flag.",
						},
						models.JSONOutFlag,
					},
					Action: albUpdate,
				},
			},
		},
	}
}

func albVersions(c *cli.Context) error {
	endpoint := resources.GetArmadaV2Endpoint(c)
	albImages, err := endpoint.GetALBImages()
	if err != nil {
		return err
	}

	jsonoutput := c.Bool(models.JSONOutFlagName)
	if jsonoutput {
		cliutils.WriteJSON(c, albImages)
		return nil
	}

	for _, version := range albImages.SupportedK8sVersions {
		if albImages.DefaultK8sVersion == version {
			version = version + " (default)"
		}
		fmt.Fprintf(c.App.Writer, "%s\n", version)
	}

	return nil
}

func albList(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	endpoint := resources.GetArmadaV2Endpoint(c)
	rawJSON, classicClusterALB, vpcClusterALB, err := endpoint.GetClusterALBs(clusterNameOrID)
	if err != nil {
		return err
	}

	jsonoutput := c.Bool(models.JSONOutFlagName)
	if jsonoutput {
		cliutils.WriteJSON(c, rawJSON)
		return nil
	}

	// If Vlan ID is empty this is a VPC cluster response
	if len(classicClusterALB.ALBs) >= 1 && classicClusterALB.ALBs[0].VlanID == "" {
		albs := vpcClusterALB

		fmt.Fprintln(c.App.Writer)
		fmt.Fprintln(c.App.Writer, "ALB ID\tEnabled\tStatus\tType\tLoad Balancer Hostname\tZone\tBuild")
		fmt.Fprintln(c.App.Writer, "------\t-------\t------\t----\t----------------------\t----\t-----")

		for _, alb := range albs.ALBs {
			fmt.Fprintf(c.App.Writer,
				"%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				alb.AlbID,
				strconv.FormatBool(alb.Enable),
				alb.State,
				alb.AlbType,
				alb.LoadBalancerHostname,
				alb.Zone,
				getALBBuildString(alb.ALBBuild, alb.AuthBuild),
			)
		}
	} else {
		// Classic cluster output
		albs := classicClusterALB

		fmt.Fprintln(c.App.Writer)
		fmt.Fprintln(c.App.Writer, "ALB ID\tEnabled\tStatus\tType\tALB IP\tZone\tBuild\tALB VLAN ID\tNLB Version")
		fmt.Fprintln(c.App.Writer, "------\t-------\t------\t----\t------\t----\t-----\t-----------\t-----------")

		for _, alb := range albs.ALBs {
			fmt.Fprintf(c.App.Writer,
				"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				alb.AlbID,
				strconv.FormatBool(alb.Enable),
				alb.State,
				alb.AlbType,
				alb.AlbIP,
				alb.Zone,
				getALBBuildString(alb.ALBBuild, alb.AuthBuild),
				alb.VlanID,
				alb.NLBVersion)
		}
	}

	return nil
}

func albUpdate(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	albs := c.StringSlice(albIDFlagName)
	albVersion := c.String(albVersionFlagName)

	config := ingressModel.V2UpdateALB{
		ClusterID: clusterNameOrID,
		ALBBuild:  albVersion,
		ALBList:   albs,
	}

	selector := "all ALBs"
	if len(albs) > 0 {
		selector = strings.Join(albs[:], ", ")
	}

	version := "latest"
	if len(albVersion) > 0 {
		version = albVersion
	}

	fmt.Fprintf(c.App.Writer, "Updating ALB pods for '%s' to version '%s' in cluster '%s'\n", selector, version, clusterNameOrID)

	endpoint := resources.GetArmadaV2Endpoint(c)
	if err := endpoint.UpdateALB(config); err != nil {
		return err
	}
	return nil
}

func getALBBuildString(albBuild, authBuild string) string {
	if authBuild != "" {
		return fmt.Sprintf("%s:%s/%s:%s", ingressBuildName, albBuild, ingressAuthBuildName, authBuild)
	}
	return fmt.Sprintf("%s:%s", ingressBuildName, albBuild)
}
