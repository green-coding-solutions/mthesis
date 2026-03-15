#!/bin/sh
set -eu

SRC_PHP="/tmp/php-n-body.php"
BIN="/tmp/php-n-body"

cp /tmp/repo/benchmarks/php/n-body/main.php "$SRC_PHP"

PHP_BIN="php"
if [ -x /opt/src/php-8.4.1/bin/php ]; then
  PHP_BIN="/opt/src/php-8.4.1/bin/php"
elif [ -x /opt/src/php/bin/php ]; then
  PHP_BIN="/opt/src/php/bin/php"
fi

EXT_DIR="$("$PHP_BIN" -n -r 'echo ini_get("extension_dir");' 2>/dev/null || true)"
OPCACHE_SO=""
if [ -f /opt/src/php-8.4.1/lib/php/extensions/no-debug-non-zts-20240924/opcache.so ]; then
  OPCACHE_SO="/opt/src/php-8.4.1/lib/php/extensions/no-debug-non-zts-20240924/opcache.so"
elif [ -n "$EXT_DIR" ] && [ -f "$EXT_DIR/opcache.so" ]; then
  OPCACHE_SO="$EXT_DIR/opcache.so"
fi

cat > "$BIN" <<EOF
#!/bin/sh
if [ -n "$OPCACHE_SO" ]; then
  exec "$PHP_BIN" -dzend_extension="$OPCACHE_SO" -dopcache.enable_cli=1 -dopcache.jit_buffer_size=64M -n -d short_open_tag=1 "$SRC_PHP" "\$@"
else
  exec "$PHP_BIN" -n -d short_open_tag=1 "$SRC_PHP" "\$@"
fi
EOF

chmod +x "$BIN"
