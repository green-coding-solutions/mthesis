#!/bin/sh
set -eu

SRC_PHP="/tmp/php-fannkuch-redux.php"
BIN="/tmp/php-fannkuch-redux"

cp /tmp/repo/benchmarks/php/fannkuch-redux/main.php "$SRC_PHP"

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

SHMOP_ARG="-dextension=shmop"
PCNTL_ARG="-dextension=pcntl"
if [ -n "$EXT_DIR" ] && [ -f "$EXT_DIR/shmop.so" ]; then
  SHMOP_ARG="-dextension=$EXT_DIR/shmop.so"
fi
if [ -n "$EXT_DIR" ] && [ -f "$EXT_DIR/pcntl.so" ]; then
  PCNTL_ARG="-dextension=$EXT_DIR/pcntl.so"
fi

cat > "$BIN" <<EOF
#!/bin/sh
if [ -n "$OPCACHE_SO" ]; then
  exec "$PHP_BIN" -dzend_extension="$OPCACHE_SO" -dopcache.enable_cli=1 -dopcache.jit_buffer_size=64M -n $SHMOP_ARG $PCNTL_ARG -d memory_limit=4096M "$SRC_PHP" "\$@"
else
  exec "$PHP_BIN" -n $SHMOP_ARG $PCNTL_ARG -d memory_limit=4096M "$SRC_PHP" "\$@"
fi
EOF

chmod +x "$BIN"
