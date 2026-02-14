# Towards a More Accurate Understanding of Programming Language Energy Efficiency

## Local Testing

This repository is wired to run with a local Green Metrics Tool installation.

### Prerequisites

- Green Metrics Tool cloned locally from [green-metrics-tool](https://github.com/green-coding-solutions/green-metrics-tool) (default in `Makefile`: `/Users/brandao/green-metrics-tool`)
- GMT Python virtual environment available at `$(GMT_DIR)/venv`
- Docker running

### Configure paths

Check these variables in `Makefile` and adjust if your paths differ:

- `GMT_DIR`
- `URI`

### Run benchmarks

Use:

```bash
make run lang=go
```

or:

```bash
make run lang=rust
```

The `run` target executes:

- `benchmarks/<lang>/default.yml`
- `benchmarks/<lang>/fasta.yml`
- `benchmarks/<lang>/mandelbrot.yml`

If `lang` is missing, `make` exits with:

```bash
Please provide the language, e.g., make run lang=go
```

## References

[0] [green-metrics-tool](https://github.com/green-coding-solutions/green-metrics-tool)

[1] [clbg](https://benchmarksgame-team.pages.debian.net/benchmarksgame/index.html)
