#!/usr/bin/env bash
set -euo pipefail

if [[ "${EUID}" -ne 0 ]]; then
  echo "Run this installer as root." >&2
  exit 1
fi

SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AGENT_BINARY="${1:-${SOURCE_DIR}/build/sdr-agent-linux-arm64}"
if [[ ! -f "${AGENT_BINARY}" || -L "${AGENT_BINARY}" ]]; then
  echo "Agent binary not found or unsafe: ${AGENT_BINARY}" >&2
  exit 1
fi

getent group lyrahcle >/dev/null || groupadd --system lyrahcle
id lyrahcle >/dev/null 2>&1 || useradd --system --gid lyrahcle --home-dir /var/lib/lyrahcle --shell /usr/sbin/nologin lyrahcle
install -d -o root -g lyrahcle -m 0750 /etc/lyrahcle /etc/lyrahcle/config-keys
install -d -o lyrahcle -g lyrahcle -m 0700 /var/lib/lyrahcle/identity /var/lib/lyrahcle/updates/pending
install -o root -g root -m 0755 "${AGENT_BINARY}" /usr/local/bin/sdr-agent
install -o root -g root -m 0755 "${SOURCE_DIR}/bin/lyrahcle-enroll" /usr/local/sbin/lyrahcle-enroll
install -o root -g root -m 0755 "${SOURCE_DIR}/bin/lyrahcle-doctor" /usr/local/sbin/lyrahcle-doctor
install -o root -g root -m 0755 "${SOURCE_DIR}/bin/lyrahcle-secure-update" /usr/local/sbin/lyrahcle-secure-update
install -o root -g root -m 0644 "${SOURCE_DIR}/systemd/lyrahcle-sdr-agent.service" /etc/systemd/system/
install -o root -g root -m 0644 "${SOURCE_DIR}/systemd/lyrahcle-agent-update.service" /etc/systemd/system/
install -o root -g root -m 0644 "${SOURCE_DIR}/systemd/lyrahcle-agent-update.timer" /etc/systemd/system/
install -o root -g root -m 0644 "${SOURCE_DIR}/systemd/lyrahcle-sdr-agent.sysusers" /usr/lib/sysusers.d/lyrahcle-sdr-agent.conf
install -o root -g root -m 0644 "${SOURCE_DIR}/systemd/lyrahcle-sdr-agent.tmpfiles" /usr/lib/tmpfiles.d/lyrahcle-sdr-agent.conf
install -o root -g root -m 0644 "${SOURCE_DIR}/config/60-lyrahcle-rtl-sdr.rules" /etc/udev/rules.d/
systemctl daemon-reload
systemctl enable lyrahcle-agent-update.timer
udevadm control --reload-rules

echo "Base station installed. Add the CA and public signing keys, then run: sudo lyrahcle-enroll"
