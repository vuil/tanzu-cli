# Tanzu CLI Documentation

VVV

## What is tanzu cli

## Important Concepts

Tanzu CLI Plugin: (or plugin for short unless otherwise noted) An executable binary, with one or more invocable commands, used to extend the functionality of the Tanzu CLI. See the next section what a complete requirements for a plugin.

Context:

Target:

## Tanzu CLI Plugin Architecture

explain a plugin

- standalone
- command tree
- passthrough
- provide metadata
- core cli responsible for lifecycle

VVV update!

### Context

VVV update!

Context is an isolated scope of relevant client-side configurations for a
combination of user identity and server identity. There can be multiple
contexts for the same combination of `(user, server)`. Previously, this was
referred to as `Server` in the Tanzu CLI. Going forward we shall refer to them
as `Context` to be explicit.

If a plugin wants to access the context it should use the
[context-related APIs](https://github.com/vmware-tanzu/tanzu-plugin-runtime/blob/main/config/contexts.go)
in the tanzu plugin runtime library to ensure forward compatibility. For
example, to get the current active context use the below snippet:

```go
import (
  config "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
  cfgtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

ctx, err := config.GetCurrentContext(cfgtypes.TargetK8s)
```

**Note:** The Tanzu CLI will also ensures interoperability between `Server` and
`Context` for as long as the Server concept is supported.

VVV more details, link to doc on config handling

### Plugin Discovery Source

Discovery is the interface to fetch the list of available plugins, their
supported versions and how to download them either standalone or scoped to a
context(server). E.g., the CLIPlugin resource in a management cluster, OCI
based plugin discovery for standalone plugins, a similar REST API etc. provides
the list of available plugins and details about the supported versions. Having
a separate interface for discovery helps to decouple discovery (which is
usually tied to a server or user identity) from distribution (which can be
shared).

Plugins can be of two different types:

  1. Standalone plugins: independent of the CLI context and are discovered using standalone discovery source

      This type of plugins are not associated with the `tanzu login` workflow and are available to the Tanzu CLI independent of the CLI context.

  2. Context(server) scoped plugins: scoped to one or more contexts and are discovered using kubernetes or other server associated discovery source

      This type of plugins are associated with the `tanzu login` workflow and are discovered from the management-cluster or global server endpoint.
      In terms of management-clusters, this type of plugins are mostly associated with the installed packages.

      Example:

      As a developer of a `velero` package, I would like to create a Tanzu CLI
plugin that can be used to configure and manage installed `velero` package
configuration.

      This usecase can be handled with context scoped plugins by installing
`CLIPlugin` CR related to `velero` plugin on the management-cluster as part of
`velero` package installation.

      ```sh
      # Login to a management-cluster
      $ tanzu login

      # Installs velero package to the management-cluster along with `velero` CLIPlugin resource
      $ tanzu package install velero-pkg --package-name velero.tanzu.vmware.com

      # Plugin list should show a new `velero` plugin available
      $ tanzu plugin list
        NAME     DESCRIPTION                    SCOPE       DISCOVERY          VERSION    STATUS
        velero   Backup and restore operations  Context     cluster-default    v0.1.0     not installed

      # Install velero plugin
      $ tanzu plugin install velero
      ```

The default standalone plugins discovery source automatically gets added to the tanzu config files and plugins from this discovery source are automatically discovered.

```sh
$ tanzu plugin list
  NAME                DESCRIPTION                                 SCOPE       DISCOVERY             VERSION      STATUS
  login               Login to the platform                       Standalone  default               v0.11.0-dev  not installed
  management-cluster  Kubernetes management-cluster operations    Standalone  default               v0.11.0-dev  not installed
```

To add a plugin discovery source the command `tanzu plugin source add` should
be used. For example, assuming the admin plugin's manifests are released as a
carvel-package at OCI image
`projects.registry.vmware.com/tkg/tanzu-plugins/admin-plugins:v0.11.0-dev` then
we use the following command to add that discovery source to the tanzu
configuration.

```sh
 tanzu plugin source add --name admin --type oci --uri projects.registry.vmware.com/tkg/tanzu-plugins/admin-plugins:v0.11.0-dev
```

We can check the newly added discovery source with

```sh
$ tanzu plugin source list
  NAME     TYPE  SCOPE
  default  oci   Standalone
  admin    oci   Standalone
```

This will allow the tanzu CLI to discover new available plugins in the newly added discovery source.

```sh
$ tanzu plugin list
  NAME                DESCRIPTION                                                        SCOPE       DISCOVERY             VERSION      STATUS
  login               Login to the platform                                              Standalone  default               v0.11.0-dev  not installed
  management-cluster  Kubernetes management-cluster operations                           Standalone  default               v0.11.0-dev  not installed
  builder             Builder plugin for CLI                                             Standalone  admin                 v0.11.0-dev  not installed
  test                Test plugin for CLI                                                Standalone  admin                 v0.11.0-dev  not installed
```

## Common commands

There is a small set of commands that every plugin provides. These commands are
typically not invoked directly by CLI users; some are in fact hidden for that
reason. Below is a brief summary of these commands

`version`: provides basic version information about the plugin, likely the only common command of broad use to the CLI user.

`info`: provides metadata about the plugin that the CLI will use when presenting information about plugins or when performing lifecycle operations on them.

`post-install`: provide a means for a plugin to optionally implement some logic to be invoked right after a plugin is installed.

`generate-docs`: generate a tree of documentation markdown files for the commands the plugin provides, typically used by the CLI's generate-all-docs command to produce command documentation for all installed plugins

`lint`: validate the command name and arguments to flag any new terms unaccounted for in the CLI taxonomy document

More information about these commands are available in the [plugin contract](../plugindev/contract.md) section of the plugin development guide.

## Plugin Lifecycle

## Plugin Discovery and Distribution
