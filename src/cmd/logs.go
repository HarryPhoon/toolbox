/*
 * Copyright © 2019 – 2020 Red Hat Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/containers/toolbox/pkg/podman"
	"github.com/containers/toolbox/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	logsFlags struct {
		listInits bool
		init      int
		follow    bool
		tail      int
	}
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show logs from the initialization of a toolbox",
	RunE:  logs,
	Args:  cobra.ExactArgs(1),
}

func init() {
	flags := rmCmd.Flags()

	flags.BoolVar(&logsFlags.listInits,
		"list-inits",
		false,
		"Show list of all container initializations")

	flags.IntVarP(&logsFlags.init,
		"init",
		"i",
		0,
		"Show logs of a specific initialization")

	flags.BoolVarP(&logsFlags.follow,
		"follow",
		"f",
		false,
		"Follow log output")

	flags.IntVar(&logsFlags.tail,
		"tail",
		-1,
		"Output the specified number of LINES at the end of the logs")

	rmCmd.SetHelpFunc(logsHelp)
	rootCmd.AddCommand(logsCmd)
}

func logs(cmd *cobra.Command, args []string) error {
	if utils.IsInsideContainer() {
		if !utils.IsInsideToolboxContainer() {
			return errors.New("this is not a toolbox container")
		}

		if _, err := utils.ForwardToHost(); err != nil {
			return err
		}

		return nil
	}

	var container string
	container = args[0]

	if logsFlags.listInits {
		podman.Logs(container)
	}

	if logs.Flags.follow {

	}

	return nil
}

func logsHelp(cmd *cobra.Command, args []string) {
	if utils.IsInsideContainer() {
		if !utils.IsInsideToolboxContainer() {
			fmt.Fprintf(os.Stderr, "Error: this is not a toolbox container\n")
			return
		}

		if _, err := utils.ForwardToHost(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			return
		}

		return
	}

	if err := utils.ShowManual("toolbox-logs"); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return
	}
}
