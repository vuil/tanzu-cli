# Plugin Contract

Implication is that the names of these commands are reserved

plugin commands

- should adhere to style guide

- plugin-runtime help satisfy contract

plugin info
  plugin descriptor

plugin documentation generation
  generate-docs
plugin lint
plugin version
plugin autocompletion integration

- plugin expected to provide autocompletion support for its own commands

main cli will capture and passthrough the arguments it receives
plugin post-install

Since the primary means through which the CLI interacts with plugins is via plugin command invocation. The contract that each plugin has to satisfy is manifested in a set of commands it is expected to implement.

`generate-docs`

`lint`

`version`

`info`

`post-install`
