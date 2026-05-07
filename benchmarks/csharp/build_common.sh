#!/bin/sh
set -eu

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <benchmark-name>" >&2
  exit 1
fi

BENCH="$1"
case "$BENCH" in
  binary-trees|fannkuch-redux|fasta|k-nucleotide|mandelbrot|n-body|regex-redux|spectral-norm)
    ;;
  *)
    echo "[ERROR] Unsupported benchmark '$BENCH'" >&2
    exit 2
    ;;
esac

if ! command -v dotnet >/dev/null 2>&1; then
  echo "[ERROR] Command not found 'dotnet'" >&2
  exit 127
fi

OS="$(uname -s)"
ARCH="$(uname -m)"
RID="linux-x64"

# Keep linux-x64 as the primary target; use linux-arm64 for Docker-on-arm hosts.
if [ "$OS" != "Linux" ] || { [ "$ARCH" != "x86_64" ] && [ "$ARCH" != "amd64" ]; }; then
  RID="linux-arm64"
fi

SRC_CS="/tmp/repo/benchmarks/csharp/$BENCH/main.cs"
BUILD_DIR="/tmp/csharp-build/$BENCH"
PROJECT_FILE="$BUILD_DIR/program.csproj"
OUT_BIN="/tmp/csharp-$BENCH"

if [ ! -f "$SRC_CS" ]; then
  echo "[ERROR] Missing source file '$SRC_CS'" >&2
  exit 3
fi

rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"
cp "$SRC_CS" "$BUILD_DIR/Program.cs"

{
  cat <<EOF
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net9.0</TargetFramework>
    <RuntimeIdentifier>$RID</RuntimeIdentifier>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
    <AllowUnsafeBlocks>true</AllowUnsafeBlocks>
    <ServerGarbageCollection>true</ServerGarbageCollection>
    <ConcurrentGarbageCollection>true</ConcurrentGarbageCollection>
    <PublishAot>true</PublishAot>
    <OptimizationPreference>Speed</OptimizationPreference>
    <IlcInstructionSet>native</IlcInstructionSet>
EOF

  if [ "$BENCH" = "mandelbrot" ]; then
    cat <<'EOF'
    <CheckForOverflowUnderflow>false</CheckForOverflowUnderflow>
EOF
  fi

  cat <<'EOF'
  </PropertyGroup>
</Project>
EOF
} > "$PROJECT_FILE"

DOTNET_CLI_TELEMETRY_OPTOUT=1 \
DOTNET_NOLOGO=1 \
NUGET_PACKAGES=/tmp/nuget-packages \
dotnet publish -r "$RID" -c Release "$PROJECT_FILE"

PUBLISH_BIN="$BUILD_DIR/bin/Release/net9.0/$RID/publish/program"
NATIVE_BIN="$BUILD_DIR/bin/Release/net9.0/$RID/native/program"

if [ -x "$PUBLISH_BIN" ]; then
  cp "$PUBLISH_BIN" "$OUT_BIN"
elif [ -x "$NATIVE_BIN" ]; then
  cp "$NATIVE_BIN" "$OUT_BIN"
else
  echo "[ERROR] Build output not found '$PUBLISH_BIN' or '$NATIVE_BIN'" >&2
  exit 4
fi

chmod +x "$OUT_BIN"
