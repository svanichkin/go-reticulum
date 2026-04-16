#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${ANNOUNCE_INTERFACE_NAME:-}" || -z "${ANNOUNCE_INTERFACE_DIR:-}" || -z "${ANNOUNCE_ENV_PREFIX:-}" ]]; then
  echo "ANNOUNCE_INTERFACE_NAME, ANNOUNCE_INTERFACE_DIR and ANNOUNCE_ENV_PREFIX are required" >&2
  exit 2
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)"
ANNOUNCE_DIR="$ROOT/tests/runners/parity/announce"
USE_DOCKER_VAR="${ANNOUNCE_ENV_PREFIX}_USE_DOCKER"
IMAGE_VAR="${ANNOUNCE_ENV_PREFIX}_IMAGE"
SCRIPTS_VAR="${ANNOUNCE_ENV_PREFIX}_SCRIPTS"
PARALLEL_VAR="${ANNOUNCE_ENV_PREFIX}_PARALLEL"

USE_DOCKER="${!USE_DOCKER_VAR:-1}"
RUNNER_IMAGE="${!IMAGE_VAR:-reticulum-parity-announce:latest}"
SCRIPT_FILTER="${!SCRIPTS_VAR:-}"
HOST_UID="$(id -u)"
HOST_GID="$(id -g)"
TS="${TS:-"$(date +"%Y%m%d-%H%M%S")"}"
PARALLEL_DEFAULT="$USE_DOCKER"
PARALLEL="${!PARALLEL_VAR:-$PARALLEL_DEFAULT}"
RUNNER_LOG_DIR="$ROOT/tests/artifacts/logs/$TS/parity_announce_${ANNOUNCE_INTERFACE_NAME}_runner"

usage() {
  cat <<EOF
Usage: tests/runners/parity/announce/${ANNOUNCE_INTERFACE_NAME}/run.sh

Runs all ${ANNOUNCE_INTERFACE_LABEL:-$ANNOUNCE_INTERFACE_NAME} announce parity scripts in this directory except:
  - run.sh
  - interface.sh
  - helper files prefixed with "_"

Env:
  ${ANNOUNCE_ENV_PREFIX}_USE_DOCKER=1|0   default 1
  ${ANNOUNCE_ENV_PREFIX}_PARALLEL=1|0      default follows docker mode
  ${ANNOUNCE_ENV_PREFIX}_IMAGE=<tag>       default reticulum-parity-announce:latest
  ${ANNOUNCE_ENV_PREFIX}_SCRIPTS="..."     optional space-separated script list
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ "$#" -ne 0 ]]; then
  usage
  exit 2
fi

echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] dir=$ANNOUNCE_INTERFACE_DIR"
echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] docker=$USE_DOCKER"
echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] parallel=$PARALLEL"

build_runner_image() {
  if [[ "$USE_DOCKER" != "1" ]]; then
    return 0
  fi
  echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] docker build $RUNNER_IMAGE"
  docker build -t "$RUNNER_IMAGE" -f "$ANNOUNCE_DIR/Dockerfile" "$ANNOUNCE_DIR" >/dev/null
}

should_run_script() {
  local name="$1"
  if [[ -z "$SCRIPT_FILTER" ]]; then
    return 0
  fi
  local item
  for item in $SCRIPT_FILTER; do
    if [[ "$item" == "$name" ]]; then
      return 0
    fi
  done
  return 1
}

run_script() {
  local script="$1"
  local name
  name="$(basename "$script")"
  local rel_script="${script#$ROOT/}"
  local container_iface="${ANNOUNCE_INTERFACE_NAME//\//-}"
  if [[ "$USE_DOCKER" != "1" ]]; then
    bash "$script"
    return
  fi

  local container="parity-announce-${container_iface}-${name%.sh}-${TS//[^a-zA-Z0-9]/}"
  local -a docker_args=(
    docker run --rm --init
    --name "$container"
  )
  if [[ -n "${ANNOUNCE_DOCKER_ARGS:-}" ]]; then
    # shellcheck disable=SC2206
    local extra_args=( $ANNOUNCE_DOCKER_ARGS )
    docker_args+=("${extra_args[@]}")
  fi
  if [[ "${ANNOUNCE_DOCKER_ROOT:-0}" == "1" ]]; then
    docker_args+=(
      -e HOST_UID="$HOST_UID"
      -e HOST_GID="$HOST_GID"
    )
  else
    docker_args+=(--user "$HOST_UID:$HOST_GID")
  fi
  docker_args+=(
    -e TS="$TS"
    -e PARITY_BIN_DIR=
    -e PYTHONUNBUFFERED=1
  )
  if [[ -n "${ANNOUNCE_DOCKER_ENV:-}" ]]; then
    local item
    for item in $ANNOUNCE_DOCKER_ENV; do
      docker_args+=(-e "$item")
    done
  fi
  docker_args+=(
    -v "$ROOT":/workspace
    -w /workspace
    "$RUNNER_IMAGE"
  )

  if [[ "${ANNOUNCE_DOCKER_ROOT:-0}" == "1" ]]; then
    "${docker_args[@]}" \
      bash -c 'bash "$1"; status=$?; chown -R "$HOST_UID:$HOST_GID" /workspace/tests/artifacts/logs /workspace/.gocache /workspace/.gotmp /workspace/.gopath /workspace/.gomodcache 2>/dev/null || true; exit "$status"' _ "$rel_script"
  else
    "${docker_args[@]}" bash "$rel_script"
  fi
}

build_runner_image

scripts=()
for script in "$ANNOUNCE_INTERFACE_DIR"/*.sh; do
  name="$(basename "$script")"
  if [[ "$name" == "run.sh" || "$name" == "interface.sh" || "$name" == _* ]]; then
    continue
  fi
  [[ -f "$script" ]] || continue
  if ! should_run_script "$name"; then
    continue
  fi
  scripts+=("$script")
done

if [[ "${#scripts[@]}" -eq 0 ]]; then
  echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] no $ANNOUNCE_INTERFACE_NAME announce parity scripts yet"
  exit 0
fi

if [[ "$PARALLEL" == "1" ]]; then
  mkdir -p "$RUNNER_LOG_DIR"
  pids=()
  names=()
  logs=()

  for script in "${scripts[@]}"; do
    name="$(basename "$script")"
    log="$RUNNER_LOG_DIR/${name%.sh}.log"
    echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] start $name -> $log"
    (run_script "$script") >"$log" 2>&1 &
    pids+=("$!")
    names+=("$name")
    logs+=("$log")
  done

  status=0
  for i in "${!pids[@]}"; do
    if wait "${pids[$i]}"; then
      code=0
    else
      code="$?"
      status=1
    fi
    echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] ${names[$i]} exit=$code log=${logs[$i]}"
  done

  exit "$status"
fi

for script in "${scripts[@]}"; do
  name="$(basename "$script")"
  echo "[parity-announce-$ANNOUNCE_INTERFACE_NAME] $name"
  run_script "$script"
done
