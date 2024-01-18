/*
Copyright (c) 2018 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"os"

	"github.com/prometheus/common/version"
	"github.com/spf13/cobra"
	"k8s.io/component-base/cli"
	"k8s.io/component-base/logs"
)

var rootCmd = &cobra.Command{
	Use: "autoheal",
	Long: "The auto-heal service receives alert notifications from the Prometheus alert manager, " +
		"decides which healing action to execute based on a set of rules, and executes it. Healing " +
		"actions can be AWX jobs or Kubernetes batch jobs.",
}

var appName = "autoheal"

func init() {
	logs.AddFlags(rootCmd.PersistentFlags())
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(versionCommand())
}

func main() {
	logs.InitLogs()
	os.Exit(cli.Run(rootCmd))
}

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Fprintln(os.Stdout, version.Print(appName))
		},
	}
}
