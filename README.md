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

Use `make measure` (wrapper around `scripts/measure.sh`):

```bash
make measure lang=go
```

Run all languages + all 8 core benchmarks in one combined run:

```bash
make measure
```

Filter language/benchmark and change profile/iterations:

```bash
make measure lang=go profile=measure
make measure lang=go profile=test
make measure lang=go,c bench=binary-trees,mandelbrot iterations=10
make measure lang=rust iterations=3
```

`profile=measure` is the default.
`iterations=1` is the default.

Direct script usage (future KWA-compatible shape):

```bash
scripts/measure.sh profile=measure lang=go,c,cpp bench=binary-trees,mandelbrot iterations=10
```

Supported script args (`key=value` only):

- `profile=measure|test`
- `lang=<csv>`
- `bench=<csv>`
- `iterations=<int>`
- `gmt_dir=<path>` (optional)
- `uri=<path>` (optional)

When `lang` is omitted, all supported languages are used:

- `c`
- `cpp`
- `csharp`
- `dart`
- `erlang`
- `fsharp`
- `go`
- `haskell`
- `java`
- `lua`
- `nodejs`
- `ocaml`
- `perl`
- `php`
- `python`
- `ruby`
- `rust`
- `swift`

When `bench` is omitted, these 8 core benchmarks are used:

- `binary-trees`
- `fannkuch-redux`
- `k-nucleotide`
- `n-body`
- `regex-redux`
- `spectral-norm`
- `fasta`
- `mandelbrot`

Profile behavior:

- `measure`: uses canonical files `benchmarks/<lang>/<benchmark>.yml` and does **not** pass `--dev-no-sleeps`
- `test`: uses generated files `benchmarks/<lang>/<benchmark>_test.yml` and passes `--dev-no-sleeps`

Test profile inputs (`*_test.yml`):

- `binary-trees: 6`
- `fannkuch-redux: 6`
- `fasta: 100`
- `k-nucleotide: fasta-100.txt`
- `mandelbrot: 100`
- `n-body: 500`
- `regex-redux: fasta-100.txt`
- `spectral-norm: 500`

Validation errors:

- Invalid `profile`: must be `measure` or `test`
- Invalid `iterations`: must be integer `>= 1`
- Invalid `lang` / `bench`: must be known keys in supported lists
- Malformed CSV in `lang` / `bench`: empty items or spaces are rejected

## References

[0] [green-metrics-tool](https://github.com/green-coding-solutions/green-metrics-tool)

[1] [clbg](https://benchmarksgame-team.pages.debian.net/benchmarksgame/index.html)
