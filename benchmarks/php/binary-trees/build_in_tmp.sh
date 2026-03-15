#!/bin/sh
set -eu

SRC_PHP="/tmp/php-binary-trees.php"
BIN="/tmp/php-binary-trees"

cp /tmp/repo/benchmarks/php/binary-trees/main.php "$SRC_PHP"

PHP_BIN="php"
if [ -x /opt/src/php-8.4.1/bin/php ]; then
  PHP_BIN="/opt/src/php-8.4.1/bin/php"
elif [ -x /opt/src/php/bin/php ]; then
  PHP_BIN="/opt/src/php/bin/php"
fi

cat > "$BIN" <<EOF
#!/bin/sh
exec "$PHP_BIN" -dopcache.enable_cli=1 -dopcache.jit_buffer_size=64M -d memory_limit=4096M "$SRC_PHP" "\$@"
EOF

chmod +x "$BIN"
