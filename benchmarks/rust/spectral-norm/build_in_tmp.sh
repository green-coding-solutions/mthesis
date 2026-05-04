#!/bin/sh
set -eu

SRC_DIR="/tmp/rust-spectral-norm-src"
CARGO_HOME_DIR="/tmp/cargo-home"
TARGET_DIR="/tmp/cargo-target/spectral-norm"

rm -rf "$SRC_DIR"
mkdir -p "$SRC_DIR"

cp /tmp/repo/benchmarks/rust/spectral-norm/main.rs "$SRC_DIR/main.rs"
cp /tmp/repo/benchmarks/rust/spectral-norm/Cargo.toml "$SRC_DIR/Cargo.toml"
cp /tmp/rust-prefetch/spectral-norm/Cargo.lock "$SRC_DIR/Cargo.lock"

cd "$SRC_DIR"
CARGO_HOME="$CARGO_HOME_DIR" \
CARGO_TARGET_DIR="$TARGET_DIR" \
RUSTFLAGS="-C opt-level=3 -C target-cpu=native -C codegen-units=1" \
cargo build --release --locked --offline

cp "$TARGET_DIR/release/spectral-norm" /tmp/rust-spectral-norm
