#!/bin/sh
set -eu

SRC_RB="/tmp/ruby-mandelbrot.rb"
BIN="/tmp/ruby-mandelbrot"

cp /tmp/repo/benchmarks/ruby/mandelbrot/main.rb "$SRC_RB"

RUBY_BIN="ruby"
if [ -x /opt/src/ruby-3.4.0/bin/ruby ]; then
  RUBY_BIN="/opt/src/ruby-3.4.0/bin/ruby"
fi

cat > "$BIN" <<EOF
#!/bin/sh
exec "$RUBY_BIN" --yjit -W0 "$SRC_RB" "\$@"
EOF

chmod +x "$BIN"
