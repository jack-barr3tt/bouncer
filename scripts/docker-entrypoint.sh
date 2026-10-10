#!/bin/sh
set -eu

SITE_ROOT=${SITE_ROOT:-/site}
SCAFFOLD=${SCAFFOLD:-/opt/bouncer/scaffold}

mkdir -p "$SITE_ROOT/apps"

copy_missing() {
  src=$1
  dest=$2
  if [ -e "$dest" ]; then
    return
  fi
  mkdir -p "$(dirname "$dest")"
  cp -a "$src" "$dest"
}

copy_missing "$SCAFFOLD/AGENTS.md" "$SITE_ROOT/AGENTS.md"
copy_missing "$SCAFFOLD/apps.AGENTS.md" "$SITE_ROOT/apps/AGENTS.md"

# An empty site gets the Hello sample. A site that already has apps.yaml is left alone.
if [ ! -e "$SITE_ROOT/apps.yaml" ]; then
  cp -a "$SCAFFOLD/apps.yaml" "$SITE_ROOT/apps.yaml"
  if [ ! -e "$SITE_ROOT/apps/hello" ]; then
    cp -a "$SCAFFOLD/apps/hello" "$SITE_ROOT/apps/hello"
    rm -rf "$SITE_ROOT/apps/hello/node_modules"
  fi
fi

# create-bouncer writes Hello source without a build. Use the image's dist.
if [ -e "$SITE_ROOT/apps/hello/package.json" ] && [ ! -e "$SITE_ROOT/apps/hello/dist/index.html" ] && [ -d "$SCAFFOLD/apps/hello/dist" ]; then
  mkdir -p "$SITE_ROOT/apps/hello/dist"
  cp -a "$SCAFFOLD/apps/hello/dist/." "$SITE_ROOT/apps/hello/dist/"
fi

if [ "${1:-}" = "init" ]; then
  exit 0
fi

if [ -n "${HUB_UPSTREAM:-}" ] && [ -f /opt/bouncer/hub/dist/server/server.js ]; then
  cd /opt/bouncer/hub
  npx srvx --prod --host 127.0.0.1 --port 3000 -s dist/client dist/server/server.js &
fi

exec /usr/local/bin/bouncer
