#!/bin/sh
set -eu

: "${SSH_KEY:?missing secret: deploy key}"
: "${DEPLOY_HOST:?missing secret: deploy host}"
: "${DEPLOY_USER:?missing secret: deploy user}"
: "${DEPLOY_PATH:?missing secret: deploy path}"

case "$DEPLOY_HOST" in
  *[!A-Za-z0-9.:-]*) echo "deploy host contains unsupported characters" >&2; exit 1 ;;
esac
case "$DEPLOY_USER" in
  *[!A-Za-z0-9._-]*) echo "deploy user contains unsupported characters" >&2; exit 1 ;;
esac
case "$DEPLOY_PATH" in
  *\'*) echo "deploy path cannot contain a single quote" >&2; exit 1 ;;
esac

root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$root"

payload=$(mktemp -d)
tarball=$(mktemp)
trap 'rm -rf "$payload" "$tarball"' EXIT

found=0
if [ -f apps.yaml ]; then
  cp apps.yaml "$payload/apps.yaml"
  found=1
fi
for dist in apps/*/dist; do
  [ -d "$dist" ] || continue
  slug=$(basename "$(dirname "$dist")")
  mkdir -p "$payload/apps/$slug"
  cp -a "$dist" "$payload/apps/$slug/dist"
  found=1
done
if [ "$found" -eq 0 ]; then
  echo "nothing to deploy" >&2
  exit 1
fi
tar czf "$tarball" -C "$payload" .

mkdir -p ~/.ssh && chmod 700 ~/.ssh
key=~/.ssh/id_ed25519
if printf '%s' "$SSH_KEY" | grep -q -- '-----BEGIN '; then
  printf '%s\n' "$SSH_KEY" > "$key"
else
  printf '%s' "$SSH_KEY" | base64 -d > "$key"
fi
chmod 600 "$key"
ssh-keyscan -H "$DEPLOY_HOST" >> ~/.ssh/known_hosts 2>/dev/null

ssh -i "$key" -o BatchMode=yes "$DEPLOY_USER@$DEPLOY_HOST" "mkdir -p '$DEPLOY_PATH/incoming'"
scp -i "$key" -o BatchMode=yes "$tarball" "$DEPLOY_USER@$DEPLOY_HOST:$DEPLOY_PATH/incoming/release.tar.gz"
ssh -i "$key" -o BatchMode=yes "$DEPLOY_USER@$DEPLOY_HOST" \
  "set -eu; tar xzf '$DEPLOY_PATH/incoming/release.tar.gz' -C '$DEPLOY_PATH'; rm -f '$DEPLOY_PATH/incoming/release.tar.gz'"

echo "==> Deploy complete"
