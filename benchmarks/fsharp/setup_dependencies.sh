#!/bin/sh
set -eu

OUT_DIR="/tmp/fsharp-deps"
SRC_DIR="$OUT_DIR/restore-src"
PROJECT_FILE="$SRC_DIR/deps.csproj"
MARKER_FILE="$OUT_DIR/.ready"

if [ -f "$MARKER_FILE" ]; then
  exit 0
fi

if ! command -v dotnet >/dev/null 2>&1; then
  echo "[ERROR] Command not found 'dotnet'" >&2
  exit 127
fi

rm -rf "$OUT_DIR"
mkdir -p "$SRC_DIR"

cat > "$PROJECT_FILE" <<'EOF'
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net9.0</TargetFramework>
    <OutputType>Library</OutputType>
    <Nullable>disable</Nullable>
    <ImplicitUsings>false</ImplicitUsings>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Microsoft.Experimental.Collections" Version="1.0.6-e190117-3" />
  </ItemGroup>
</Project>
EOF

DOTNET_CLI_TELEMETRY_OPTOUT=1 \
DOTNET_NOLOGO=1 \
NUGET_PACKAGES=/tmp/nuget-packages \
dotnet restore "$PROJECT_FILE"

touch "$MARKER_FILE"
