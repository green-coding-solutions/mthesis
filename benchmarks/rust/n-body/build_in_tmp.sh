#!/bin/sh
set -eu

rustc \
  -C opt-level=3 \
  -C target-cpu=native \
  -C codegen-units=1 \
  /tmp/repo/benchmarks/rust/n-body/main.rs \
  -o /tmp/rust-n-body
