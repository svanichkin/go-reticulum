#!/usr/bin/env bash

ANNOUNCE_INTERFACE_NAME="tcp_local"
ANNOUNCE_INTERFACE_LABEL="TCP local"
ANNOUNCE_INTERFACE_SHORT_LABEL="TCP"
ANNOUNCE_OBSERVE_PATTERN="${ANNOUNCE_OBSERVE_PATTERN:-"Valid announce for|Destination .* is now .* on .*TCP|Reconnected socket for TCPInterface"}"
START_TIMEOUT_SECS="${START_TIMEOUT_SECS:-35}"

announce_prepare_scenario() {
  local _tool="$1"
  local _label="$2"
  ANNOUNCE_SHARED_BASE=$(( (RANDOM % 10000) + 38000 ))
  ANNOUNCE_SHARED_BASE=$(( ANNOUNCE_SHARED_BASE / 4 * 4 ))
  ANNOUNCE_TCP_PORT=$(( (RANDOM % 10000) + 52000 ))
}

announce_write_server_config() {
  local cfg_dir="$1"
  local instance_name="$2"
  local shared_port="$3"
  local control_port="$4"
  local if_name="$5"
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
    type = TCPServerInterface
    enabled = yes
    listen_ip = 127.0.0.1
    listen_port = $ANNOUNCE_TCP_PORT
EOF
}

announce_write_client_transport_config() {
  local cfg_dir="$1"
  local instance_name="$2"
  local shared_port="$3"
  local control_port="$4"
  local if_name="$5"
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
    type = TCPClientInterface
    enabled = yes
    target_host = 127.0.0.1
    target_port = $ANNOUNCE_TCP_PORT
EOF
}

announce_write_rnsd_config() {
  local cfg_dir="$1"
  local instance_name="$2"
  local shared_port="$3"
  local control_port="$4"
  local if_name="$5"
  local role="$6"
  if [[ "$role" == "receiver" ]]; then
    announce_write_server_config "$cfg_dir" "$instance_name" "$shared_port" "$control_port" "$if_name"
  else
    announce_write_client_transport_config "$cfg_dir" "$instance_name" "$shared_port" "$control_port" "$if_name"
  fi
}

announce_write_standalone_config() {
  local cfg_dir="$1"
  local instance_name="$2"
  local tool="$3"
  local if_name="TCP Client"
  if [[ "$tool" == "rnid" ]]; then
    if_name="TCP Sender"
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
    type = TCPClientInterface
    enabled = yes
    target_host = 127.0.0.1
    target_port = $ANNOUNCE_TCP_PORT
EOF
}

announce_write_rnsd_configs() {
  local py_dir="$1"
  local go_dir="$2"
  ANNOUNCE_TCP_PORT=50010
  announce_write_rnsd_config "$py_dir" "parity_py_tcp_local" 37432 37433 "TCP Py" receiver
  announce_write_rnsd_config "$go_dir" "parity_go_tcp_local" 37430 37431 "TCP Go" sender
}

announce_wait_sender_ready() {
  local log_path="$1"
  local label="$2"
  if wait_for_file_contains "$START_TIMEOUT_SECS" "$log_path" "Connected to server|Spawned TCP client|Establishing TCP connection"; then
    return 0
  fi
  echo "$(announce_prefix) $label: sender-side rnsd did not establish TCP path; see $log_path"
  return 1
}
