#!/bin/sh
# Daily backup of the database and uploaded photos, keeping the last $KEEP_DAYS days.
set -eu

KEEP_DAYS="${KEEP_DAYS:-14}"
DIR=/backups

while true; do
  stamp=$(date +%Y-%m-%d_%H%M)
  if pg_dump --format=custom --file="$DIR/db_$stamp.dump"; then
    echo "$(date -Iseconds) database backup ok: db_$stamp.dump"
  else
    echo "$(date -Iseconds) database backup FAILED" >&2
  fi
  tar -czf "$DIR/uploads_$stamp.tar.gz" -C /uploads . && echo "$(date -Iseconds) uploads backup ok"
  find "$DIR" -type f -mtime +"$KEEP_DAYS" -delete
  sleep 86400
done
