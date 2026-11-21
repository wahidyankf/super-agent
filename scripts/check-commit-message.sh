#!/usr/bin/env bash
# Commitlint lints the commit message that the typed input supplies. The text arrives in the environment, from either
# Git's hook message file or an immutable pull-request range, so this script never receives a pathname.
set -euo pipefail

: "${RHINO_GATE_MESSAGE:?RHINO_GATE_MESSAGE is required}"
printf '%s\n' "$RHINO_GATE_MESSAGE" | npm exec -- commitlint
