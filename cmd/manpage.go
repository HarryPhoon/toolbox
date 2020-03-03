/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

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
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/containers/toolbox/pkg/utils"

	"github.com/cpuguy83/go-md2man/md2man"
	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

var (
	manpageFlags struct {
		inputFolder  string
		outputFolder string
	}
)

// manpageCmd represents the manpage command
var manpageCmd = &cobra.Command{
	Use:   "manpage",
	Short: "A tool for generating manpages out of markdown files",
	Run: func(cmd *cobra.Command, args []string) {
		inputFolderPath, err := filepath.Abs(manpageFlags.inputFolder)
		if err != nil {
			logrus.Fatalf("Could not get absolute path to the input folder: %v", err)
		}
		logrus.Debugf("Input folder: %s", inputFolderPath)

		fileList, err := ioutil.ReadDir(inputFolderPath)
		if err != nil {
			logrus.Fatalf("Could not read content of the input folder: %v", err)
		}

		outputFolderPath, err := filepath.Abs(manpageFlags.outputFolder)
		if err != nil {
			logrus.Fatalf("Could not get absolute path to the output folde: %v", err)
		}
		logrus.Debugf("Output folder: %s", outputFolderPath)

		if !utils.PathExists(outputFolderPath) {
			logrus.Infof("Output folder %s does not exist. Creating it..", outputFolderPath)
			err = os.MkdirAll(outputFolderPath, 0775)
			if err != nil {
				logrus.Fatalf("Could not create output folder %s: %v", outputFolderPath, err)
			}
		}

		for _, file := range fileList {
			inputName := file.Name()
			inputFile := fmt.Sprintf("%s/%s", inputFolderPath, inputName)
			matched, err := regexp.MatchString(`^toolbox[.-]?[a-z-]*\.1\.md`, inputName)
			if err != nil {
				logrus.Errorf("There was an error while matching regexp: %v", err)
			}

			if matched {
				outputName := strings.TrimSuffix(inputName, ".md")
				outputFile := fmt.Sprintf("%s/%s", outputFolderPath, outputName)
				logrus.Infof("File %s matches. Creating manpage %s", inputName, outputName)

				doc, err := ioutil.ReadFile(inputFile)
				if err != nil {
					logrus.Errorf("Could not read file %s: %v", inputName, err)
				}

				out := md2man.Render(doc)

				err = ioutil.WriteFile(outputFile, out, file.Mode().Perm())
				if err != nil {
					logrus.Errorf("Could not create/write to file %s: %v", outputName, err)
				}
			}
		}
	},
}

func init() {
	generateCmd.AddCommand(manpageCmd)

	manpageCmd.Flags().StringVar(&manpageFlags.inputFolder, "input-folder", "", "Folder that holds markdown files with the documentation")
	manpageCmd.MarkFlagRequired("input-folder")
	manpageCmd.Flags().StringVar(&manpageFlags.outputFolder, "output-folder", "", "Folder where the generated manpages will be saved (it is created when it does not exist")
	manpageCmd.MarkFlagRequired("output-folder")
}
