/*
Copyright © 2019 Red Hat Inc.

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
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	rmiFlags struct {
		deleteAll   bool
		forceDelete bool
	}
)

var rmiCmd = &cobra.Command{
	Use:   "rmi",
	Short: "Remove one or more toolbox images",
	Run: func(cmd *cobra.Command, args []string) {
		rmi()
	},
}

func init() {
	rootCmd.AddCommand(rmiCmd)

	flags := rmiCmd.Flags()
	flags.BoolVarP(&rmiFlags.deleteAll, "all", "a", false, "Remove all toolbox containers")
	flags.BoolVarP(&rmiFlags.forceDelete, "force", "f", false, "Force the removal of running and paused toolbox containers")
}

func rmi() {
	fmt.Println("function 'rmi'")
}
