#!/usr/bin/env bash

ANNOUNCE_INTERFACE_NAME="auto"
ANNOUNCE_INTERFACE_LABEL="AutoInterface"
ANNOUNCE_INTERFACE_SHORT_LABEL="Auto"
ANNOUNCE_OBSERVE_PATTERN="${ANNOUNCE_OBSERVE_PATTERN:-"Valid announce for|Destination .* is now .* on .*AutoInterface"}"
ANNOUNCE_PY_RNSD_MODE="${ANNOUNCE_PY_RNSD_MODE:-wrapper}"
START_TIMEOUT_SECS="${START_TIMEOUT_SECS:-45}"

announce_interface_setup() {
  if ! command -v ip >/dev/null 2>&1; then
    return 0
  fi
  ip link add auto_rx type veth peer name auto_tx >/dev/null 2>&1 || true
  ip link set dev auto_rx addrgenmode none >/dev/null 2>&1 || true
  ip link set dev auto_tx addrgenmode none >/dev/null 2>&1 || true
  ip link set auto_rx up >/dev/null 2>&1 || true
  ip link set auto_tx up >/dev/null 2>&1 || true
  ip -6 addr flush dev auto_rx scope link >/dev/null 2>&1 || true
  ip -6 addr flush dev auto_tx scope link >/dev/null 2>&1 || true
  ip -6 addr add fe80::a001/64 dev auto_rx nodad >/dev/null 2>&1 || true
  ip -6 addr add fe80::a002/64 dev auto_tx nodad >/dev/null 2>&1 || true
}

announce_prepare_scenario() {
  local tool="$1"
  local label="$2"
  ANNOUNCE_SHARED_BASE=$(( (RANDOM % 10000) + 38000 ))
  ANNOUNCE_SHARED_BASE=$(( ANNOUNCE_SHARED_BASE / 4 * 4 ))
  ANNOUNCE_AUTO_GROUP_ID="parity-auto-${tool}-${label}"
  ANNOUNCE_RECEIVER_DEVICE="auto_rx"
  ANNOUNCE_SENDER_DEVICE="auto_tx"
}

announce_modes_for_tool() {
  local tool="$1"
  case "$tool" in
    rnid|rnx)
      echo "local"
      ;;
    *)
      echo "standalone local"
      ;;
  esac
}

announce_write_rnsd_config() {
  local cfg_dir="$1"
  local instance_name="$2"
  local shared_port="$3"
  local control_port="$4"
  local if_name="$5"
  local role="$6"
  local device="$ANNOUNCE_RECEIVER_DEVICE"
  if [[ "$role" == "sender" ]]; then
    device="$ANNOUNCE_SENDER_DEVICE"
  fi
  cat >"$cfg_dir/config" <<EOF
[reticulum]
  share_instance = Yes
  instance_name = $instance_name
  shared_instance_type = tcp
  shared_instance_port = $shared_port
  instance_control_port = $control_port
  enable_transport = Yes
  respond_to_probes = Yes

[logging]
  loglevel = 7

[interfaces]
  [[$if_name]]
    type = AutoInterface
    enabled = yes
    group_id = $ANNOUNCE_AUTO_GROUP_ID
    devices = $device
EOF
}

announce_write_standalone_config() {
  local cfg_dir="$1"
  local instance_name="$2"
  local tool="$3"
  local if_name="Auto Listener"
  if [[ "$tool" == "rnid" ]]; then
    if_name="Auto Sender"
  fi
  cat >"$cfg_dir/config" <<EOF
[reticulum]
  share_instance = No
  instance_name = $instance_name
  enable_transport = Yes

[logging]
  loglevel = 7

[interfaces]
  [[$if_name]]
    type = AutoInterface
    enabled = yes
    group_id = $ANNOUNCE_AUTO_GROUP_ID
    devices = $ANNOUNCE_SENDER_DEVICE
EOF
}

announce_write_rnsd_configs() {
  local py_dir="$1"
  local go_dir="$2"
  ANNOUNCE_AUTO_GROUP_ID="parity-auto-rnsd"
  ANNOUNCE_RECEIVER_DEVICE="auto_rx"
  ANNOUNCE_SENDER_DEVICE="auto_tx"
  announce_write_rnsd_config "$py_dir" "parity_py_auto" 37432 37433 "Auto Py" receiver
  announce_write_rnsd_config "$go_dir" "parity_go_auto" 37430 37431 "Auto Go" sender
}

announce_wait_sender_ready() {
  local log_path="$1"
  local label="$2"
  if wait_for_file_contains "$START_TIMEOUT_SECS" "$log_path" "added peer"; then
    return 0
  fi
  echo "$(announce_prefix) $label: sender-side rnsd did not discover receiver; see $log_path"
  return 1
}
