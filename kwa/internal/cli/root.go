package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

type interactiveRunner func(ctx context.Context, execute ExportExecutor, stdout, stderr io.Writer) error

type rootDependencies struct {
	execute ExportExecutor
	runTUI  interactiveRunner
}

// Execute runs the Cobra root command and exits with code 1 on command errors.
func Execute() {
	rootCmd := newRootCmd(defaultDependencies())
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(rootCmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func defaultDependencies() rootDependencies {
	return rootDependencies{
		execute: runExport,
		runTUI:  runInteractive,
	}
}

func newRootCmd(deps rootDependencies) *cobra.Command {
	if deps.execute == nil {
		deps.execute = runExport
	}
	if deps.runTUI == nil {
		deps.runTUI = runInteractive
	}

	rootCmd := &cobra.Command{
		Use:          "kwa",
		Short:        "KWA CLI for exporting green metrics",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return deps.runTUI(cmd.Context(), deps.execute, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	rootCmd.AddCommand(newBatchCmd(deps.execute))
	rootCmd.AddCommand(newByIDCmd(deps.execute))

	return rootCmd
}

func newBatchCmd(execute ExportExecutor) *cobra.Command {
	var (
		batchSize int
		outPath   string
	)

	batchCmd := &cobra.Command{
		Use:   "batch",
		Short: "Export measurements in paginated batches",
		RunE: func(cmd *cobra.Command, _ []string) error {
			req := ExportRequest{
				Mode:      ExportModeBatch,
				BatchSize: batchSize,
				OutPath:   outPath,
			}
			if err := execute(cmd.Context(), req); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "export finished: %s\n", outPath)
			return nil
		},
	}

	batchCmd.Flags().IntVar(&batchSize, "batch-size", DefaultBatchSize, "rows per batch")
	batchCmd.Flags().StringVar(&outPath, "out", DefaultOutPath, "output CSV file path")
	return batchCmd
}

func newByIDCmd(execute ExportExecutor) *cobra.Command {
	var (
		runID   string
		outPath string
	)

	byIDCmd := &cobra.Command{
		Use:     "by-id",
		Aliases: []string{"byID"},
		Short:   "Export measurements for one run ID",
		RunE: func(cmd *cobra.Command, _ []string) error {
			req := ExportRequest{
				Mode:    ExportModeByID,
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
	byIDCmd.Flags().StringVar(&outPath, "out", DefaultOutPath, "output CSV file path")
	_ = byIDCmd.MarkFlagRequired("run-id")

	return byIDCmd
}
