#!/usr/bin/env bash

ANNOUNCE_INTERFACE_NAME="pipe"
ANNOUNCE_INTERFACE_LABEL="PipeInterface"
ANNOUNCE_INTERFACE_SHORT_LABEL="Pipe"
ANNOUNCE_OBSERVE_PATTERN="${ANNOUNCE_OBSERVE_PATTERN:-"Valid announce for|Destination .* is now .* on .*PipeInterface"}"
START_TIMEOUT_SECS="${START_TIMEOUT_SECS:-25}"

announce_prepare_scenario() {
  local tool="$1"
  local label="$2"
  ANNOUNCE_SHARED_BASE=$(( (RANDOM % 10000) + 38000 ))
  ANNOUNCE_SHARED_BASE=$(( ANNOUNCE_SHARED_BASE / 4 * 4 ))

  local scenario_dir="$WORK_DIR/$label"
  ANNOUNCE_PIPE_A_TO_B="$scenario_dir/${tool}.a_to_b.pipe"
  ANNOUNCE_PIPE_B_TO_A="$scenario_dir/${tool}.b_to_a.pipe"
  rm -f "$ANNOUNCE_PIPE_A_TO_B" "$ANNOUNCE_PIPE_B_TO_A"
  mkfifo "$ANNOUNCE_PIPE_A_TO_B" "$ANNOUNCE_PIPE_B_TO_A"
}

announce_write_pipe_config() {
  local cfg_dir="$1"
  local instance_name="$2"
  local shared_port="$3"
  local control_port="$4"
  local if_name="$5"
  local role="$6"
  local rx_path="$ANNOUNCE_PIPE_A_TO_B"
  local tx_path="$ANNOUNCE_PIPE_B_TO_A"
  if [[ "$role" == "sender" ]]; then
    rx_path="$ANNOUNCE_PIPE_B_TO_A"
    tx_path="$ANNOUNCE_PIPE_A_TO_B"
  fi
  local helper="$ROOT/tests/runners/parity/announce/common/pipe_stdio_bridge.py"
  local command="python3 $helper --rx $rx_path --tx $tx_path"
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
    type = PipeInterface
    enabled = yes
    command = $command
    respawn_delay = 0.1
EOF
}

announce_write_rnsd_config() {
  local cfg_dir="$1"
  local instance_name="$2"
  local shared_port="$3"
  local control_port="$4"
  local if_name="$5"
  local role="$6"
  announce_write_pipe_config "$cfg_dir" "$instance_name" "$shared_port" "$control_port" "$if_name" "$role"
}

announce_write_standalone_config() {
  local cfg_dir="$1"
  local instance_name="$2"
  local tool="$3"
  local if_name="Pipe Listener"
  if [[ "$tool" == "rnid" ]]; then
    if_name="Pipe Sender"
  fi
  local helper="$ROOT/tests/runners/parity/announce/common/pipe_stdio_bridge.py"
  local command="python3 $helper --rx $ANNOUNCE_PIPE_B_TO_A --tx $ANNOUNCE_PIPE_A_TO_B"
  cat >"$cfg_dir/config" <<EOF
[reticulum]
  share_instance = No
  instance_name = $instance_name
  enable_transport = Yes

[logging]
  loglevel = 7

[interfaces]
  [[$if_name]]
    type = PipeInterface
    enabled = yes
    command = $command
    respawn_delay = 0.1
EOF
}

announce_write_rnsd_configs() {
  local py_dir="$1"
  local go_dir="$2"
  ANNOUNCE_SHARED_BASE=37430
  local scenario_dir="$WORK_DIR/rnsd_pair"
  mkdir -p "$scenario_dir"
  ANNOUNCE_PIPE_A_TO_B="$scenario_dir/rnsd.a_to_b.pipe"
  ANNOUNCE_PIPE_B_TO_A="$scenario_dir/rnsd.b_to_a.pipe"
  rm -f "$ANNOUNCE_PIPE_A_TO_B" "$ANNOUNCE_PIPE_B_TO_A"
  mkfifo "$ANNOUNCE_PIPE_A_TO_B" "$ANNOUNCE_PIPE_B_TO_A"
  announce_write_rnsd_config "$py_dir" "parity_py_pipe" 37432 37433 "Pipe Py" receiver
  announce_write_rnsd_config "$go_dir" "parity_go_pipe" 37430 37431 "Pipe Go" sender
}

announce_wait_sender_ready() {
  local log_path="$1"
  local label="$2"
  if wait_for_file_contains "$START_TIMEOUT_SECS" "$log_path" "Valid announce for .* on PipeInterface|Destination .* is now .* on PipeInterface"; then
    return 0
  fi
  echo "$(announce_prefix) $label: sender-side rnsd did not learn remote pipe path; see $log_path"
  return 1
}
