
package cmdcluster

import (
	"fmt"
	"strings"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
	"gopkg.in/yaml.v2"

	apiModelV1 "github.ibm.com/alchemy-containers/armada-model/model/api/json/v1"
)

const (
	addonIstio      = "istio"
	addonOdf        = "openshift-data-foundation"
	addonVpcBlock   = "vpc-block-csi-driver"
	addonVpcFile    = "vpc-file-csi-driver"
	addonAutoscaler = "cluster-autoscaler"

	addonIstioDescription      = "Managed Istio service mesh add-on"
	addonOdfDescription        = "Openshift Container Storage (OCS) / Openshift Data Foundation (ODF) on IBM Cloud add-on"
	addonVpcBlockDescription   = "IBM Cloud VPC Block Storage CSI Driver add-on"
	addonVpcFileDescription    = "IBM Cloud VPC File Storage CSI Driver add-on"
	addonAutoscalerDescription = "IBM Cloud Node Autoscaler add-on"

	addonVersionFlagName = "version"
	paramsFlagName       = "param"
)

type optionsConfigMap struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   cmMetadata        `yaml:"metadata"`
	Data       map[string]string `yaml:"data"`
}

type cmMetadata struct {
	Name        string      `yaml:"name"`
	Namespace   string      `yaml:"namespace"`
	Labels      interface{} `yaml:"labels"`
	Annotations interface{} `yaml:"annotations"`
}

// AddonSubCommandRegister registers `cluster addon` commands
func AddonSubCommandRegister() cli.Command {
	addonVersionFlag := cli.StringFlag{
		Name:  addonVersionFlagName,
		Usage: "Optional: Specify the version of the add-on to install. If no version is specified, the default version is installed.",
	}

	paramsFlag := cli.StringSliceFlag{
		Name:  paramsFlagName,
		Usage: "Specify installation options for the add-on. If no parameters are specified, the default values are used. Review the available options with the 'ibmcloud ks cluster addon options' command.",
	}

	// generates an enable command for generic addon $name with the given description
	genericEnableCommand := func(name, description string, params bool) cli.Command {
		flags := []cli.Flag{
			models.RequiredClusterFlag,
			addonVersionFlag,
		}

		if params {
			flags = append(flags, paramsFlag)
		}

		return cli.Command{
			Name:        name,
			Description: description,
			Usage:       description,
			Action:      enableGenericAddon(name),
			Flags:       flags,
		}
	}

	// generates a disable command for generic addon $name with the given description
	genericDisableCommand := func(name, description string) cli.Command {
		return cli.Command{
			Name:        name,
			Description: description,
			Usage:       description,
			Action:      disableGenericAddon(name),
			Flags: []cli.Flag{
				models.RequiredClusterFlag,
			},
		}
	}

	return cli.Command{
		Name:        models.NamespaceAddon,
		Description: "View, enable, and disable cluster add-ons.",
		Usage:       "View, enable, and disable cluster add-ons.",
		Subcommands: []cli.Command{
			{
				Name:        models.CmdEnable,
				Description: "Enable cluster add-ons.",
				Usage:       "Enable cluster add-ons.",
				Subcommands: cli.Commands{
					genericEnableCommand(addonIstio, addonIstioDescription, false),
					genericEnableCommand(addonOdf, addonOdfDescription, true),
					genericEnableCommand(addonVpcBlock, addonVpcBlockDescription, false),
					genericEnableCommand(addonVpcFile, addonVpcFileDescription, false),
					genericEnableCommand(addonAutoscaler, addonAutoscalerDescription, false),
				},
			},
			{
				Name:        models.CmdDisable,
				Description: "Disable cluster add-ons.",
				Usage:       "Disable cluster add-ons.",
				Subcommands: cli.Commands{
					genericDisableCommand(addonIstio, addonIstioDescription),
					genericDisableCommand(addonOdf, addonOdfDescription),
					genericDisableCommand(addonVpcBlock, addonVpcBlockDescription),
					genericDisableCommand(addonVpcFile, addonVpcFileDescription),
					genericDisableCommand(addonAutoscaler, addonAutoscalerDescription),
				},
			},
			{
				Name:        models.CmdList,
				Description: "List enabled add-ons.",
				Usage:       "List enabled add-ons.",
				Action:      listAddons,
				Flags: []cli.Flag{
					models.RequiredClusterFlag,
					models.JSONOutFlag,
				},
			},
		},
	}
}

func enableGenericAddon(addonName string) cli.ActionFunc {
	return func(c *cli.Context) error {
		clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
		if err != nil {
			return err
		}

		version := c.String(addonVersionFlagName)
		params := c.StringSlice(paramsFlagName)

		options, optionsTemplate, err := checkOptionsOnEnable(c, addonName, version, params)
		if err != nil {
			return err
		}

		if version == "" {
			fmt.Fprintf(c.App.Writer, "Enabling add-on '%s' for cluster '%s'.\n", addonName, clusterNameOrID)
		} else {
			fmt.Fprintf(c.App.Writer, "Enabling add-on '%s %s' for cluster '%s'.\n", addonName, version, clusterNameOrID)
		}

		addon := apiModelV1.ClusterAddon{
			AddonCommon: apiModelV1.AddonCommon{
				Name:    addonName,
				Version: version,
			},
		}

		if optionsTemplate.Content != "" && len(options.Data) > 0 {
			opt, err := yaml.Marshal(options)
			if err != nil {
				return fmt.Errorf("issue parsing the options YAML. Wait a few moments, then try again")
			}
			addon.Options = string(opt)
			fmt.Fprintln(c.App.Writer, "Using installation options...")
			err = printOptions(c, addon.Name, addon.Options, false)
			if err != nil {
				return err
			}
		}
		if optionsTemplate.Content != "" && len(options.Data) == 0 {
			fmt.Fprintf(c.App.Writer, "Using default installation options...")
			err = printOptions(c, addon.Name, optionsTemplate.Content, false)
			if err != nil {
				return err
			}
		}

		endpoint := resources.GetArmadaEndpoint(c)
		resp, err := endpoint.EnableClusterAddons(clusterNameOrID, addon)
		if err != nil {
			return err
		}

		// Need to enable any dependencies?
		if len(resp.MissingDeps) > 0 {
			for _, dep := range resp.MissingDeps {
				fmt.Fprintf(c.App.Writer, "Also enabling addon '%s %s' which is required to enable '%s'.\n", dep.Name, dep.Version, addon.Name)
			}

			_, err = endpoint.EnableClusterAddons(clusterNameOrID, append(resp.MissingDeps, addon)...)
			if err != nil {
				return err
			}
		}

		fmt.Fprintf(c.App.Writer, "Request to enable addon '%s' for cluster '%s' successful.\n", addonName, clusterNameOrID)
		return nil
	}
}

func disableGenericAddon(addonName string) cli.ActionFunc {
	return func(c *cli.Context) error {
		clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
		if err != nil {
			return err
		}

		addon := apiModelV1.ClusterAddon{
			AddonCommon: apiModelV1.AddonCommon{
				Name: addonName,
			},
		}

		endpoint := resources.GetArmadaEndpoint(c)
		resp, err := endpoint.DisableClusterAddons(clusterNameOrID, addon)
		if err != nil {
			return err
		}

		if len(resp.Orphaned) > 0 {
			addons := []apiModelV1.ClusterAddon{addon}
			for name := range resp.Orphaned {
				addons = append(addons, apiModelV1.ClusterAddon{
					AddonCommon: apiModelV1.AddonCommon{
						Name: name,
					},
				})
				fmt.Fprintf(c.App.Writer, "Also disabling orphaned addon '%s'.\n", name)
			}
			_, err := endpoint.DisableClusterAddons(clusterNameOrID, addons...)
			if err != nil {
				return err
			}
		}

		fmt.Fprintf(c.App.Writer, "Request to disable addon '%s' for cluster '%s' successful.\n", addonName, clusterNameOrID)
		return nil
	}
}

func listAddons(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	endpoint := resources.GetArmadaEndpoint(c)
	addons, err := endpoint.ListClusterAddons(clusterNameOrID)
	if err != nil {
		return err
	}

	jsonoutput := c.Bool(models.JSONOutFlagName)
	if jsonoutput {
		cliutils.WriteJSON(c, addons)
		return nil
	}

	fmt.Fprintln(c.App.Writer)
	fmt.Fprintln(c.App.Writer, "Name\tVersion\tState\tStatus")
	fmt.Fprintln(c.App.Writer, "----\t-------\t-----\t------")

	for _, addon := range addons {
		fmt.Fprintf(c.App.Writer,
			"%s\t%s\t%s\t%s\n",
			addon.Name,
			addon.Version,
			addon.HealthState,
			addon.HealthStatus,
		)
	}

	return nil
}

func printOptions(c *cli.Context, addonName string, options string, defaultOptions bool) error {
	fmt.Fprintln(c.App.Writer)
	fmt.Fprintln(c.App.Writer, "Add-on Options\n--------------")

	cm, err := unmarshalCMYAML(c, options, addonName)
	if err != nil {
		return err
	}

	for key, option := range cm.Data {
		fmt.Fprintf(c.App.Writer, "%s: %s\n", key, option)
	}
	return err
}

type errFetchAddonOptions struct {
	message string
}

func (e *errFetchAddonOptions) Error() string {
	return e.message
}

func getAddonOptionsTemplate(c *cli.Context, addonName string, version string) (apiModelV1.AddonOptionsTemplate, error) {
	endpoint := resources.GetArmadaEndpoint(c)
	addons, err := endpoint.GetAddonVersions()
	if err != nil {
		return apiModelV1.AddonOptionsTemplate{}, err
	}

	for _, addon := range addons {
		if addon.Name == addonName {
			optionsVersion := version
			if optionsVersion == "" {
				optionsVersion = addon.TargetVersion
			}
			if optionsVersion == addon.Version {
				if addon.OptionsTemplate.Content == "" {
					e := &errFetchAddonOptions{
						message: fmt.Sprintf("No add-on installation options are available for %s", addonName),
					}
					return apiModelV1.AddonOptionsTemplate{}, e
				}
				return addon.OptionsTemplate, nil
			}
		}
	}
	e := &errFetchAddonOptions{
		message: fmt.Sprintf("Unable to find add-on '%s'. Check the add-on name and try again.", addonName),
	}
	return apiModelV1.AddonOptionsTemplate{}, e
}

func unmarshalCMYAML(c *cli.Context, options string, addonName string) (optionsConfigMap, error) {
	cm := optionsConfigMap{}
	err := yaml.Unmarshal([]byte(options), &cm)
	if err != nil {
		fmt.Fprintf(c.App.Writer, "Error parsing addon options as ConfigMap: %s", err)
		return cm, fmt.Errorf("error parsing the options for %s", addonName)
	}
	return cm, nil
}

func checkOptionsOnEnable(c *cli.Context, addonName string, version string, params []string) (optionsConfigMap, apiModelV1.AddonOptionsTemplate, error) {
	options := optionsConfigMap{}

	optionsTemplate, err := getAddonOptionsTemplate(c, addonName, version)
	if err != nil {
		if _, isFetchErr := err.(*errFetchAddonOptions); err != nil && !isFetchErr {
			return options, optionsTemplate, err
		}
	}

	if optionsTemplate.Content == "" && len(params) != 0 {
		return options, optionsTemplate, fmt.Errorf("Passing installation options is not valid for %s", addonName)
	}

	if optionsTemplate.Content == "" {
		return options, optionsTemplate, nil
	}

	options, err = unmarshalCMYAML(c, optionsTemplate.Content, addonName)
	if err != nil {
		return options, optionsTemplate, err
	}
	for _, p := range params {
		option := strings.Split(p, "=")
		if len(option) < 2 {
			return options, optionsTemplate, fmt.Errorf("invalid parameter format")
		}
		if _, ok := options.Data[option[0]]; !ok {
			return options, optionsTemplate, fmt.Errorf("Invalid option %s for %s", option[0], addonName)
		}
		options.Data[option[0]] = option[1]
	}
	return options, optionsTemplate, nil
}
