// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin/buildinfo"
)

var descriptor = plugin.PluginDescriptor{
	Name:        "sample",
	Description: "sample plugin to test command mapping",
	Target:      types.TargetGlobal,
	Version:     buildinfo.Version,
	BuildSHA:    buildinfo.SHA,
	Group:       plugin.ManageCmdGroup,
	CommandMap:  []plugin.CommandMapEntry{},
}

func main() {
	p, err := plugin.NewPlugin(&descriptor)
	if err != nil {
		log.Fatal(err, "")
	}
	p.AddCommands(
		newEchoCmd(),
		newShoutCmd(),
		newDeeperCmd(),
	)
	if err := p.Execute(); err != nil {
		os.Exit(1)
	}
}

func newEchoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "echo MESSAGE",
		Short:  "echo something",
		Args:   cobra.ExactArgs(1),
		Hidden: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(args[0])
			return nil
		},
	}
	return cmd
}

func newShoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "shout MESSAGE",
		Short:  "shout something",
		Args:   cobra.ExactArgs(1),
		Hidden: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			ic := plugin.GetInvocationContext()
			fmt.Printf("%s!!\nInvocation Context:\n% #v\n", args[0], ic)
			return nil
		},
	}
	return cmd
}

func newYellCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "yell MESSAGE",
		Short:  "yell something",
		Args:   cobra.ExactArgs(1),
		Hidden: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("%s weeeee!!!!!\n", args[0])
			return nil
		},
	}
	return cmd
}

func newDeeperCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deeper",
		Short:   "deeper commands",
		Aliases: []string{"deep"},
	}

	cmd.AddCommand(newYellCmd())

	return cmd
}
