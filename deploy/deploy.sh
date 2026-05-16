#!/usr/bin/env bash
set -euo pipefail

# Run on the Debian 12 server. Idempotent — safe to re-run.
# Layout: /opt/dujiao/{api,user,admin,config.yml,*.conf,docker-compose.yml,.secrets}

ROOT=/opt/dujiao
cd "$ROOT"

echo "==> [0/7] Configuring swap (6GB) + kernel for low-RAM builds"
swapoff -a 2>/dev/null || true
if [ ! -f /swapfile ] || [ "$(stat -c%s /swapfile 2>/dev/null || echo 0)" -lt $((6*1024*1024*1024)) ]; then
  rm -f /swapfile
  fallocate -l 6G /swapfile 2>/dev/null || dd if=/dev/zero of=/swapfile bs=1M count=6144
  chmod 600 /swapfile
  mkswap /swapfile
fi
swapon /swapfile
grep -q '^/swapfile' /etc/fstab || echo '/swapfile none swap sw 0 0' >> /etc/fstab
sysctl -w vm.swappiness=60 >/dev/null
sysctl -w vm.vfs_cache_pressure=50 >/dev/null
free -h

echo "==> [1/7] Ensuring Docker + Compose plugin are installed"
if ! command -v docker >/dev/null 2>&1; then
  apt-get update -y
  apt-get install -y ca-certificates curl gnupg git
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian bookworm stable" \
    > /etc/apt/sources.list.d/docker.list
  apt-get update -y
  apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  systemctl enable --now docker
else
  apt-get install -y git || true
fi
docker --version
docker compose version

echo "==> [2/7] Cloning / updating source repos"
clone_or_pull() {
  local name=$1 url=$2
  if [ -d "$ROOT/$name/.git" ]; then
    git -C "$ROOT/$name" fetch --depth=1 origin
    git -C "$ROOT/$name" reset --hard origin/HEAD
  else
    rm -rf "$ROOT/$name"
    git clone --depth=1 "$url" "$ROOT/$name"
  fi
}
clone_or_pull api   https://github.com/dujiao-next/dujiao-next.git
clone_or_pull user  https://github.com/dujiao-next/user.git
clone_or_pull admin https://github.com/dujiao-next/admin.git

# Inject NODE_OPTIONS into the frontend Dockerfiles so vite/vue-tsc don't OOM
# Heap limit must be below the cgroup --memory ceiling we set in build_one (900m)
# to leave room for V8 metadata + node binary + child processes.
inject_node_opts() {
  local f=$1
  # Remove any existing injection so we can update the value safely on re-run
  sed -i '/^ENV NODE_OPTIONS=/d' "$f"
  awk '/^FROM node/ && !done { print; print "ENV NODE_OPTIONS=--max-old-space-size=700"; done=1; next } { print }' "$f" > "$f.new"
  mv "$f.new" "$f"
}
inject_node_opts "$ROOT/user/Dockerfile"
inject_node_opts "$ROOT/admin/Dockerfile"

echo "==> [3/7] Generating secrets (idempotent — first run only)"
if [ ! -f "$ROOT/.secrets" ]; then
  umask 077
  APP_SECRET=$(openssl rand -hex 32)
  JWT_SECRET=$(openssl rand -hex 32)
  USER_JWT_SECRET=$(openssl rand -hex 32)
  ADMIN_PWD="Dj$(openssl rand -base64 18 | tr -d '/+=' | head -c14)1A"
  cat > "$ROOT/.secrets" <<EOF
APP_SECRET=$APP_SECRET
JWT_SECRET=$JWT_SECRET
USER_JWT_SECRET=$USER_JWT_SECRET
ADMIN_PASSWORD=$ADMIN_PWD
EOF
  echo "    wrote $ROOT/.secrets (chmod 600)"
else
  echo "    $ROOT/.secrets already exists, reusing"
fi
# shellcheck disable=SC1090
. "$ROOT/.secrets"

echo "==> [4/7] Rendering config.yml from template"
sed \
  -e "s|__APP_SECRET__|$APP_SECRET|g" \
  -e "s|__JWT_SECRET__|$JWT_SECRET|g" \
  -e "s|__USER_JWT_SECRET__|$USER_JWT_SECRET|g" \
  -e "s|__ADMIN_PASSWORD__|$ADMIN_PASSWORD|g" \
  "$ROOT/config.template.yml" > "$ROOT/config.yml"
chmod 600 "$ROOT/config.yml"

echo "==> [5/7] Pulling base images (redis)"
docker compose -f "$ROOT/docker-compose.yml" pull --ignore-buildable || true

build_one() {
  local svc=$1 ctx=$2 tag=$3 mem=$4
  echo "==> Building $svc (memory=$mem)"
  sync; echo 3 > /proc/sys/vm/drop_caches 2>/dev/null || true
  free -h
  # Classic builder honors --memory cgroup limits; pass build-time platform args
  # explicitly since the classic builder does not auto-inject TARGET* like buildx does.
  DOCKER_BUILDKIT=0 docker build \
    --memory="$mem" --memory-swap=5g \
    --build-arg TARGETOS=linux \
    --build-arg TARGETARCH=amd64 \
    --build-arg TARGETVARIANT= \
    -t "$tag" "$ctx"
  echo "    $svc image built"
}

echo "==> [6/7] Building images one at a time"
# api: Go compile is RAM-hungry (single linker process); give it nearly all RAM
build_one api   "$ROOT/api"   dujiao-api:local   950m
# user/admin: Node heap capped to 700M via NODE_OPTIONS, cgroup at 900m for headroom
build_one user  "$ROOT/user"  dujiao-user:local  900m
build_one admin "$ROOT/admin" dujiao-admin:local 900m

echo "==> [7/7] Starting stack"
docker compose -f "$ROOT/docker-compose.yml" up -d
echo "    waiting for API health (up to 90s)..."
ok=0
for i in $(seq 1 45); do
  if curl -fsS http://127.0.0.1:8080/health >/dev/null 2>&1; then ok=1; break; fi
  sleep 2
done
if [ $ok -eq 1 ]; then echo "    API healthy"; else echo "    !! API not healthy — check: docker compose logs api"; fi

PUBIP=$(curl -fsS --max-time 5 https://api.ipify.org 2>/dev/null || hostname -I | awk '{print $1}')
cat <<EOF

================================================================
Deploy complete.

  User front:  http://$PUBIP/
  Admin:       http://$PUBIP:8081/
  API health:  http://$PUBIP:8080/health

  Initial admin login:
    username: admin
    password: $ADMIN_PASSWORD

  Secrets file (root-only): $ROOT/.secrets
  Logs:   docker compose -f $ROOT/docker-compose.yml logs -f
  Stop:   docker compose -f $ROOT/docker-compose.yml down
================================================================
EOF
