# Towards a More Accurate Understanding of Programming Language Energy Efficiency

Agent docs: [AGENTS.md](AGENTS.md) for repo overview and [kwa/AGENTS.md](kwa/AGENTS.md) for deep KWA architecture/workflows.

## Linux Setup (Ubuntu 22.04/24.04)

Use `make setup` to bootstrap the full local environment (Linux only):

```bash
make setup
```

`make setup`:

- checks/install required base tools (`git`, `curl`, `make`, `gcc`, etc.)
- installs Docker if missing and enables the daemon
- installs/ensures Python `3.12`
- installs/ensures the Go version required by `kwa/go.mod`
- clones GMT into this repo at `./green-metrics-tool`
- runs GMT `install_linux.sh` non-interactively with local URLs
- attempts full metric-provider dependency setup, retrying with best-effort fallbacks for hardware-specific providers if needed

Important notes:

- This setup is intended for Ubuntu `22.04` and `24.04` only.
- `sudo` is required.
- If your user is newly added to the `docker` group, you may need to relogin (or run `newgrp docker`) before running Docker without sudo.
- If `./green-metrics-tool` already exists, setup prompts whether to overwrite it.
- DB defaults are sourced from `kwa/.env.example` (notably `DATABASE_PASSWORD`).

## Uninstall

Use `make uninstall` for safe local teardown:

```bash
make uninstall
```

`make uninstall`:

- always asks whether to remove DB/data volume
- stops/removes GMT containers (best effort)
- runs `docker system prune` (best effort)
- removes local artifacts:
  - `kwa/build`
  - `.gocache`
  - `.gocache_local`
  - `.gomodcache`
  - `./green-metrics-tool`
- prompts (Linux) whether to remove pre-install requirements and Docker packages

Notes:

- This uninstall flow is Linux-oriented and destructive.
- `sudo` may be required for package/sudoers cleanup.

## KWA Build/Run

Build KWA binary into `kwa/build/kwa`:

```bash
make kwa-build
```

Run KWA directly from source (`kwa/cmd/main.go`):

```bash
make kwa-run
```

## Run Benchmarks

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

## Direct Runner Script

Direct script usage (future KWA-compatible shape):

```bash
scripts/measure.sh profile=measure lang=go,c,cpp bench=binary-trees,mandelbrot iterations=10
```

Supported script args (`key=value` only):

- `profile=measure|test`
- `lang=<csv>`
- `bench=<csv>`
- `iterations=<int>`
- `gmt_dir=<path>` (optional, default: `./green-metrics-tool`)
- `uri=<path>` (optional, default: repo root)

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

## References

[0] [green-metrics-tool](https://github.com/green-coding-solutions/green-metrics-tool)

[1] [clbg](https://benchmarksgame-team.pages.debian.net/benchmarksgame/index.html)
