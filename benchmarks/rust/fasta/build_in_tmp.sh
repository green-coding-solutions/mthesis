#!/bin/sh
set -eu

SRC_DIR="/tmp/rust-fasta-src"
CARGO_HOME_DIR="/tmp/cargo-home"
TARGET_DIR="/tmp/cargo-target/fasta"

rm -rf "$SRC_DIR"
mkdir -p "$SRC_DIR"

cp /tmp/repo/benchmarks/rust/fasta/main.rs "$SRC_DIR/main.rs"
cp /tmp/repo/benchmarks/rust/fasta/Cargo.toml "$SRC_DIR/Cargo.toml"

cd "$SRC_DIR"
CARGO_HOME="$CARGO_HOME_DIR" \
CARGO_TARGET_DIR="$TARGET_DIR" \
RUSTFLAGS="-C opt-level=3 -C target-cpu=native -C codegen-units=1" \
cargo build --release

cp "$TARGET_DIR/release/fasta" /tmp/rust-fasta
