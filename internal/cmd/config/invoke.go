package config

import (
	"context"
	"github.com/spf13/cobra"
	"github.com/turbolytics/collector/internal/collector"
	"github.com/turbolytics/collector/internal/config"
	"go.uber.org/zap"
)

func NewInvokeCmd() *cobra.Command {
	var configPath string

	var invokeCmd = &cobra.Command{
		Use:   "invoke",
		Short: "Invoke a signal collection",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			logger, _ := zap.NewProduction()
			defer logger.Sync() // flushes buffer, if any

			ctx := context.Background()
			config, err := config.NewFromFile(
				configPath,
				config.WithLogger(logger),
			)
			if err != nil {
				panic(err)
			}
			c, err := collector.New(
				config,
				collector.WithLogger(logger),
			)
			if err != nil {
				panic(err)
			}

			_, err = c.Invoke(ctx)
			if err != nil {
				panic(err)
			}
		},
	}

	invokeCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to config file")
	invokeCmd.MarkFlagRequired("config")

	return invokeCmd
}
