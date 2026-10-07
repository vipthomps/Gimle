#!/usr/bin/env bash
# Install or update Gimle from its GitHub releases, inside a Debian container or VM.
#
#   update            the latest release
#   update v1.2.0     a particular release
#
# The release is a prebuilt binary, so nothing is compiled here and 512 MB of
# memory is plenty. Installed as /usr/bin/update by the Proxmox installer, and
# refreshed from each release it installs.
set -euo pipefail

REPO="${GIMLE_REPO:-vipthomps/Gimle}"
WANT="${1:-latest}"
DATA=/var/lib/gimle

die() { echo "Error: $*" >&2; exit 1; }
[[ $EUID -eq 0 ]] || die "run as root"
command -v curl >/dev/null || die "curl is needed: apt-get install -y curl"

case "$(dpkg --print-architecture 2>/dev/null || uname -m)" in
  amd64|x86_64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  armhf|armv7l) arch=armv7 ;;
  *) die "no Gimle build for $(uname -m)" ;;
esac

if [[ "$WANT" == latest ]]; then
  base="https://github.com/${REPO}/releases/latest/download"
else
  [[ "$WANT" == v* ]] || WANT="v${WANT}"
  base="https://github.com/${REPO}/releases/download/${WANT}"
fi
asset="gimle_linux_${arch}.tar.gz"

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

echo "Downloading ${asset} (${WANT})"
curl -fsSL "${base}/${asset}" -o "$work/$asset" || die "could not download ${base}/${asset}"
curl -fsSL "${base}/checksums.txt" -o "$work/checksums.txt" || die "could not download the checksums"
(cd "$work" && grep " ${asset}\$" checksums.txt | sha256sum -c --quiet -) || die "checksum mismatch for ${asset}"
tar -xzf "$work/$asset" -C "$work"
[[ -x "$work/gimle" ]] || die "the release has no gimle binary"

new=$("$work/gimle" -v)
old=$( (command -v gimle >/dev/null && gimle -v) 2>/dev/null || echo none)

mkdir -p "$DATA"

install -m 0755 "$work/gimle" /usr/bin/gimle.new
mv -f /usr/bin/gimle.new /usr/bin/gimle
if [[ -f "$work/update.sh" ]]; then
  install -m 0755 "$work/update.sh" /usr/bin/update.new
  mv -f /usr/bin/update.new /usr/bin/update
fi

if [[ ! -f /etc/systemd/system/gimle.service ]]; then
  cat > /etc/systemd/system/gimle.service <<UNIT
[Unit]
Description=Gimle home lab dashboard
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/usr/bin/gimle -d ${DATA}
Restart=on-failure

[Install]
WantedBy=multi-user.target
UNIT
  systemctl daemon-reload
  systemctl enable gimle >/dev/null 2>&1
fi
systemctl restart gimle

if [[ "$old" == "$new" ]]; then
  echo "Gimle ${new} reinstalled"
else
  echo "Gimle updated: ${old} -> ${new}"
fi
