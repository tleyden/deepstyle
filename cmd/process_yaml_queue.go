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
	Short: "Process images specified in toml queue",
	Long:  `Process images specified in toml queue.  See examples/toml_queue for example toml file`,
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
