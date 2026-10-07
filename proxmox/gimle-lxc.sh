#!/usr/bin/env bash
# Create a Proxmox LXC running Gimle.
# Run on a Proxmox VE host as root. Every network setting is a flag; nothing is hardcoded.
set -euo pipefail

usage() {
  cat <<USAGE
Usage: $0 [options]

Gimle:
  --version VER        Release to install, e.g. v1.0.0 (default: the latest)
  --binary PATH        Install a gimle binary you built yourself instead of a release
  --port PORT          Web UI port (default: 8840)
  --timeout SEC        Seconds between scans (default: 120)
  The container gets an 'update' command: run 'pct exec CTID -- update' later for the latest release.

Container:
  --ctid ID            Container ID (default: next free ID)
  --hostname NAME      Hostname (default: gimle)
  --storage NAME       Rootfs storage (default: local-lvm)
  --tmpl-storage NAME  Template storage (default: local)
  --disk GB            Rootfs size in GB (default: 4)
  --cores N            CPU cores (default: 1)
  --memory MB          Memory in MB (default: 512)

Network (first NIC is eth0; each --nic adds eth1, eth2, ...):
  --bridge BRIDGE      Bridge for eth0 (default: vmbr0)
  --vlan TAG           VLAN tag for eth0 (default: none)
  --ip CIDR|dhcp       eth0 address (default: dhcp)
  --gw IP              eth0 gateway (required with a static --ip)
  --nic BRIDGE[,TAG]   Extra NIC with DHCP for ARP discovery on another L2 segment (repeatable)
USAGE
}

REPO="${GIMLE_REPO:-vipthomps/Gimle}"
VERSION="latest" BINARY="" CTID="" HOSTNAME_="gimle" STORAGE="local-lvm" TMPL_STORAGE="local"
DISK=4 CORES=1 MEMORY=512 BRIDGE="vmbr0" VLAN="" IP="dhcp" GW="" PORT=8840 TIMEOUT=120
EXTRA_NICS=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) VERSION="$2"; shift 2 ;;
    --binary) BINARY="$2"; shift 2 ;;
    --ctid) CTID="$2"; shift 2 ;;
    --hostname) HOSTNAME_="$2"; shift 2 ;;
    --storage) STORAGE="$2"; shift 2 ;;
    --tmpl-storage) TMPL_STORAGE="$2"; shift 2 ;;
    --disk) DISK="$2"; shift 2 ;;
    --cores) CORES="$2"; shift 2 ;;
    --memory) MEMORY="$2"; shift 2 ;;
    --bridge) BRIDGE="$2"; shift 2 ;;
    --vlan) VLAN="$2"; shift 2 ;;
    --ip) IP="$2"; shift 2 ;;
    --gw) GW="$2"; shift 2 ;;
    --nic) EXTRA_NICS+=("$2"); shift 2 ;;
    --port) PORT="$2"; shift 2 ;;
    --timeout) TIMEOUT="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage; exit 1 ;;
  esac
done

die() { echo "Error: $*" >&2; exit 1; }

command -v pct >/dev/null || die "pct not found; run this on a Proxmox VE host"
[[ -z "$BINARY" || -f "$BINARY" ]] || die "binary not found: $BINARY"
[[ "$IP" == "dhcp" || -n "$GW" ]] || die "--gw is required with a static --ip"

[[ -n "$CTID" ]] || CTID=$(pvesh get /cluster/nextid)

# Debian 12 template
pveam update >/dev/null
TEMPLATE=$(pveam available --section system | awk '{print $2}' | grep '^debian-12-standard' | sort -V | tail -1)
[[ -n "$TEMPLATE" ]] || die "no debian-12-standard template available"
if ! pveam list "$TMPL_STORAGE" | grep -q "$TEMPLATE"; then
  echo "Downloading $TEMPLATE"
  pveam download "$TMPL_STORAGE" "$TEMPLATE"
fi

net0="name=eth0,bridge=${BRIDGE},ip=${IP}"
[[ -n "$VLAN" ]] && net0+=",tag=${VLAN}"
[[ "$IP" != "dhcp" ]] && net0+=",gw=${GW}"

NET_ARGS=(--net0 "$net0")
IFACES="eth0"
i=1
for nic in "${EXTRA_NICS[@]}"; do
  IFS=, read -r nbridge ntag <<<"$nic"
  spec="name=eth${i},bridge=${nbridge},ip=dhcp"
  [[ -n "$ntag" ]] && spec+=",tag=${ntag}"
  NET_ARGS+=(--net${i} "$spec")
  IFACES+=" eth${i}"
  i=$((i + 1))
done

echo "Creating CT $CTID ($HOSTNAME_)"
pct create "$CTID" "${TMPL_STORAGE}:vztmpl/${TEMPLATE}" \
  --hostname "$HOSTNAME_" \
  --unprivileged 1 \
  --features nesting=1 \
  --cores "$CORES" --memory "$MEMORY" --swap 0 \
  --rootfs "${STORAGE}:${DISK}" \
  --onboot 1 \
  "${NET_ARGS[@]}"

pct start "$CTID"
echo "Waiting for network"
for _ in $(seq 1 30); do
  pct exec "$CTID" -- getent hosts deb.debian.org >/dev/null 2>&1 && break
  sleep 2
done

pct exec "$CTID" -- bash -c "apt-get update -qq && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq arp-scan tzdata ca-certificates curl >/dev/null"

TMPDIR_=$(mktemp -d)
trap 'rm -rf "$TMPDIR_"' EXIT

# Settings first, so the service starts with them
cat > "$TMPDIR_/config_v2.yaml" <<CONF
host: 0.0.0.0
port: "${PORT}"
ifaces: ${IFACES}
timeout: ${TIMEOUT}
CONF
pct exec "$CTID" -- mkdir -p /var/lib/gimle
pct push "$CTID" "$TMPDIR_/config_v2.yaml" /var/lib/gimle/config_v2.yaml

# The update command installs releases; it is also how Gimle is installed now
curl -fsSL "https://raw.githubusercontent.com/${REPO}/main/proxmox/update.sh" -o "$TMPDIR_/update"
pct push "$CTID" "$TMPDIR_/update" /usr/bin/update --perms 0755

if [[ -n "$BINARY" ]]; then
  pct exec "$CTID" -- env GIMLE_REPO="$REPO" update "$VERSION" # service, data folder and the update command
  pct push "$CTID" "$BINARY" /usr/bin/gimle --perms 0755
  pct exec "$CTID" -- systemctl restart gimle
else
  pct exec "$CTID" -- env GIMLE_REPO="$REPO" update "$VERSION"
fi

CT_IP=$(pct exec "$CTID" -- hostname -I | awk '{print $1}')
echo "Gimle is running: http://${CT_IP}:${PORT}"
echo "Update it later with: pct exec ${CTID} -- update"
