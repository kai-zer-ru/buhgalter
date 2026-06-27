#!/bin/sh
set -e

# Bind mounts from the host are often created by Docker as root. Fix ownership
# before dropping to the app user (same pattern as postgres, redis, etc.).
for dir in /app/data /app/logs /app/backups; do
	mkdir -p "$dir"
	chown -R buhgalter:buhgalter "$dir"
done
mkdir -p /app/logs/audit
chown -R buhgalter:buhgalter /app/logs

exec su-exec buhgalter /app/buhgalter "$@"
