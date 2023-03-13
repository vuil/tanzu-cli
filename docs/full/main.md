# Tanzu CLI Documentation

VVV

## What is tanzu cli

## Important concept

plugin
context
target

## Tanzu CLI Plugin Architecture

explain a plugin

- standalone
- command tree
- passthrough
- provide metadata
- core cli responsible for lifecycle

## Command commands

There is a small set of commands that every plugin provides. These commands are
typically not invoked directly by CLI users; some are in fact hidden for that
reason. Below is a brief summary of these commands

`version`: provides basic version information about the plugin, likely the only common command of broad use to the CLI user.

`info`: provides metadata about the plugin that the CLI will use when presenting information about plugins or when performing lifecycle operations on them.

`post-install`: provide a means for a plugin to optionally implement some logic to be invoked right after a plugin is installed.

`generate-docs`: generate a tree of documentation markdown files for the commands the plugin provides, typically used by the CLI's generate-all-docs command to produce command documentation for all installed plugins

`lint`: validate the command name and arguments to flag any new terms unaccounted for in the CLI taxonomy document

More information about these commands are available in the plugin contract
section of the plugin development guide.

## Plugin Lifecycle

## Plugin Discovery and Distribution
