#!/usr/bin/env bash

ANNOUNCE_INTERFACE_NAME="tcp_bootstrap"
ANNOUNCE_INTERFACE_LABEL="TCP bootstrap"
ANNOUNCE_INTERFACE_SHORT_LABEL="TCP"
ANNOUNCE_OBSERVE_PATTERN="${ANNOUNCE_OBSERVE_PATTERN:-"Valid announce for|Destination .* is now .* on .*TCP|Connected to server|TCP connection for .* established|Reconnected socket for TCPInterface"}"
START_TIMEOUT_SECS="${START_TIMEOUT_SECS:-60}"

announce_prepare_scenario() {
  local _tool="$1"
  local _label="$2"
  ANNOUNCE_SHARED_BASE=$(( (RANDOM % 10000) + 38000 ))
  ANNOUNCE_SHARED_BASE=$(( ANNOUNCE_SHARED_BASE / 4 * 4 ))
}

announce_tcp_bootstrap_interfaces() {
  cat <<'EOF'
  [[TCP bootstrap dead]]
    type = TCPClientInterface
    enabled = yes
    target_host = 212.233.88.164
    target_port = 4242

  [[TCP bootstrap dublin]]
    type = TCPClientInterface
    enabled = yes
    target_host = dublin.connect.reticulum.network
    target_port = 4965
EOF
}

announce_write_shared_bootstrap_config() {
  local cfg_dir="$1"
  local instance_name="$2"
  local shared_port="$3"
  local control_port="$4"
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
$(announce_tcp_bootstrap_interfaces)
EOF
}

announce_write_rnsd_config() {
  local cfg_dir="$1"
  local instance_name="$2"
  local shared_port="$3"
  local control_port="$4"
  local _if_name="$5"
  local _role="$6"
  announce_write_shared_bootstrap_config "$cfg_dir" "$instance_name" "$shared_port" "$control_port"
}

announce_write_standalone_config() {
  local cfg_dir="$1"
  local instance_name="$2"
  local _tool="$3"
  cat >"$cfg_dir/config" <<EOF
[reticulum]
  share_instance = No
  instance_name = $instance_name
  enable_transport = Yes

[logging]
  loglevel = 7

[interfaces]
$(announce_tcp_bootstrap_interfaces)
EOF
}

announce_write_rnsd_configs() {
  local py_dir="$1"
  local go_dir="$2"
  announce_write_shared_bootstrap_config "$py_dir" "parity_py_tcp_bootstrap" 37432 37433
  announce_write_shared_bootstrap_config "$go_dir" "parity_go_tcp_bootstrap" 37430 37431
}

announce_wait_sender_ready() {
  local log_path="$1"
  local label="$2"
  if wait_for_file_contains "$START_TIMEOUT_SECS" "$log_path" "Valid announce for|Destination .* is now .* on .*TCP"; then
    return 0
  fi
  echo "$(announce_prefix) $label: sender-side bootstrap transport did not observe any announce before sender start; see $log_path"
  return 1
}

announce_wait_receiver_ready() {
  local log_path="$1"
  local label="$2"
  if wait_for_file_contains "$START_TIMEOUT_SECS" "$log_path" "Valid announce for|Destination .* is now .* on .*TCP"; then
    return 0
  fi
  echo "$(announce_prefix) $label: receiver did not observe any bootstrap announce before sender start; see $log_path"
  return 1
}
