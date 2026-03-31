package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	appexport "mthesis/kwa/internal/app/export"
	"mthesis/kwa/internal/constant"

	"github.com/spf13/cobra"
)

type interactiveRunner func(
	ctx context.Context,
	execute executeRequestFunc,
	stdout io.Writer,
	stderr io.Writer,
) error

type rootDependencies struct {
	execute executeRequestFunc
	runTUI  interactiveRunner
}

// executeRequestFunc bridges CLI/TUI input handling with application execution.
type executeRequestFunc func(context.Context, appexport.Request) error

// Execute runs the Cobra root command and exits with code 1 on command errors.
func Execute() {
	rootCmd := newRootCmd(defaultDependencies())
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(rootCmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func defaultDependencies() rootDependencies {
	executor := appexport.NewExecutor()
	return rootDependencies{
		execute: executor.Execute,
		runTUI:  runInteractive,
	}
}

func newRootCmd(deps rootDependencies) *cobra.Command {
	if deps.execute == nil {
		deps.execute = appexport.NewExecutor().Execute
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

func newBatchCmd(execute executeRequestFunc) *cobra.Command {
	var (
		batchSize int
		outPath   string
	)

	batchCmd := &cobra.Command{
		Use:   "batch",
		Short: "Export measurements in paginated batches",
		RunE: func(cmd *cobra.Command, _ []string) error {
			req := appexport.Request{
				Mode:      constant.ExportModeBatch,
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

	batchCmd.Flags().IntVar(&batchSize, "batch-size", constant.DefaultBatchSize, "rows per batch")
	batchCmd.Flags().StringVar(&outPath, "out", constant.DefaultOutPath, "output CSV file path")
	return batchCmd
}

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
