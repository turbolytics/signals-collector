package cmd

import (
	"context"
	"github.com/spf13/cobra"
	"github.com/turbolytics/collector/internal/collector"
	"github.com/turbolytics/collector/internal/collector/service"
	"github.com/turbolytics/collector/internal/config"
	"github.com/turbolytics/collector/internal/obs"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"log"
)

func NewRunCmd() *cobra.Command {
	var configDir string
	var otelExporter string

	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run collector daemon",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			// Create resource.
			res, err := obs.NewResource()
			if err != nil {
				panic(err)
			}

			// Create a meter provider.
			// You can pass this instance directly to your instrumented code if it
			// accepts a MeterProvider instance.
			meterProvider, err := obs.NewMeterProvider(obs.Exporter(otelExporter), res)
			if err != nil {
				panic(err)
			}

			// Handle shutdown properly so nothing leaks.
			defer func() {
				if err := meterProvider.Shutdown(context.Background()); err != nil {
					log.Println(err)
				}
			}()

			otel.SetMeterProvider(meterProvider)

			go otelruntime.Start(otelruntime.WithMeterProvider(meterProvider))

			logger, _ := zap.NewProduction()
			defer logger.Sync() // flushes buffer, if any

			if obs.Exporter(otelExporter) == obs.ExporterPrometheus {
				go obs.ServeMetrics(logger, ":12223")
			}

			logger.Info(
				"loading configs",
				zap.String("path", configDir),
			)

			// initialize all collectors in the path
			confs, err := config.NewFromDir(
				configDir,
				config.WithLogger(logger),
			)
			if err != nil {
				panic(err)
			}

			cs, err := collector.NewFromConfigs(
				confs,
				collector.WithLogger(logger),
			)
			if err != nil {
				panic(err)
			}

			logger.Info(
				"initialized collectors",
				zap.Int("num_collectors", len(cs)),
			)

			// schedule all collectors at their desired intervals
			s, err := service.NewService(
				cs,
				service.WithLogger(logger),
			)
			if err != nil {
				panic(err)
			}

			defer s.Shutdown()

			ctx := context.Background()
			if err := s.Run(ctx); err != nil {
				panic(err)
			}
		},
	}

	runCmd.Flags().StringVarP(&configDir, "config-dir", "c", "", "Path to config directory")
	runCmd.Flags().StringVarP(&otelExporter, "otel-exporter", "", "prometheus", "Opentelemetry exporter: 'console', prometheus")
	runCmd.MarkFlagRequired("config")

	return runCmd
}
