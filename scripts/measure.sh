#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEFAULT_GMT_DIR="${REPO_ROOT}/green-metrics-tool"
DEFAULT_URI="${REPO_ROOT}"

LANGUAGES=(c cpp csharp dart erlang fsharp go haskell java lua nodejs ocaml perl php python ruby rust swift)
BENCHMARKS=(binary-trees fannkuch-redux k-nucleotide n-body regex-redux spectral-norm fasta mandelbrot)

profile="measure"
lang_csv=""
bench_csv=""
iterations="1"
gmt_dir="$DEFAULT_GMT_DIR"
uri="$DEFAULT_URI"

usage() {
  cat <<'USAGE'
Usage:
  scripts/measure.sh [profile=<measure|test>] [lang=<csv>] [bench=<csv>] [iterations=<n>] [gmt_dir=<path>] [uri=<path>]

Examples:
  scripts/measure.sh
  scripts/measure.sh profile=test lang=go,c bench=binary-trees,mandelbrot iterations=10
USAGE
}

die() {
  echo "Error: $*" >&2
  exit 1
}

contains_in_array() {
  local needle="$1"
  shift
  local item
  for item in "$@"; do
    if [ "$item" = "$needle" ]; then
      return 0
    fi
  done
  return 1
}

validate_csv_format() {
  local csv="$1"
  local label="$2"

  if [ -z "$csv" ]; then
    return 0
  fi

  case "$csv" in
    *,|,*|*,,*)
      die "Malformed $label CSV '$csv'. Use comma-separated values without empty items."
      ;;
  esac

  if printf '%s' "$csv" | grep -q '[[:space:]]'; then
    die "Malformed $label CSV '$csv'. Do not include spaces."
  fi
}

for arg in "$@"; do
  if [ "$arg" = "-h" ] || [ "$arg" = "--help" ]; then
    usage
    exit 0
  fi

  case "$arg" in
    *=*)
      key="${arg%%=*}"
      value="${arg#*=}"
      ;;
    *)
      die "Invalid argument '$arg'. Use key=value."
      ;;
  esac

  case "$key" in
    profile) profile="$value" ;;
    lang) lang_csv="$value" ;;
    bench) bench_csv="$value" ;;
    iterations) iterations="$value" ;;
    gmt_dir) gmt_dir="$value" ;;
    uri) uri="$value" ;;
    *)
      die "Unknown argument '$key'. Supported keys: profile lang bench iterations gmt_dir uri"
      ;;
  esac
done

if [ "$profile" != "measure" ] && [ "$profile" != "test" ]; then
  die "Invalid profile '$profile'. Use profile=measure or profile=test"
fi

if ! printf '%s' "$iterations" | grep -Eq '^[0-9]+$'; then
  die "Invalid iterations '$iterations'. Use a positive integer (e.g., iterations=1)"
fi

if [ "$iterations" -lt 1 ]; then
  die "Invalid iterations '$iterations'. Use a value >= 1"
fi

validate_csv_format "$lang_csv" "lang"
validate_csv_format "$bench_csv" "bench"

target_languages=()
if [ -z "$lang_csv" ]; then
  target_languages=("${LANGUAGES[@]}")
else
  IFS=',' read -r -a requested_languages <<< "$lang_csv"
  for token in "${requested_languages[@]}"; do
    if ! contains_in_array "$token" "${LANGUAGES[@]}"; then
      die "Unknown language '$token'. Allowed: ${LANGUAGES[*]}"
    fi
  done
  for token in "${LANGUAGES[@]}"; do
    if contains_in_array "$token" "${requested_languages[@]}"; then
      target_languages+=("$token")
    fi
  done
fi

target_benchmarks=()
if [ -z "$bench_csv" ]; then
  target_benchmarks=("${BENCHMARKS[@]}")
else
  IFS=',' read -r -a requested_benchmarks <<< "$bench_csv"
  for token in "${requested_benchmarks[@]}"; do
    if ! contains_in_array "$token" "${BENCHMARKS[@]}"; then
      die "Unknown benchmark '$token'. Allowed: ${BENCHMARKS[*]}"
    fi
  done
  for token in "${BENCHMARKS[@]}"; do
    if contains_in_array "$token" "${requested_benchmarks[@]}"; then
      target_benchmarks+=("$token")
    fi
  done
fi

if [ "${#target_languages[@]}" -eq 0 ]; then
  die "No languages selected."
fi

if [ "${#target_benchmarks[@]}" -eq 0 ]; then
  die "No benchmarks selected."
fi

profile_suffix=""
if [ "$profile" = "test" ]; then
  profile_suffix="_test"
fi

filename_args=()
for lang in "${target_languages[@]}"; do
  for bench in "${target_benchmarks[@]}"; do
    rel_file="./benchmarks/$lang/${bench}${profile_suffix}.yml"
    abs_file="${uri%/}/benchmarks/$lang/${bench}${profile_suffix}.yml"
    if [ ! -f "$abs_file" ]; then
      die "Missing benchmark file: $abs_file"
    fi
    filename_args+=(--filename "$rel_file")
  done
done

if [ -z "$lang_csv" ]; then
  run_name_base="runall"
else
  run_name_base="run$(IFS=-; echo "${target_languages[*]}")"
fi

run_name="$run_name_base"
if [ "$profile" = "test" ]; then
  run_name="${run_name}-test"
fi

venv_python="${gmt_dir%/}/venv/bin/python3"
runner_py="${gmt_dir%/}/runner.py"

if [ ! -x "$venv_python" ]; then
  die "Missing GMT venv python executable: $venv_python"
fi

if [ ! -f "$runner_py" ]; then
  die "Missing GMT runner.py: $runner_py"
fi

runner_cmd=("$venv_python" "$runner_py" --uri "$uri" --name "$run_name")
runner_cmd+=("${filename_args[@]}")

if [ "$profile" = "test" ]; then
  runner_cmd+=(--dev-no-sleeps)
fi

runner_cmd+=(--iterations "$iterations" --docker-prune)

"${runner_cmd[@]}"
