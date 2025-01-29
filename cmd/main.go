package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"item_compositiom_service/internal/config"
	"item_compositiom_service/internal/setup"
	"log"
)

var rootParams struct {
	ConfigPath string
}

func init() {
	rootCmd.
		PersistentFlags().
		StringVarP(
			&rootParams.ConfigPath,
			"config",
			"c",
			"config/config.yaml",
			"path to config file",
		)

	rootCmd.AddCommand(defaultConfigGenCmd)
}

var rootCmd = &cobra.Command{
	Use:   "app",
	Short: "Item Composition Service entrypoint",
	RunE: func(cmd *cobra.Command, _ []string) error {
		app, err := setup.Setup(rootParams.ConfigPath)
		if err != nil {
			return fmt.Errorf("setup application: %w", err)
		}

		app.Run()

		return app.Err()
	},
}

var defaultConfigGenCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate config file",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return config.GenerateDefaultConfig(rootParams.ConfigPath)
	},
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatalf("execute command: %s", err.Error())
	}
}
