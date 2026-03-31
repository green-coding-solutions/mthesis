package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"mthesis/kwa/internal/api"
	"mthesis/kwa/internal/config"
	"mthesis/kwa/internal/data"
	"mthesis/kwa/internal/service"
)

func main() {
	mode := flag.String("mode", "batch", "export mode: batch or by-id")
	batchSize := flag.Int("batch-size", 100, "batch size for batch mode")
	runID := flag.String("run-id", "", "run ID for by-id mode")
	outPath := flag.String("out", "results/measurements.csv", "output CSV file path")
	flag.Parse()

	cfg, err := config.LoadDatabaseConfig()
	if err != nil {
		log.Fatalf("load database config: %v", err)
	}

	dataService, err := data.New(cfg)
	if err != nil {
		log.Fatalf("init data service: %v", err)
	}
	defer func() {
		if err := dataService.Close(); err != nil {
			log.Printf("close data service: %v", err)
		}
	}()

	exporterService := service.NewExporterService(service.NewParserService(), dataService)
	cliAPI := api.NewCLIHandler(exporterService)
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		log.Fatalf("create output directory for %q: %v", *outPath, err)
	}
	outFile, err := os.Create(*outPath)
	if err != nil {
		log.Fatalf("create output file %q: %v", *outPath, err)
	}
	defer func() {
		if err := outFile.Close(); err != nil {
			log.Printf("close output file %q: %v", *outPath, err)
		}
	}()

	ctx := context.Background()
	switch *mode {
	case "batch":
		if err := cliAPI.ExportBatch(ctx, outFile, *batchSize); err != nil {
			log.Fatalf("batch export failed: %v", err)
		}
	case "by-id":
		if err := cliAPI.ExportByID(ctx, outFile, *runID); err != nil {
			log.Fatalf("single-run export failed: %v", err)
		}
	default:
		log.Fatalf("invalid mode %q: use batch or by-id", *mode)
	}

	fmt.Fprintf(os.Stderr, "export finished: %s\n", *outPath)
}
