// Copyright ©2015 NAME HERE <EMAIL ADDRESS>
//
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
package cmd

import (
	"fmt"
	"log"
	"os/exec"
	"path"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
)

// process_yaml_queueCmd respresents the process_yaml_queue command
var process_yaml_queueCmd = &cobra.Command{
	Use:   "process_yaml_queue",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a Cli library for Go that empowers applications. This
application is a tool to generate the needed files to quickly create a Cobra
application.`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) < 1 {
			log.Panicf("Missing required arg: path to config.toml file")
		}

		var config tomlConfig
		if _, err := toml.DecodeFile(args[0], &config); err != nil {
			fmt.Println(err)
			return
		}
		for _, itemToProcess := range config.Items {

			log.Printf(
				"Painting: %v, Photo: %v, Output: %v",
				itemToProcess.Painting,
				itemToProcess.Photo,
				itemToProcess.OutputImagePath(config.OutputPath),
			)

			out, err := exec.Command(
				"th",
				"neural_style.lua",
				"-style_image",
				itemToProcess.Painting,
				"-content_image",
				itemToProcess.Photo,
				"-output_image",
				itemToProcess.OutputImagePath(config.OutputPath),
			).CombinedOutput()

			log.Printf("Command output: %v Command err: %v", string(out), err)
		}

	},
}

type tomlConfig struct {
	Items      []item
	OutputPath string
}

type item struct {
	Painting string
	Photo    string
}

func (i item) OutputImagePath(outputPath string) string {

	_, paintingFileName := path.Split(i.Painting)
	_, photoFileName := path.Split(i.Photo)

	log.Printf("painting: %v, photo: %v", paintingFileName, photoFileName)

	paintingNoExt := strings.Split(paintingFileName, ".")[0]
	photoNoExt := strings.Split(photoFileName, ".")[0]
	extension := path.Ext(paintingFileName)

	outputFilename := fmt.Sprintf("%v--%v%v", paintingNoExt, photoNoExt, extension)
	return path.Join(outputPath, outputFilename)

}

func init() {
	RootCmd.AddCommand(process_yaml_queueCmd)

	// Here you will define your flags and configuration settings

	// Cobra supports Persistent Flags which will work for this command and all subcommands
	// process_yaml_queueCmd.PersistentFlags().String("config", "", "Path to config toml file")
	// process_yaml_queueCmd.MarkPersistentFlagRequired("config")

	// Cobra supports local flags which will only run when this command is called directly
	// process_yaml_queueCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle" )

}
