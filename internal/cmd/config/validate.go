package config

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/turbolytics/collector/internal/config"
)

func NewValidateCmd() *cobra.Command {
	var configPath string

	var validateCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validate a signal collector config",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(configPath)
			_, err := config.NewFromFile(
				configPath,
				config.WithJustValidation(true),
			)
			if err != nil {
				panic(err)
			}

			fmt.Println("VALID=true")
		},
	}

	validateCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to config file")
	validateCmd.MarkFlagRequired("config")

	return validateCmd
}
