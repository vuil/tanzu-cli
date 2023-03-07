# Tanzu CLI deprecation policy

Tanzu CLI is comprised of several different commands and plugins.
Sometimes, a release might remove flags or CLI commands (collectively
"CLI elements").

This document sets out the deprecation policy for Tanzu CLI.

VVV
Add details for deprecation policy

## How deprecate CLI functionality

To deprecate a particular piece of CLI functionality,

1. Deprecated CLI elements must display warnings when used.
1. The warning message should include a functional alternative to the
   deprecated command or flag if they exist.
1. The warning message should include the release for when the command/flag
   will be removed.
1. The deprecation should be documented in the Release notes to make users
   aware of the changes.

See [file](https://github.com/vmware-tanzu/tanzu-plugin-runtime/blob/main/command/deprecation.go)
for the helper functions that can be used to deprecate a command.

Example usage to deprecate a command `foo`:

```golang
import "github.com/vmware-tanzu/tanzu-plugin-runtime/command"
//...
command.DeprecateCommand(fooCmd, "1.5.0", "bar")
```

Running the `foo` command will display the following:

```console
Command "foo" is deprecated, will be removed in version as early as "1.5.0". Use "bar" instead.
```

Similarly, to deprecate a flag --use-grouping in a `describe` command:

```golang
import "github.com/vmware-tanzu/tanzu-plugin-runtime/command"
//...
command.DeprecateFlagWithAlternative(describeCmd, "use-grouping", "1.6.0", "--show-group-members")
```
