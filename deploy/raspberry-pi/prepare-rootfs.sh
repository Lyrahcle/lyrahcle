#!/usr/bin/env bash
set -euo pipefail

if [[ "${EUID}" -ne 0 || "$#" -ne 2 ]]; then
  echo "Usage: sudo $0 <mounted-rootfs> <arm64-agent-binary>" >&2
  exit 1
fi
ROOTFS="$(realpath "$1")"
BINARY="$(realpath "$2")"
case "${ROOTFS}" in
  /|/home|/usr|/etc|/var) echo "Refusing unsafe rootfs target: ${ROOTFS}" >&2; exit 1 ;;
esac
[[ -d "${ROOTFS}/etc" && -f "${BINARY}" && ! -L "${BINARY}" ]] || { echo "Invalid rootfs or binary" >&2; exit 1; }
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
install -d -m 0755 "${ROOTFS}/usr/local/bin" "${ROOTFS}/usr/local/sbin" "${ROOTFS}/usr/lib/sysusers.d" "${ROOTFS}/usr/lib/tmpfiles.d" "${ROOTFS}/etc/udev/rules.d" "${ROOTFS}/etc/systemd/system/multi-user.target.wants" "${ROOTFS}/etc/systemd/system/timers.target.wants"
install -d -m 0750 "${ROOTFS}/etc/lyrahcle" "${ROOTFS}/etc/lyrahcle/config-keys"
install -d -m 0700 "${ROOTFS}/var/lib/lyrahcle/identity" "${ROOTFS}/var/lib/lyrahcle/updates/pending"
install -m 0755 "${BINARY}" "${ROOTFS}/usr/local/bin/sdr-agent"
install -m 0755 "${SOURCE_DIR}/bin/lyrahcle-enroll" "${ROOTFS}/usr/local/sbin/"
install -m 0755 "${SOURCE_DIR}/bin/lyrahcle-doctor" "${ROOTFS}/usr/local/sbin/"
install -m 0755 "${SOURCE_DIR}/bin/lyrahcle-secure-update" "${ROOTFS}/usr/local/sbin/"
install -m 0644 "${SOURCE_DIR}/systemd/"*.service "${SOURCE_DIR}/systemd/"*.timer "${ROOTFS}/etc/systemd/system/"
install -m 0644 "${SOURCE_DIR}/systemd/lyrahcle-sdr-agent.sysusers" "${ROOTFS}/usr/lib/sysusers.d/lyrahcle-sdr-agent.conf"
install -m 0644 "${SOURCE_DIR}/systemd/lyrahcle-sdr-agent.tmpfiles" "${ROOTFS}/usr/lib/tmpfiles.d/lyrahcle-sdr-agent.conf"
install -m 0644 "${SOURCE_DIR}/config/60-lyrahcle-rtl-sdr.rules" "${ROOTFS}/etc/udev/rules.d/"
ln -sfn ../lyrahcle-agent-update.timer "${ROOTFS}/etc/systemd/system/timers.target.wants/lyrahcle-agent-update.timer"
echo "Root filesystem prepared. User/group creation and enrollment run on first boot or via install.sh."
