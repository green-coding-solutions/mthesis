package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	appexport "mthesis/kwa/internal/app/export"
	appmeasure "mthesis/kwa/internal/app/measure"
	"mthesis/kwa/internal/constant"

	"github.com/spf13/cobra"
)

type interactiveRunner func(
	ctx context.Context,
	executeExport executeRequestFunc,
	executeMeasure executeMeasureFunc,
	stdout io.Writer,
	stderr io.Writer,
) error

type rootDependencies struct {
	execute        executeRequestFunc
	executeMeasure executeMeasureFunc
	runTUI         interactiveRunner
}

// executeRequestFunc bridges CLI/TUI input handling with application execution.
type executeRequestFunc func(context.Context, appexport.Request) error

// executeMeasureFunc bridges CLI/TUI measure input handling with the measure application executor.
type executeMeasureFunc func(context.Context, appmeasure.Request) error

// Execute runs the Cobra root command and exits with code 1 on command errors.
func Execute() {
	rootCmd := newRootCmd(defaultDependencies())
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(rootCmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

// defaultDependencies wires the production executor and interactive runner.
func defaultDependencies() rootDependencies {
	exportExecutor := appexport.NewExecutor()
	measureExecutor := appmeasure.NewExecutor(exportExecutor.Execute)
	return rootDependencies{
		execute:        exportExecutor.Execute,
		executeMeasure: measureExecutor.Execute,
		runTUI:         runInteractive,
	}
}

// newRootCmd builds the root CLI command and registers interactive and subcommand flows.
func newRootCmd(deps rootDependencies) *cobra.Command {
	if deps.execute == nil {
		deps.execute = appexport.NewExecutor().Execute
	}
	if deps.executeMeasure == nil {
		deps.executeMeasure = appmeasure.NewExecutor(deps.execute).Execute
	}
	if deps.runTUI == nil {
		deps.runTUI = runInteractive
	}

	rootCmd := &cobra.Command{
		Use:          "kwa",
		Short:        "KWA CLI for measuring and exporting green metrics",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return deps.runTUI(cmd.Context(), deps.execute, deps.executeMeasure, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	rootCmd.AddCommand(newBatchCmd(deps.execute))
	rootCmd.AddCommand(newByIDCmd(deps.execute))

	return rootCmd
}

// newBatchCmd builds the non-interactive batch export command.
func newBatchCmd(execute executeRequestFunc) *cobra.Command {
	var (
		batchSize int
		outPath   string
		fromInput string
		toInput   string
	)

	batchCmd := &cobra.Command{
		Use:   "batch",
		Short: "Export measurements in paginated batches",
		RunE: func(cmd *cobra.Command, _ []string) error {
			timeRange, err := appexport.ParseTimeRange(fromInput, toInput)
			if err != nil {
				return err
			}

			req := appexport.Request{
				Mode:      constant.ExportModeBatch,
				BatchSize: batchSize,
				OutPath:   outPath,
				TimeRange: timeRange,
			}
			if err := execute(cmd.Context(), req); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "export finished: %s\n", outPath)
			return nil
		},
	}

	batchCmd.Flags().IntVar(&batchSize, "batch-size", constant.DefaultBatchSize, "rows per batch")
	batchCmd.Flags().StringVar(&outPath, "out", constant.DefaultOutPath, "output CSV file path")
	batchCmd.Flags().StringVar(&fromInput, "from", "", "start timestamp (YYYY-MM-DD or YYYY-MM-DD HH:MM:SS)")
	batchCmd.Flags().StringVar(&toInput, "to", "", "end timestamp (YYYY-MM-DD or YYYY-MM-DD HH:MM:SS)")
	return batchCmd
}

// newByIDCmd builds the non-interactive single-run export command.
// It accepts only run ID and output path flags, dispatches one by-id request,
// writes a success line, and returns command execution errors.
func newByIDCmd(execute executeRequestFunc) *cobra.Command {
	var (
		runID   string
		outPath string
	)

	byIDCmd := &cobra.Command{
		Use:     "by-id",
		Aliases: []string{"byID"},
		Short:   "Export measurements for one run ID",
		RunE: func(cmd *cobra.Command, _ []string) error {
			req := appexport.Request{
				Mode:    constant.ExportModeByID,
				RunID:   runID,
				OutPath: outPath,
			}
			if err := execute(cmd.Context(), req); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "export finished: %s\n", outPath)
			return nil
		},
	}

	byIDCmd.Flags().StringVar(&runID, "run-id", "", "run ID to export")
	byIDCmd.Flags().StringVar(&outPath, "out", constant.DefaultOutPath, "output CSV file path")
	_ = byIDCmd.MarkFlagRequired("run-id")

	return byIDCmd
}
