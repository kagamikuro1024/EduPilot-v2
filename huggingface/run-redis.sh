#!/usr/bin/env bash
set -Eeuo pipefail

MODULE_DIR=/opt/redis-stack/lib

exec /opt/redis-stack/bin/redis-server \
  --bind 127.0.0.1 \
  --port 6379 \
  --protected-mode no \
  --dir "$REDIS_DATA_DIR" \
  --appendonly yes \
  --loadmodule "$MODULE_DIR/rediscompat.so" \
  --loadmodule "$MODULE_DIR/redisearch.so" MAXSEARCHRESULTS 10000 MAXAGGREGATERESULTS 10000 \
  --loadmodule "$MODULE_DIR/redistimeseries.so" \
  --loadmodule "$MODULE_DIR/rejson.so" \
  --loadmodule "$MODULE_DIR/redisbloom.so" \
  --loadmodule "$MODULE_DIR/redisgears.so" v8-plugin-path "$MODULE_DIR/libredisgears_v8_plugin.so"
