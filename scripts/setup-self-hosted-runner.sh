#!/usr/bin/env bash
set -euo pipefail

REPO="${REPO:-emershoww/15soat-tech-challenge-step-1}"
RUNNER_DIR="${RUNNER_DIR:-$HOME/actions-runner/15soat-tech-challenge-step-1}"
RUNNER_NAME="${RUNNER_NAME:-$(hostname)-15soat-tech-challenge-step-1}"
RUNNER_LABELS="${RUNNER_LABELS:-self-hosted,linux,x64,local-kind}"

if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI is required to discover the runner version and, when possible, request a registration token." >&2
  exit 1
fi

RUNNER_VERSION="${RUNNER_VERSION:-$(gh api repos/actions/runner/releases/latest --jq '.tag_name' | sed 's/^v//')}"
RUNNER_ARCHIVE="actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz"
RUNNER_URL="https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/${RUNNER_ARCHIVE}"

if [ -z "${RUNNER_TOKEN:-}" ]; then
  RUNNER_TOKEN="$(gh api --method POST "repos/${REPO}/actions/runners/registration-token" --jq '.token')"
fi

mkdir -p "$RUNNER_DIR"
cd "$RUNNER_DIR"

if [ ! -f "$RUNNER_ARCHIVE" ]; then
  curl -fsSLO "$RUNNER_URL"
fi

tar xzf "$RUNNER_ARCHIVE"

if [ ! -f .runner ]; then
  ./config.sh \
    --url "https://github.com/${REPO}" \
    --token "$RUNNER_TOKEN" \
    --name "$RUNNER_NAME" \
    --labels "$RUNNER_LABELS" \
    --unattended
fi

cat <<EOF
Runner configured in: ${RUNNER_DIR}

Start it interactively:
  cd "${RUNNER_DIR}"
  ./run.sh

Or install it as a service:
  cd "${RUNNER_DIR}"
  sudo ./svc.sh install
  sudo ./svc.sh start
  sudo ./svc.sh status
EOF
