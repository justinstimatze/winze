#!/usr/bin/env bash
# Install (or remove) an hourly systemd user timer that fires one metabolism
# cycle per tick. Paths are derived at install time, so nothing machine-local
# is committed to the repo.
#
#   scripts/install-metabolism-timer.sh              # install + start
#   scripts/install-metabolism-timer.sh --uninstall  # stop + remove
#
# Inspect afterwards with:
#   systemctl --user list-timers winze-metabolism.timer
#   journalctl --user -u winze-metabolism.service -f
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
UNIT_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
SERVICE="winze-metabolism.service"
TIMER="winze-metabolism.timer"

if [ "${1:-}" = "--uninstall" ]; then
  systemctl --user disable --now "$TIMER" 2>/dev/null || true
  rm -f "$UNIT_DIR/$TIMER" "$UNIT_DIR/$SERVICE"
  systemctl --user daemon-reload
  echo "removed $TIMER and $SERVICE"
  exit 0
fi

mkdir -p "$UNIT_DIR"

cat > "$UNIT_DIR/$SERVICE" <<EOF
[Unit]
Description=winze metabolism cycle (one tick)
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
WorkingDirectory=$REPO
ExecStart=$REPO/scripts/metabolism-tick.sh
# The Go linker peaks 1-2 GB RSS; cap the slice so a build can never take
# the desktop down via systemd-oomd.
MemoryMax=3G
# Never win a CPU fight against interactive work.
Nice=10
IOSchedulingClass=idle
TimeoutStartSec=45min
EOF

cat > "$UNIT_DIR/$TIMER" <<EOF
[Unit]
Description=winze metabolism hourly tick

[Timer]
Unit=$SERVICE
OnBootSec=10min
OnUnitInactiveSec=1h
# Catch up on ticks missed while the machine was off.
Persistent=true
# Avoid firing in lockstep with every other timer on the host.
RandomizedDelaySec=5min

[Install]
WantedBy=timers.target
EOF

systemctl --user daemon-reload
systemctl --user enable --now "$TIMER"

if ! loginctl show-user "$USER" -p Linger 2>/dev/null | grep -q 'Linger=yes'; then
  echo
  echo "NOTE: lingering is off, so the timer only runs while you are logged in."
  echo "      To let it run headless:  loginctl enable-linger $USER"
fi

echo
systemctl --user list-timers "$TIMER" --no-pager
