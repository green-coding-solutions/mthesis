# Path to your Green Metrics Tool installation
GMT_DIR := /Users/brandao/green-metrics-tool
VENV := $(GMT_DIR)/venv/bin/activate
URI := /Users/brandao/mthesis

# Base command to run Green Metrics Tool
RUN_GMT = source $(VENV) && \
    python3 $(GMT_DIR)/runner.py \
        --uri $(URI) \
        --name run$(lang) \
        --filename ./benchmarks/$(lang)/k-nucleotide.yml \
        --dev-no-sleeps \
        --iterations 1 \
        --docker-prune

# General run target
run:
	@if [ -z "$(lang)" ]; then \
		echo "Please provide the language, e.g., make run lang=go"; \
		exit 1; \
	fi
	@$(RUN_GMT)
