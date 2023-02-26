// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package pluginmanager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aunum/log"
	"github.com/stretchr/testify/assert"

	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginsupplier"
)

var expectedDiscoveredContextPlugins = []discovery.Discovered{
	{
		Name:               "cluster",
		RecommendedVersion: "v1.6.0",
		Scope:              common.PluginScopeContext,
		ContextName:        "mgmt",
		Target:             configtypes.TargetK8s,
	},
	{
		Name:               "cluster",
		RecommendedVersion: "v0.2.0",
		Scope:              common.PluginScopeContext,
		ContextName:        "tmc-fake",
		Target:             configtypes.TargetTMC,
	},
	{
		Name:               "management-cluster",
		RecommendedVersion: "v0.2.0",
		Scope:              common.PluginScopeContext,
		ContextName:        "tmc-fake",
		Target:             configtypes.TargetTMC,
	},
}
var expectedDiscoveredStandalonePlugins = []discovery.Discovered{
	{
		Name:               "login",
		RecommendedVersion: "v0.2.0",
		Scope:              common.PluginScopeStandalone,
		ContextName:        "",
		Target:             configtypes.TargetUnknown,
	},
	{
		Name:               "management-cluster",
		RecommendedVersion: "v1.6.0",
		Scope:              common.PluginScopeStandalone,
		ContextName:        "",
		Target:             configtypes.TargetK8s,
	},
	{
		Name:               "myplugin",
		RecommendedVersion: "v1.6.0",
		Scope:              common.PluginScopeStandalone,
		ContextName:        "",
		Target:             configtypes.TargetK8s,
	},
	{
		Name:               "myplugin",
		RecommendedVersion: "v0.2.0",
		Scope:              common.PluginScopeStandalone,
		ContextName:        "",
		Target:             configtypes.TargetTMC,
	},
}

func Test_DiscoverPlugins(t *testing.T) {
	assertions := assert.New(t)

	defer setupLocalDistoForTesting()()

	serverPlugins, standalonePlugins := DiscoverPlugins()
	assertions.Equal(len(expectedDiscoveredContextPlugins), len(serverPlugins))
	assertions.Equal(len(expectedDiscoveredStandalonePlugins), len(standalonePlugins))

	discoveredPlugins := append(serverPlugins, standalonePlugins...)
	expectedDiscoveredPlugins := append(expectedDiscoveredContextPlugins, expectedDiscoveredStandalonePlugins...)

	for i := 0; i < len(expectedDiscoveredPlugins); i++ {
		p := findDiscoveredPlugin(discoveredPlugins, expectedDiscoveredPlugins[i].Name, expectedDiscoveredPlugins[i].Target)
		assertions.NotNil(p)
		assertions.Equal(expectedDiscoveredPlugins[i].Name, p.Name)
		assertions.Equal(expectedDiscoveredPlugins[i].RecommendedVersion, p.RecommendedVersion)
		assertions.Equal(expectedDiscoveredPlugins[i].Target, p.Target)
	}

	err := configlib.SetFeature("global", "context-target-v2", "false")
	assertions.Nil(err)

	serverPlugins, standalonePlugins = DiscoverPlugins()
	assertions.Equal(1, len(serverPlugins))
	assertions.Equal(len(expectedDiscoveredStandalonePlugins), len(standalonePlugins))
}

func Test_InstallPlugin_InstalledPlugins(t *testing.T) {
	assertions := assert.New(t)

	defer setupLocalDistoForTesting()()
	execCommand = fakeInfoExecCommand
	defer func() { execCommand = exec.Command }()

	// Try installing nonexistent plugin
	err := InstallPlugin("not-exists", "v0.2.0", configtypes.TargetUnknown)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find plugin 'not-exists'")

	// Install login (standalone) plugin
	err = InstallPlugin("login", "v0.2.0", configtypes.TargetUnknown)
	assertions.Nil(err)
	// Verify installed plugin
	installedPlugins, err := pluginsupplier.GetInstalledPlugins()
	assertions.Nil(err)
	assertions.Equal(1, len(installedPlugins))
	assertions.Equal("login", installedPlugins[0].Name)

	// Try installing cluster plugin with no context-type
	err = InstallPlugin("cluster", "v0.2.0", configtypes.TargetUnknown)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to uniquely identify plugin 'cluster'. Please specify correct Target(kubernetes[k8s]/mission-control[tmc]) of the plugin with `--target` flag")

	// Try installing cluster plugin with context-type=tmc
	err = InstallPlugin("cluster", "v0.2.0", configtypes.TargetTMC)
	assertions.Nil(err)

	// Try installing cluster plugin through context-type=k8s with incorrect version
	err = InstallPlugin("cluster", "v1.0.0", configtypes.TargetK8s)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "plugin pre-download verification failed")

	// Try installing cluster plugin through context-type=k8s
	err = InstallPlugin("cluster", "v1.6.0", configtypes.TargetK8s)
	assertions.Nil(err)

	// Try installing management-cluster plugin from standalone discovery without context-type
	err = InstallPlugin("management-cluster", "v1.6.0", configtypes.TargetUnknown)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to uniquely identify plugin 'management-cluster'. Please specify correct Target(kubernetes[k8s]/mission-control[tmc]) of the plugin with `--target` flag")

	// Try installing management-cluster plugin from standalone discovery
	err = InstallPlugin("management-cluster", "v1.6.0", configtypes.TargetK8s)
	assertions.Nil(err)

	// Verify installed plugins
	installedStandalonePlugins, err := pluginsupplier.GetInstalledStandalonePlugins()
	assertions.Nil(err)
	assertions.Equal(2, len(installedStandalonePlugins))
	installedServerPlugins, err := pluginsupplier.GetInstalledServerPlugins()
	assertions.Nil(err)
	assertions.Equal(2, len(installedServerPlugins))

	expectedInstalledServerPlugins := []cli.PluginInfo{
		{
			Name:    "cluster",
			Version: "v1.6.0",
			Scope:   common.PluginScopeContext,
			Target:  configtypes.TargetK8s,
		},
		{
			Name:    "cluster",
			Version: "v0.2.0",
			Scope:   common.PluginScopeContext,
			Target:  configtypes.TargetTMC,
		},
	}
	expectedInstalledStandalonePlugins := []cli.PluginInfo{
		{
			Name:    "login",
			Version: "v0.2.0",
			Scope:   common.PluginScopeStandalone,
			Target:  configtypes.TargetUnknown,
		},
		{
			Name:    "management-cluster",
			Version: "v1.6.0",
			Scope:   common.PluginScopeStandalone,
			Target:  configtypes.TargetK8s,
		},
	}

	for i := 0; i < len(expectedInstalledServerPlugins); i++ {
		pd := findPluginInfo(installedServerPlugins, expectedInstalledServerPlugins[i].Name, expectedInstalledServerPlugins[i].Target)
		assertions.NotNil(pd)
		assertions.Equal(expectedInstalledServerPlugins[i].Version, pd.Version)
	}
	for i := 0; i < len(expectedInstalledStandalonePlugins); i++ {
		pd := findPluginInfo(installedStandalonePlugins, expectedInstalledStandalonePlugins[i].Name, expectedInstalledStandalonePlugins[i].Target)
		assertions.NotNil(pd)
		assertions.Equal(expectedInstalledStandalonePlugins[i].Version, pd.Version)
	}
}

func Test_InstallPlugin_InstalledPlugins_Central_Repo(t *testing.T) {
	t.Skip("Skipping until TANZU_CLI_PRE_RELEASE_REPO_IMAGE is no longer used")
	assertions := assert.New(t)

	defer setupLocalDistoForTesting()()
	execCommand = fakeInfoExecCommand
	defer func() { execCommand = exec.Command }()

	// Turn on the Central Repository feature
	featureArray := strings.Split(constants.FeatureCentralRepository, ".")
	err := configlib.SetFeature(featureArray[1], featureArray[2], "true")
	assertions.Nil(err)

	// Try installing nonexistent plugin
	err = InstallPlugin("not-exists", "v0.2.0", configtypes.TargetUnknown)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find plugin 'not-exists'")

	// Install login (standalone) plugin
	err = InstallPlugin("login", "v0.2.0", configtypes.TargetUnknown)
	assertions.Nil(err)
	// Verify installed plugin
	installedPlugins, err := pluginsupplier.GetInstalledPlugins()
	assertions.Nil(err)
	assertions.Equal(1, len(installedPlugins))
	assertions.Equal("login", installedPlugins[0].Name)

	// Try installing myplugin plugin with no context-type
	err = InstallPlugin("myplugin", "v0.2.0", configtypes.TargetUnknown)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to uniquely identify plugin 'myplugin'. Please specify correct Target(kubernetes[k8s]/mission-control[tmc]) of the plugin with `--target` flag")

	// Try installing myplugin plugin with context-type=tmc
	err = InstallPlugin("myplugin", "v0.2.0", configtypes.TargetTMC)
	assertions.Nil(err)

	// Try installing myplugin plugin through context-type=k8s with incorrect version
	err = InstallPlugin("myplugin", "v1.0.0", configtypes.TargetK8s)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "plugin pre-download verification failed")

	// Try installing myplugin plugin through context-type=k8s
	err = InstallPlugin("myplugin", "v1.6.0", configtypes.TargetK8s)
	assertions.Nil(err)

	// Try installing management-cluster plugin from standalone discovery
	err = InstallPlugin("management-cluster", "v1.6.0", configtypes.TargetK8s)
	assertions.Nil(err)

	// Verify installed plugins
	installedStandalonePlugins, err := pluginsupplier.GetInstalledStandalonePlugins()
	assertions.Nil(err)
	assertions.Equal(4, len(installedStandalonePlugins))
	installedServerPlugins, err := pluginsupplier.GetInstalledServerPlugins()
	assertions.Nil(err)
	assertions.Equal(0, len(installedServerPlugins))

	expectedInstalledStandalonePlugins := []cli.PluginInfo{
		{
			Name:    "login",
			Version: "v0.2.0",
			Scope:   common.PluginScopeStandalone,
			Target:  configtypes.TargetUnknown,
		},
		{
			Name:    "management-cluster",
			Version: "v1.6.0",
			Scope:   common.PluginScopeStandalone,
			Target:  configtypes.TargetK8s,
		},
		{
			Name:    "myplugin",
			Version: "v1.6.0",
			Scope:   common.PluginScopeStandalone,
			Target:  configtypes.TargetK8s,
		},
		{
			Name:    "myplugin",
			Version: "v0.2.0",
			Scope:   common.PluginScopeStandalone,
			Target:  configtypes.TargetTMC,
		},
	}

	for i := 0; i < len(expectedInstalledStandalonePlugins); i++ {
		pd := findPluginInfo(installedStandalonePlugins, expectedInstalledStandalonePlugins[i].Name, expectedInstalledStandalonePlugins[i].Target)
		assertions.NotNil(pd)
		assertions.Equal(expectedInstalledStandalonePlugins[i].Version, pd.Version)
	}
}

func Test_AvailablePlugins(t *testing.T) {
	assertions := assert.New(t)

	defer setupLocalDistoForTesting()()

	expectedDiscoveredPlugins := append(expectedDiscoveredContextPlugins, expectedDiscoveredStandalonePlugins...)
	discoveredPlugins, err := AvailablePlugins()
	assertions.Nil(err)
	assertions.Equal(len(expectedDiscoveredPlugins), len(discoveredPlugins))

	for i := 0; i < len(expectedDiscoveredPlugins); i++ {
		pd := findDiscoveredPlugin(discoveredPlugins, expectedDiscoveredPlugins[i].Name, expectedDiscoveredPlugins[i].Target)
		assertions.NotNil(pd)
		assertions.Equal(expectedDiscoveredPlugins[i].Name, pd.Name)
		assertions.Equal(expectedDiscoveredPlugins[i].RecommendedVersion, pd.RecommendedVersion)
		assertions.Equal(expectedDiscoveredPlugins[i].Target, pd.Target)
		assertions.Equal(expectedDiscoveredPlugins[i].Scope, pd.Scope)
		assertions.Equal(common.PluginStatusNotInstalled, pd.Status)
	}

	// Install login, cluster plugins
	mockInstallPlugin(assertions, "login", "v0.2.0", configtypes.TargetUnknown)
	mockInstallPlugin(assertions, "cluster", "v0.2.0", configtypes.TargetTMC)

	expectedInstallationStatusOfPlugins := []discovery.Discovered{
		{
			Name:             "cluster",
			Target:           configtypes.TargetTMC,
			InstalledVersion: "v0.2.0",
			Status:           common.PluginStatusInstalled,
		},
		{
			Name:             "cluster",
			Target:           configtypes.TargetK8s,
			InstalledVersion: "",
			Status:           common.PluginStatusNotInstalled,
		},
		{
			Name:             "login",
			Target:           configtypes.TargetUnknown,
			InstalledVersion: "v0.2.0",
			Status:           common.PluginStatusInstalled,
		},
	}

	// Get available plugin after install and verify installation status
	discoveredPlugins, err = AvailablePlugins()
	assertions.Nil(err)
	assertions.Equal(len(expectedDiscoveredPlugins), len(discoveredPlugins))

	for _, eisp := range expectedInstallationStatusOfPlugins {
		p := findDiscoveredPlugin(discoveredPlugins, eisp.Name, eisp.Target)
		assertions.NotNil(p)
		assertions.Equal(eisp.Status, p.Status)
		assertions.Equal(eisp.InstalledVersion, p.InstalledVersion)
	}

	// Install management-cluster, cluster plugins
	mockInstallPlugin(assertions, "management-cluster", "v0.2.0", configtypes.TargetTMC)
	mockInstallPlugin(assertions, "cluster", "v1.6.0", configtypes.TargetK8s)

	expectedInstallationStatusOfPlugins = []discovery.Discovered{
		{
			Name:             "management-cluster",
			Target:           configtypes.TargetTMC,
			InstalledVersion: "v0.2.0",
			Status:           common.PluginStatusInstalled,
		},
		{
			Name:             "cluster",
			Target:           configtypes.TargetK8s,
			InstalledVersion: "v1.6.0",
			Status:           common.PluginStatusInstalled,
		},
		{
			Name:             "management-cluster",
			Target:           configtypes.TargetK8s,
			InstalledVersion: "",
			Status:           common.PluginStatusNotInstalled,
		},
		{
			Name:             "login",
			Target:           configtypes.TargetUnknown,
			InstalledVersion: "v0.2.0",
			Status:           common.PluginStatusInstalled,
		},
	}

	// Get available plugin after install and verify installation status
	discoveredPlugins, err = AvailablePlugins()
	assertions.Nil(err)
	assertions.Equal(len(expectedDiscoveredPlugins), len(discoveredPlugins))

	for _, eisp := range expectedInstallationStatusOfPlugins {
		p := findDiscoveredPlugin(discoveredPlugins, eisp.Name, eisp.Target)
		assertions.NotNil(p)
		assertions.Equal(eisp.Status, p.Status, eisp.Name)
		assertions.Equal(eisp.InstalledVersion, p.InstalledVersion, eisp.Name)
	}
}

func Test_AvailablePlugins_With_K8s_None_Target_Plugin_Name_Conflict_With_One_Installed_Getting_Discovered(t *testing.T) {
	assertions := assert.New(t)

	defer setupLocalDistoForTesting()()

	expectedDiscoveredPlugins := append(expectedDiscoveredContextPlugins, expectedDiscoveredStandalonePlugins...)
	discoveredPlugins, err := AvailablePlugins()
	assertions.Nil(err)
	assertions.Equal(len(expectedDiscoveredPlugins), len(discoveredPlugins))

	// Install login, cluster plugins
	mockInstallPlugin(assertions, "login", "v0.2.0", configtypes.TargetUnknown)

	// Considering `login` plugin with `<none>` target is already installed and
	// getting discovered through some discoveries source
	//
	// if the same `login` plugin is now getting discovered with `k8s` target
	// verify the result of AvailablePlugins

	discoverySource := configtypes.PluginDiscovery{
		Local: &configtypes.LocalDiscovery{
			Name: "fake-with-k8s-target",
			Path: "standalone-k8s-target",
		},
	}
	err = configlib.SetCLIDiscoverySource(discoverySource)
	assertions.Nil(err)

	discoveredPlugins, err = AvailablePlugins()
	assertions.Nil(err)
	assertions.Equal(len(expectedDiscoveredPlugins), len(discoveredPlugins))

	expectedInstallationStatusOfPlugins := []discovery.Discovered{
		{
			Name:             "login",
			Target:           configtypes.TargetK8s,
			InstalledVersion: "v0.2.0",
			Status:           common.PluginStatusInstalled,
		},
	}

	for i := range discoveredPlugins {
		log.Infof("Discovered: %v, %v, %v, %v", discoveredPlugins[i].Name, discoveredPlugins[i].Target, discoveredPlugins[i].Status, discoveredPlugins[i].InstalledVersion)
	}

	for _, eisp := range expectedInstallationStatusOfPlugins {
		p := findDiscoveredPlugin(discoveredPlugins, eisp.Name, eisp.Target)
		assertions.NotNil(p)
		assertions.Equal(eisp.Status, p.Status, eisp.Name)
		assertions.Equal(eisp.InstalledVersion, p.InstalledVersion, eisp.Name)
	}
}

func Test_AvailablePlugins_With_K8s_None_Target_Plugin_Name_Conflict_With_Plugin_Installed_But_Not_Getting_Discovered(t *testing.T) {
	assertions := assert.New(t)

	defer setupLocalDistoForTesting()()

	expectedDiscoveredPlugins := append(expectedDiscoveredContextPlugins, expectedDiscoveredStandalonePlugins...)
	discoveredPlugins, err := AvailablePlugins()
	assertions.Nil(err)
	assertions.Equal(len(expectedDiscoveredPlugins), len(discoveredPlugins))

	// Install login, cluster plugins
	mockInstallPlugin(assertions, "login", "v0.2.0", configtypes.TargetUnknown)

	// Considering `login` plugin with `<none>` target is already installed and
	// getting discovered through some discoveries source
	//
	// if the same `login` plugin is now getting discovered with `k8s` target
	// verify the result of AvailablePlugins

	// Replace old discovery source to point to new standalone discovery where the same plugin is getting
	// discovered through k8s target
	discoverySource := configtypes.PluginDiscovery{
		Local: &configtypes.LocalDiscovery{
			Name: "fake",
			Path: "standalone-k8s-target",
		},
	}
	err = configlib.SetCLIDiscoverySource(discoverySource)
	assertions.Nil(err)

	discoveredPlugins, err = AvailablePlugins()
	assertions.Nil(err)
	assertions.Equal(len(expectedDiscoveredPlugins), len(discoveredPlugins))

	expectedInstallationStatusOfPlugins := []discovery.Discovered{
		{
			Name:             "login",
			Target:           configtypes.TargetK8s,
			InstalledVersion: "v0.2.0",
			Status:           common.PluginStatusInstalled,
		},
	}

	for _, eisp := range expectedInstallationStatusOfPlugins {
		p := findDiscoveredPlugin(discoveredPlugins, eisp.Name, eisp.Target)
		assertions.NotNil(p)
		assertions.Equal(eisp.Status, p.Status, eisp.Name)
		assertions.Equal(eisp.InstalledVersion, p.InstalledVersion, eisp.Name)
	}
}

func Test_AvailablePlugins_From_LocalSource(t *testing.T) {
	assertions := assert.New(t)

	defer setupLocalDistoForTesting()()

	currentDirAbsPath, _ := filepath.Abs(".")
	discoveredPlugins, err := AvailablePluginsFromLocalSource(filepath.Join(currentDirAbsPath, "test", "local"))
	assertions.Nil(err)

	expectedInstallationStatusOfPlugins := []discovery.Discovered{
		{
			Name:   "cluster",
			Scope:  common.PluginScopeStandalone,
			Target: configtypes.TargetK8s,
			Status: common.PluginStatusNotInstalled,
		},
		{
			Name:   "management-cluster",
			Scope:  common.PluginScopeStandalone,
			Target: configtypes.TargetK8s,
			Status: common.PluginStatusNotInstalled,
		},
		{
			Name:   "management-cluster",
			Scope:  common.PluginScopeStandalone,
			Target: configtypes.TargetTMC,
			Status: common.PluginStatusNotInstalled,
		},
		{
			Name:   "login",
			Scope:  common.PluginScopeStandalone,
			Target: configtypes.TargetK8s,
			Status: common.PluginStatusNotInstalled,
		},
		{
			Name:   "cluster",
			Scope:  common.PluginScopeStandalone,
			Target: configtypes.TargetTMC,
			Status: common.PluginStatusNotInstalled,
		},
		{
			Name:   "myplugin",
			Scope:  common.PluginScopeStandalone,
			Target: configtypes.TargetK8s,
			Status: common.PluginStatusNotInstalled,
		},
		{
			Name:   "myplugin",
			Scope:  common.PluginScopeStandalone,
			Target: configtypes.TargetTMC,
			Status: common.PluginStatusNotInstalled,
		},
	}

	assertions.Equal(len(expectedInstallationStatusOfPlugins), len(discoveredPlugins))

	for _, eisp := range expectedInstallationStatusOfPlugins {
		p := findDiscoveredPlugin(discoveredPlugins, eisp.Name, eisp.Target)
		assertions.NotNil(p, "plugin %q with target %q not found", eisp.Name, eisp.Target)
		assertions.Equal(eisp.Status, p.Status, "status mismatch for plugin %q with target %q", eisp.Name, eisp.Target)
		assertions.Equal(eisp.Scope, p.Scope, "scope mismatch for plugin %q with target %q", eisp.Name, eisp.Target)
	}
}

func Test_InstallPlugin_InstalledPlugins_From_LocalSource(t *testing.T) {
	assertions := assert.New(t)

	defer setupLocalDistoForTesting()()

	execCommand = fakeInfoExecCommand
	defer func() { execCommand = exec.Command }()

	currentDirAbsPath, _ := filepath.Abs(".")
	localPluginSourceDir := filepath.Join(currentDirAbsPath, "test", "local")

	// Try installing nonexistent plugin
	err := InstallPluginsFromLocalSource("not-exists", "v0.2.0", configtypes.TargetUnknown, localPluginSourceDir, false)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find plugin 'not-exists'")

	// Install login from local source directory
	err = InstallPluginsFromLocalSource("login", "v0.2.0", configtypes.TargetUnknown, localPluginSourceDir, false)
	assertions.Nil(err)
	// Verify installed plugin
	installedStandalonePlugins, err := pluginsupplier.GetInstalledStandalonePlugins()
	assertions.Nil(err)
	assertions.Equal(1, len(installedStandalonePlugins))
	assertions.Equal("login", installedStandalonePlugins[0].Name)

	// Try installing cluster plugin from local source directory
	err = InstallPluginsFromLocalSource("cluster", "v0.2.0", configtypes.TargetTMC, localPluginSourceDir, false)
	assertions.Nil(err)
	installedStandalonePlugins, err = pluginsupplier.GetInstalledStandalonePlugins()
	assertions.Nil(err)
	assertions.Equal(2, len(installedStandalonePlugins))
	installedServerPlugins, err := pluginsupplier.GetInstalledServerPlugins()
	assertions.Nil(err)
	assertions.Equal(0, len(installedServerPlugins))

	// Try installing a plugin from incorrect local path
	err = InstallPluginsFromLocalSource("cluster", "v0.2.0", configtypes.TargetTMC, "fakepath", false)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "no such file or directory")
}

func Test_DescribePlugin(t *testing.T) {
	assertions := assert.New(t)

	defer setupLocalDistoForTesting()()

	// Try to describe plugin when plugin is not installed
	_, err := DescribePlugin("login", configtypes.TargetUnknown)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find plugin 'login'")

	// Install login (standalone) package
	mockInstallPlugin(assertions, "login", "v0.2.0", configtypes.TargetUnknown)

	// Try to describe plugin when plugin after installing plugin
	pd, err := DescribePlugin("login", configtypes.TargetUnknown)
	assertions.Nil(err)
	assertions.Equal("login", pd.Name)
	assertions.Equal("v0.2.0", pd.Version)

	// Try to describe plugin when plugin is not installed
	_, err = DescribePlugin("cluster", configtypes.TargetTMC)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find plugin 'cluster'")

	// Install cluster (context) package
	mockInstallPlugin(assertions, "cluster", "v0.2.0", configtypes.TargetTMC)

	// Try to describe plugin when plugin after installing plugin
	pd, err = DescribePlugin("cluster", configtypes.TargetTMC)
	assertions.Nil(err)
	assertions.Equal("cluster", pd.Name)
	assertions.Equal("v0.2.0", pd.Version)
}

func Test_DeletePlugin(t *testing.T) {
	assertions := assert.New(t)

	defer setupLocalDistoForTesting()()

	// Try to delete plugin when plugin is not installed
	err := DeletePlugin(DeletePluginOptions{PluginName: "login", Target: configtypes.TargetUnknown, ForceDelete: true})
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find plugin 'login'")

	// Install login (standalone) package
	mockInstallPlugin(assertions, "login", "v0.2.0", configtypes.TargetUnknown)

	// Try to delete plugin when plugin is installed
	err = DeletePlugin(DeletePluginOptions{PluginName: "cluster", Target: configtypes.TargetTMC, ForceDelete: true})
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find plugin 'cluster'")

	// Install cluster (context) package from TMC target
	mockInstallPlugin(assertions, "cluster", "v0.2.0", configtypes.TargetTMC)

	// Try to Delete plugin after installing plugin
	err = DeletePlugin(DeletePluginOptions{PluginName: "cluster", Target: configtypes.TargetTMC, ForceDelete: true})
	assertions.Nil(err)

	// Install cluster (context) package from TMC target
	mockInstallPlugin(assertions, "cluster", "v0.2.0", configtypes.TargetTMC)

	// Try to Delete plugin after installing plugin
	err = DeletePlugin(DeletePluginOptions{PluginName: "cluster", Target: "", ForceDelete: true})
	assertions.Nil(err)

	// Install cluster (context) package from TMC target
	mockInstallPlugin(assertions, "cluster", "v0.2.0", configtypes.TargetTMC)
	// Install cluster (context) package from k8s target
	mockInstallPlugin(assertions, "cluster", "v1.6.0", configtypes.TargetK8s)
	// Try to Delete plugin without passing target after installing plugin with different targets
	err = DeletePlugin(DeletePluginOptions{PluginName: "cluster", Target: "", ForceDelete: true})
	assertions.Contains(err.Error(), "unable to uniquely identify plugin 'cluster'. Please specify correct Target(kubernetes[k8s]/mission-control[tmc]) of the plugin with `--target` flag")
}

func Test_ValidatePlugin(t *testing.T) {
	assertions := assert.New(t)

	pd := cli.PluginInfo{}
	err := ValidatePlugin(&pd)
	assertions.Contains(err.Error(), "plugin name cannot be empty")

	pd.Name = "fake-plugin"
	err = ValidatePlugin(&pd)
	assertions.NotContains(err.Error(), "plugin name cannot be empty")
	assertions.Contains(err.Error(), "plugin \"fake-plugin\" version cannot be empty")
	assertions.Contains(err.Error(), "plugin \"fake-plugin\" group cannot be empty")
}

func Test_SyncPlugins_All_Plugins(t *testing.T) {
	assertions := assert.New(t)

	defer setupLocalDistoForTesting()()
	execCommand = fakeInfoExecCommand
	defer func() { execCommand = exec.Command }()

	expectedDiscoveredPlugins := append(expectedDiscoveredContextPlugins, expectedDiscoveredStandalonePlugins...)

	// Get all available plugins(standalone+context-aware) and verify the status is `not installed`
	discovered, err := AvailablePlugins()
	assertions.Nil(err)
	assertions.Equal(len(expectedDiscoveredPlugins), len(discovered))

	for _, edp := range expectedDiscoveredPlugins {
		p := findDiscoveredPlugin(discovered, edp.Name, edp.Target)
		assertions.NotNil(p)
		assertions.Equal(common.PluginStatusNotInstalled, p.Status)
	}

	// Sync all available plugins
	err = SyncPlugins()
	assertions.Nil(err)

	// Get all available plugins(standalone+context-aware) and verify the status is updated to `installed`
	discovered, err = AvailablePlugins()
	assertions.Nil(err)
	assertions.Equal(len(expectedDiscoveredPlugins), len(discovered))

	for _, edp := range expectedDiscoveredPlugins {
		p := findDiscoveredPlugin(discovered, edp.Name, edp.Target)
		assertions.NotNil(p)
		assertions.Equal(common.PluginStatusInstalled, p.Status)
		assertions.Equal(edp.RecommendedVersion, p.InstalledVersion)
	}
}

func Test_getInstalledButNotDiscoveredStandalonePlugins(t *testing.T) {
	assertions := assert.New(t)

	availablePlugins := []discovery.Discovered{{Name: "fake1", DiscoveryType: "oci", RecommendedVersion: "v1.0.0", Status: common.PluginStatusInstalled}}
	installedPlugin := []cli.PluginInfo{{Name: "fake2", Version: "v2.0.0", Discovery: "local"}}

	// If installed plugin is not part of available(discovered) plugins
	plugins := getInstalledButNotDiscoveredStandalonePlugins(availablePlugins, installedPlugin)
	assertions.Equal(len(plugins), 1)
	assertions.Equal("fake2", plugins[0].Name)
	assertions.Equal("v2.0.0", plugins[0].RecommendedVersion)
	assertions.Equal(common.PluginStatusInstalled, plugins[0].Status)

	// If installed plugin is part of available(discovered) plugins and provided available plugin is already marked as `installed`
	installedPlugin = append(installedPlugin, cli.PluginInfo{Name: "fake1", Version: "v1.0.0", Discovery: "local"})
	plugins = getInstalledButNotDiscoveredStandalonePlugins(availablePlugins, installedPlugin)
	assertions.Equal(len(plugins), 1)
	assertions.Equal("fake2", plugins[0].Name)
	assertions.Equal("v2.0.0", plugins[0].RecommendedVersion)
	assertions.Equal(common.PluginStatusInstalled, plugins[0].Status)

	// If installed plugin is part of available(discovered) plugins and provided available plugin is already marked as `not installed`
	// then test the availablePlugin status gets updated to `installed`
	availablePlugins[0].Status = common.PluginStatusNotInstalled
	plugins = getInstalledButNotDiscoveredStandalonePlugins(availablePlugins, installedPlugin)
	assertions.Equal(len(plugins), 1)
	assertions.Equal("fake2", plugins[0].Name)
	assertions.Equal("v2.0.0", plugins[0].RecommendedVersion)
	assertions.Equal(common.PluginStatusInstalled, plugins[0].Status)
	assertions.Equal(common.PluginStatusInstalled, availablePlugins[0].Status)

	// If installed plugin is part of available(discovered) plugins and versions installed is different than discovered version
	availablePlugins[0].Status = common.PluginStatusNotInstalled
	availablePlugins[0].RecommendedVersion = "v4.0.0"
	plugins = getInstalledButNotDiscoveredStandalonePlugins(availablePlugins, installedPlugin)
	assertions.Equal(len(plugins), 1)
	assertions.Equal("fake2", plugins[0].Name)
	assertions.Equal("v2.0.0", plugins[0].RecommendedVersion)
	assertions.Equal(common.PluginStatusInstalled, plugins[0].Status)
	assertions.Equal(common.PluginStatusInstalled, availablePlugins[0].Status)
}

func Test_setAvailablePluginsStatus(t *testing.T) {
	assertions := assert.New(t)

	availablePlugins := []discovery.Discovered{{Name: "fake1", DiscoveryType: "oci", RecommendedVersion: "v1.0.0", Status: common.PluginStatusNotInstalled, Target: configtypes.TargetK8s}}
	installedPlugin := []cli.PluginInfo{{Name: "fake2", Version: "v2.0.0", Discovery: "local", DiscoveredRecommendedVersion: "v2.0.0", Target: configtypes.TargetUnknown}}

	// If installed plugin is not part of available(discovered) plugins then
	// installed version == ""
	// status  == not installed
	setAvailablePluginsStatus(availablePlugins, installedPlugin)
	assertions.Equal(len(availablePlugins), 1)
	assertions.Equal("fake1", availablePlugins[0].Name)
	assertions.Equal("v1.0.0", availablePlugins[0].RecommendedVersion)
	assertions.Equal("", availablePlugins[0].InstalledVersion)
	assertions.Equal(common.PluginStatusNotInstalled, availablePlugins[0].Status)

	// If installed plugin is not part of available(discovered) plugins because of the Target mismatch
	installedPlugin = []cli.PluginInfo{{Name: "fake1", Version: "v1.0.0", Discovery: "local", DiscoveredRecommendedVersion: "v1.0.0", Target: configtypes.TargetUnknown}}
	setAvailablePluginsStatus(availablePlugins, installedPlugin)
	assertions.Equal(len(availablePlugins), 1)
	assertions.Equal("fake1", availablePlugins[0].Name)
	assertions.Equal("v1.0.0", availablePlugins[0].RecommendedVersion)
	assertions.Equal("", availablePlugins[0].InstalledVersion)
	assertions.Equal(common.PluginStatusNotInstalled, availablePlugins[0].Status)

	// If installed plugin is part of available(discovered) plugins and provided available plugin is already installed
	installedPlugin = []cli.PluginInfo{{Name: "fake1", Version: "v1.0.0", Discovery: "local", DiscoveredRecommendedVersion: "v1.0.0", Target: configtypes.TargetK8s}}
	setAvailablePluginsStatus(availablePlugins, installedPlugin)
	assertions.Equal(len(availablePlugins), 1)
	assertions.Equal("fake1", availablePlugins[0].Name)
	assertions.Equal("v1.0.0", availablePlugins[0].RecommendedVersion)
	assertions.Equal("v1.0.0", availablePlugins[0].InstalledVersion)
	assertions.Equal(common.PluginStatusInstalled, availablePlugins[0].Status)

	// If installed plugin is part of available(discovered) plugins but recommended discovered version is different than the one installed
	// then available plugin status should show 'update available'
	availablePlugins = []discovery.Discovered{{Name: "fake1", DiscoveryType: "oci", RecommendedVersion: "v8.0.0-latest", Status: common.PluginStatusNotInstalled}}
	installedPlugin = []cli.PluginInfo{{Name: "fake1", Version: "v1.0.0", Discovery: "local", DiscoveredRecommendedVersion: "v1.0.0"}}
	setAvailablePluginsStatus(availablePlugins, installedPlugin)
	assertions.Equal(len(availablePlugins), 1)
	assertions.Equal("fake1", availablePlugins[0].Name)
	assertions.Equal("v8.0.0-latest", availablePlugins[0].RecommendedVersion)
	assertions.Equal("v1.0.0", availablePlugins[0].InstalledVersion)
	assertions.Equal(common.PluginStatusUpdateAvailable, availablePlugins[0].Status)

	// If installed plugin is part of available(discovered) plugins but recommended discovered version is same as the recommended discovered version
	// for the installed plugin(stored as part of catalog cache) then available plugin status should show 'installed'
	availablePlugins = []discovery.Discovered{{Name: "fake1", DiscoveryType: "oci", RecommendedVersion: "v8.0.0-latest", Status: common.PluginStatusNotInstalled}}
	installedPlugin = []cli.PluginInfo{{Name: "fake1", Version: "v1.0.0", Discovery: "local", DiscoveredRecommendedVersion: "v8.0.0-latest"}}
	setAvailablePluginsStatus(availablePlugins, installedPlugin)
	assertions.Equal(len(availablePlugins), 1)
	assertions.Equal("fake1", availablePlugins[0].Name)
	assertions.Equal("v8.0.0-latest", availablePlugins[0].RecommendedVersion)
	assertions.Equal("v1.0.0", availablePlugins[0].InstalledVersion)
	assertions.Equal(common.PluginStatusInstalled, availablePlugins[0].Status)

	// If installed plugin is part of available(discovered) plugins and versions installed is different from discovered version
	// it should be reflected in RecommendedVersion as well as InstalledVersion and status should be `update available`
	availablePlugins[0].Status = common.PluginStatusNotInstalled
	availablePlugins[0].RecommendedVersion = "v3.0.0"
	setAvailablePluginsStatus(availablePlugins, installedPlugin)
	assertions.Equal(len(availablePlugins), 1)
	assertions.Equal("fake1", availablePlugins[0].Name)
	assertions.Equal("v3.0.0", availablePlugins[0].RecommendedVersion)
	assertions.Equal("v1.0.0", availablePlugins[0].InstalledVersion)
	assertions.Equal(common.PluginStatusUpdateAvailable, availablePlugins[0].Status)
}

func Test_DiscoverPluginsFromLocalSourceBasedOnManifestFile(t *testing.T) {
	assertions := assert.New(t)

	// When passing directory structure where manifest.yaml and plugin_manifest.yaml both files are missing
	_, err := discoverPluginsFromLocalSourceBasedOnManifestFile(filepath.Join("test", "local"))
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "could not find manifest.yaml file")
	assertions.Contains(err.Error(), "could not find plugin_manifest.yaml file")

	// When passing directory structure which contains manifest.yaml file
	discoveredPlugins, err := discoverPluginsFromLocalSourceBasedOnManifestFile(filepath.Join("test", "legacy"))
	assertions.Nil(err)
	assertions.Equal(2, len(discoveredPlugins))

	assertions.Equal("foo", discoveredPlugins[0].Name)
	assertions.Equal("Foo plugin", discoveredPlugins[0].Description)
	assertions.Equal("v0.12.0", discoveredPlugins[0].RecommendedVersion)
	assertions.Equal(common.PluginScopeStandalone, discoveredPlugins[0].Scope)
	assertions.Equal(configtypes.TargetUnknown, discoveredPlugins[0].Target)

	assertions.Equal("bar", discoveredPlugins[1].Name)
	assertions.Equal("Bar plugin", discoveredPlugins[1].Description)
	assertions.Equal("v0.10.0", discoveredPlugins[1].RecommendedVersion)
	assertions.Equal(common.PluginScopeStandalone, discoveredPlugins[1].Scope)
	assertions.Equal(configtypes.TargetUnknown, discoveredPlugins[1].Target)

	// When passing directory structure which contains plugin_manifest.yaml file
	discoveredPlugins, err = discoverPluginsFromLocalSourceBasedOnManifestFile(filepath.Join("test", "artifacts1"))
	assertions.Nil(err)
	assertions.Equal(2, len(discoveredPlugins))

	assertions.Equal("foo", discoveredPlugins[0].Name)
	assertions.Equal("Foo plugin", discoveredPlugins[0].Description)
	assertions.Equal("v0.12.0", discoveredPlugins[0].RecommendedVersion)
	assertions.Equal(common.PluginScopeStandalone, discoveredPlugins[0].Scope)
	assertions.Equal(configtypes.TargetK8s, discoveredPlugins[0].Target)

	assertions.Equal("bar", discoveredPlugins[1].Name)
	assertions.Equal("Bar plugin", discoveredPlugins[1].Description)
	assertions.Equal("v0.10.0", discoveredPlugins[1].RecommendedVersion)
	assertions.Equal(common.PluginScopeStandalone, discoveredPlugins[1].Scope)
	assertions.Equal(configtypes.TargetGlobal, discoveredPlugins[1].Target)
}

func Test_InstallPluginsFromLocalSourceWithLegacyDirectoryStructure(t *testing.T) {
	assertions := assert.New(t)

	execCommand = fakeInfoExecCommand
	defer func() { execCommand = exec.Command }()

	// Using generic InstallPluginsFromLocalSource to test the legacy directory install
	// When passing legacy directory structure which contains manifest.yaml file
	err := InstallPluginsFromLocalSource("all", "", configtypes.TargetUnknown, filepath.Join("test", "legacy"), false)
	assertions.Nil(err)

	// Verify installed plugin
	installedStandalonePlugins, err := pluginsupplier.GetInstalledStandalonePlugins()
	assertions.Nil(err)
	assertions.Equal(2, len(installedStandalonePlugins))
	assertions.ElementsMatch([]string{"bar", "foo"}, []string{installedStandalonePlugins[0].Name, installedStandalonePlugins[1].Name})
	installedServerPlugins, err := pluginsupplier.GetInstalledServerPlugins()
	assertions.Nil(err)
	assertions.Equal(0, len(installedServerPlugins))
}

func Test_VerifyRegistry(t *testing.T) {
	assertions := assert.New(t)

	var err error

	testImage := "fake.repo.com/image:v1.0.0"
	err = configureAndTestVerifyRegistry(testImage, "", "", "")
	assertions.NotNil(err)

	err = configureAndTestVerifyRegistry(testImage, "fake.repo.com", "", "")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "fake.repo.com/image", "", "")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "fake.repo.com/foo", "", "")
	assertions.NotNil(err)

	err = configureAndTestVerifyRegistry(testImage, "", "fake.repo.com", "")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "", "fake.repo.com/image", "")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "", "fake.repo.com/foo", "")
	assertions.NotNil(err)

	err = configureAndTestVerifyRegistry(testImage, "", "", "fake.repo.com")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "", "", "fake.repo.com/image")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "", "", "fake.repo.com/foo")
	assertions.NotNil(err)

	err = configureAndTestVerifyRegistry(testImage, "fake.repo.com", "", "fake.repo.com/foo")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "", "fake.repo.com", "fake.repo.com/foo")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "fake.repo.com", "fake.repo.com", "fake.repo.com/foo")
	assertions.Nil(err)

	err = configureAndTestVerifyRegistry(testImage, "fake.repo.com.private.com", "", "")
	assertions.NotNil(err)
	err = configureAndTestVerifyRegistry(testImage, "private.fake.repo.com", "", "")
	assertions.NotNil(err)
	err = configureAndTestVerifyRegistry(testImage, "fake.repo.com/image/foo", "", "")
	assertions.NotNil(err)

	err = configureAndTestVerifyRegistry(testImage, "", "", "fake.repo.com.private.com,private.fake.repo.com")
	assertions.NotNil(err)
	err = configureAndTestVerifyRegistry(testImage, "", "", "fake.repo.com,private.fake.repo.com")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "", "", "private.fake.repo.com,fake.repo.com")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "", "", "fake.repo.com/image,fake.repo.com")
	assertions.Nil(err)

	testImage = "fake1.repo.com/image:v1.0.0"
	err = configureAndTestVerifyRegistry(testImage, "fake.repo.com/image", "", "")
	assertions.NotNil(err)
	err = configureAndTestVerifyRegistry(testImage, "fake.repo.com/image,fake1.repo.com/image", "", "")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "fake1.repo.com/image", "", "")
	assertions.Nil(err)
}

func configureAndTestVerifyRegistry(testImage, defaultRegistry, customImageRepository, allowedRegistries string) error {
	config.DefaultAllowedPluginRepositories = defaultRegistry
	os.Setenv(constants.ConfigVariableCustomImageRepository, customImageRepository)
	os.Setenv(constants.AllowedRegistries, allowedRegistries)

	err := verifyRegistry(testImage)

	config.DefaultAllowedPluginRepositories = ""
	os.Setenv(constants.ConfigVariableCustomImageRepository, "")
	os.Setenv(constants.AllowedRegistries, "")
	return err
}

func TestVerifyArtifactLocation(t *testing.T) {
	tcs := []struct {
		name   string
		uri    string
		errStr string
	}{
		{
			name: "trusted location",
			uri:  "https://storage.googleapis.com/tanzu-cli-advanced-plugins/artifacts/latest/tanzu-foo-darwin-amd64",
		},
		{
			name:   "untrusted location",
			uri:    "https://storage.googleapis.com/tanzu-cli-advanced-plugins-artifacts/latest/tanzu-foo-darwin-amd64",
			errStr: "untrusted artifact location detected with URI \"https://storage.googleapis.com/tanzu-cli-advanced-plugins-artifacts/latest/tanzu-foo-darwin-amd64\". Allowed locations are [https://storage.googleapis.com/tanzu-cli-advanced-plugins/ https://tmc-cli.s3-us-west-2.amazonaws.com/plugins/artifacts]",
		},
		{
			name: "trusted location",
			uri:  "https://tmc-cli.s3-us-west-2.amazonaws.com/plugins/artifacts",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := verifyArtifactLocation(tc.uri)
			if tc.errStr != "" {
				assert.EqualError(t, err, tc.errStr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVerifyPluginPostDownload(t *testing.T) {
	tcs := []struct {
		name string
		p    *discovery.Discovered
		d    string
		path string
		err  string
	}{
		{
			name: "success - no source digest",
			p:    &discovery.Discovered{Name: "login"},
			path: "test/local/distribution/v0.2.0/tanzu-login",
		},
		{
			name: "success - with source digest",
			p:    &discovery.Discovered{Name: "login"},
			d:    "e109197e3e4ed9f13065596367f1fd0992df43717c7098324da4a00cb8b81c36",
			path: "test/local/distribution/v0.2.0/tanzu-login",
		},
		{
			name: "failure - digest mismatch",
			p:    &discovery.Discovered{Name: "login"},
			d:    "f3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			path: "test/local/distribution/v0.2.0/tanzu-login",
			err:  "plugin \"login\" has been corrupted during download. source digest: f3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855, actual digest: e109197e3e4ed9f13065596367f1fd0992df43717c7098324da4a00cb8b81c36",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			b, err := os.ReadFile(tc.path)
			assert.NoError(t, err)

			err = verifyPluginPostDownload(tc.p, tc.d, b)
			if tc.err != "" {
				assert.EqualError(t, err, tc.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_removeDuplicates(t *testing.T) {
	assertions := assert.New(t)

	tcs := []struct {
		name           string
		inputPlugins   []discovery.Discovered
		expectedResult []discovery.Discovered
	}{
		{
			name: "when plugin name-target conflict happens with '' and 'k8s' targeted plugins ",
			inputPlugins: []discovery.Discovered{
				{
					Name:   "foo",
					Target: configtypes.TargetUnknown,
					Scope:  common.PluginScopeStandalone,
				},
				{
					Name:   "foo",
					Target: configtypes.TargetK8s,
					Scope:  common.PluginScopeStandalone,
				},
				{
					Name:   "bar",
					Target: configtypes.TargetK8s,
					Scope:  common.PluginScopeStandalone,
				},
			},
			expectedResult: []discovery.Discovered{
				{
					Name:   "foo",
					Target: configtypes.TargetK8s,
					Scope:  common.PluginScopeStandalone,
				},
				{
					Name:   "bar",
					Target: configtypes.TargetK8s,
					Scope:  common.PluginScopeStandalone,
				},
			},
		},
		{
			name: "when same plugin exists for '', 'k8s' and 'tmc' target as standalone plugin",
			inputPlugins: []discovery.Discovered{
				{
					Name:   "foo",
					Target: configtypes.TargetUnknown,
					Scope:  common.PluginScopeStandalone,
				},
				{
					Name:   "foo",
					Target: configtypes.TargetK8s,
					Scope:  common.PluginScopeStandalone,
				},
				{
					Name:   "foo",
					Target: configtypes.TargetTMC,
					Scope:  common.PluginScopeStandalone,
				},
			},
			expectedResult: []discovery.Discovered{
				{
					Name:   "foo",
					Target: configtypes.TargetK8s,
					Scope:  common.PluginScopeStandalone,
				},
				{
					Name:   "foo",
					Target: configtypes.TargetTMC,
					Scope:  common.PluginScopeStandalone,
				},
			},
		},
		{
			name: "when foo standalone plugin is available with `k8s` and `` target and also available as context-scoped plugin with `k8s` target",
			inputPlugins: []discovery.Discovered{
				{
					Name:   "foo",
					Target: configtypes.TargetUnknown,
					Scope:  common.PluginScopeStandalone,
				},
				{
					Name:   "foo",
					Target: configtypes.TargetK8s,
					Scope:  common.PluginScopeStandalone,
				},
				{
					Name:   "foo",
					Target: configtypes.TargetK8s,
					Scope:  common.PluginScopeContext,
				},
			},
			expectedResult: []discovery.Discovered{
				{
					Name:   "foo",
					Target: configtypes.TargetK8s,
					Scope:  common.PluginScopeContext,
				},
			},
		},
		{
			name: "when tmc targeted plugin exists as standalone as well as context-scope",
			inputPlugins: []discovery.Discovered{
				{
					Name:   "foo",
					Target: configtypes.TargetTMC,
					Scope:  common.PluginScopeStandalone,
				},
				{
					Name:   "foo",
					Target: configtypes.TargetTMC,
					Scope:  common.PluginScopeContext,
				},
			},
			expectedResult: []discovery.Discovered{
				{
					Name:   "foo",
					Target: configtypes.TargetTMC,
					Scope:  common.PluginScopeContext,
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			result := combineDuplicatePlugins(tc.inputPlugins)
			assertions.Equal(len(result), len(tc.expectedResult))
			for i := range tc.expectedResult {
				p := findDiscoveredPlugin(result, tc.expectedResult[i].Name, tc.expectedResult[i].Target)
				assertions.Equal(p.Scope, tc.expectedResult[i].Scope)
			}
		})
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)
	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}
	filePath := os.Getenv("FILE_PATH")
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read plugin\n")
		os.Exit(2)
	}
	fmt.Fprint(os.Stdout, string(bytes))
}
