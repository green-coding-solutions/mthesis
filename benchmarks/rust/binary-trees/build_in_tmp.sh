#!/bin/sh
set -eu

SRC_DIR="/tmp/rust-binary-trees-src"
CARGO_HOME_DIR="/tmp/cargo-home"
TARGET_DIR="/tmp/cargo-target/binary-trees"

rm -rf "$SRC_DIR"
mkdir -p "$SRC_DIR"

cp /tmp/repo/benchmarks/rust/binary-trees/main.rs "$SRC_DIR/main.rs"
cp /tmp/repo/benchmarks/rust/binary-trees/Cargo.toml "$SRC_DIR/Cargo.toml"

RUSTFLAGS_VALUE="-C opt-level=3 -C target-cpu=ivybridge -C codegen-units=1"
if [ -d /opt/src/rust-libs ]; then
  RUSTFLAGS_VALUE="$RUSTFLAGS_VALUE -L /opt/src/rust-libs"
fi

cd "$SRC_DIR"
CARGO_HOME="$CARGO_HOME_DIR" \
CARGO_TARGET_DIR="$TARGET_DIR" \
RUSTFLAGS="$RUSTFLAGS_VALUE" \
cargo build --release
