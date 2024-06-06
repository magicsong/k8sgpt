/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package manifest

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	apply     = "Apply"
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
		m, err := NewManifester("", 1)
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
		fmt.Println(str)
	},
}

var (
	kubernetesConfigFlags = genericclioptions.NewConfigFlags(false)

	requireConfirmation = ManifestCmd.Flags().Bool("require-confirmation", true, "Whether to require confirmation before executing the command. Defaults to true.")
	raw                 = ManifestCmd.Flags().Bool("raw", false, "Prints the raw YAML output immediately. Defaults to false.")
	k8sOpenAPIURL       = ManifestCmd.Flags().String("k8s-openapi-url", "", "The URL to a Kubernetes OpenAPI spec. Only used if use-k8s-api ManifestCmd.Flags() is true.")
	debug               = ManifestCmd.Flags().Bool("debug", false, "Whether to print debug logs. Defaults to false.")
)
