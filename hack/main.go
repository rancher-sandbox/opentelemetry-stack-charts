package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path"
	"slices"
	"time"

	"github.com/lmittmann/tint"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func BuildRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "hack",
	}
	cmd.AddCommand(BuildCollectCommand())
	cmd.AddCommand(BuildCompareCommand())
	return cmd
}

func BuildCollectCommand() *cobra.Command {
	var (
		prometheusEndpoint string
		targetDirectory    string
	)
	cmd := &cobra.Command{
		Use:   "collect",
		Short: "collects all prometheus metrics from remote",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := os.Mkdir(targetDirectory, 0777); err != nil {
				return err
			}

			return collectAndWrite(prometheusEndpoint, targetDirectory)
		},
	}
	cmd.Flags().StringVarP(&prometheusEndpoint, "endpoint", "e", "http://localhost:9090", "prometheus endpoint")
	cmd.Flags().StringVarP(&targetDirectory, "dir", "d", "out", "output directory of collected information")
	return cmd
}

func BuildCompareCommand() *cobra.Command {
	var (
		sourceDir string
		targetDir string
	)
	cmd := &cobra.Command{
		Use:   "compare",
		Short: "compares two sets of prometheus metrics",
		Long:  "dumps the metrics not found in source, and metrics not found in target",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := readAndCompare(sourceDir, targetDir); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&sourceDir, "source", "s", "prometheus", "source for comparison")
	cmd.Flags().StringVarP(&targetDir, "target", "t", "otel", "target for comparison")

	return cmd
}

func init() {
	w := os.Stderr
	// Set global logger with custom options
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
		}),
	))
}

func main() {
	cmd := BuildRootCmd()
	logger := slog.Default()
	if err := cmd.Execute(); err != nil {
		logger.With("err", err).Error("failed to run command")
	}
}

const (
	allMetrics = "metric_names.json"
	metricJobs = "metric_jobs.json"
)

func query(client v1.API, query string, at time.Time, outFile string) error {
	logger := slog.Default().With("query", query)
	ctx := context.Background()
	logger.Info("running query")
	result, warnings, err := client.Query(ctx, query, at)
	if err != nil {
		logger.With("err", err).Error("failed to query instance")
		return err
	}

	if len(warnings) > 0 {
		for _, warn := range warnings {
			logger.Warn(warn)
		}
	}
	logger.Info("got result")

	data, err := json.Marshal(result)
	if err != nil {
		logger.With("err", err).Error("failed to marshal payload")
		return err
	}

	if err := os.WriteFile(outFile, data, 0777); err != nil {
		logger.With("err", err, "file", outFile).Error("failed to persist result")
		return err
	}
	return nil
}

func collectAndWrite(prometheusEndp, targetDirectory string) error {
	logger := slog.Default()
	logger.With("endpoint", prometheusEndp).Info("creating client")
	client, err := api.NewClient(api.Config{
		Address: prometheusEndp,
	})
	if err != nil {
		return err
	}
	v1api := v1.NewAPI(client)

	if err := query(v1api, "{__name__!=\"\"}", time.Now(), path.Join(targetDirectory, allMetrics)); err != nil {
		return err
	}
	return nil
}

func readAndCompare(sourceDir, targetDir string) error {
	logger := slog.Default()
	d1, err := read(sourceDir, allMetrics)
	if err != nil {
		return err
	}

	d2, err := read(targetDir, allMetrics)
	if err != nil {
		return err
	}

	// FIXME: need to determine which type the dumped series are
	m1 := model.Vector{}
	m2 := model.Vector{}

	if err := json.Unmarshal(d1, &m1); err != nil {
		return err
	}
	if err := json.Unmarshal(d2, &m2); err != nil {
		return err
	}

	logger.With("series-count", len(m1)).Info("got series from source")
	logger.With("series-count", len(m2)).Info("got series from target")

	sourceNames := map[string]struct{}{}
	targetNames := map[string]struct{}{}
	for _, dp := range m1 {
		name := dp.Metric["__name__"]
		sourceNames[string(name)] = struct{}{}
	}

	for _, dp := range m2 {
		name := dp.Metric["__name__"]
		targetNames[string(name)] = struct{}{}
	}

	notInTarget, notInSource := lo.Difference(lo.Keys(sourceNames), lo.Keys(targetNames))
	logger.With("target", targetDir, "count", len(notInTarget)).Info("series not in target")
	logger.With("source", sourceDir, "count", len(notInSource)).Info("series not in source")

	slices.Sort(notInSource)
	slices.Sort(notInTarget)

	notTargetData, err := json.Marshal(notInTarget)
	if err != nil {
		return err
	}
	notInSourceData, err := json.Marshal(notInSource)
	if err != nil {
		return err
	}

	if err := os.WriteFile("not_in_target.json", notTargetData, 0777); err != nil {
		return err
	}

	if err := os.WriteFile("not_in_source.json", notInSourceData, 0777); err != nil {
		return err
	}

	return nil
}

func read(dir, file string) ([]byte, error) {
	p := path.Join(dir, file)
	logger := slog.With("path", p)
	logger.Info("reading contents")
	data, err := os.ReadFile(p)
	if err != nil {
		logger.Error("failed to read contents")
		return nil, err
	}
	return data, nil
}
