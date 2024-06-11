/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package manifest

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const (
	dontApply = "Don't Apply"
	reprompt  = "Reprompt"
)

// ManifestCmd represents the manifest command
var ManifestCmd = &cobra.Command{
	Use:   "manifest",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		runManifestCmd(args)
	},
}

var (
	apply *bool
)

func init() {
	apply = ManifestCmd.Flags().Bool("apply", false, "Whether to apply the generated manifest. Defaults to false.")
}
func runManifestCmd(args []string) {
	m, err := NewManifester("", 1, true)
	if err != nil {
		color.Red("Error: %v", err)
		os.Exit(1)
	}
	defer m.Close()
	if len(args) == 0 {
		fmt.Println("prompt must be provided")
		os.Exit(1)
	}
	str, err := m.GenerateManifest(args[0], false)
	if err != nil {
		color.Red("Error: %v", err)
		os.Exit(1)
	}
	print(str)
	if *apply {
		fmt.Println("will apply manifest above to k8s api server")
		if err := m.ApplyManifest(str); err != nil {
			color.Red("Apply manifest Error: %v", err)
			os.Exit(1)
		}
	}
}
