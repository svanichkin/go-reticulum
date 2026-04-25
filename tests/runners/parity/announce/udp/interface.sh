#!/usr/bin/env bash

ANNOUNCE_INTERFACE_NAME="udp"
ANNOUNCE_INTERFACE_LABEL="UDP"
ANNOUNCE_INTERFACE_SHORT_LABEL="UDP"
ANNOUNCE_OBSERVE_PATTERN="${ANNOUNCE_OBSERVE_PATTERN:-"Valid announce for|Destination .* is now .* on .*UDP"}"
START_TIMEOUT_SECS="${START_TIMEOUT_SECS:-25}"

announce_prepare_scenario() {
  local _tool="$1"
  local _label="$2"
  local base
  base=$(( (RANDOM % 10000) + 52000 ))
  base=$(( base / 2 * 2 ))
  ANNOUNCE_SHARED_BASE=$(( (RANDOM % 10000) + 38000 ))
  ANNOUNCE_SHARED_BASE=$(( ANNOUNCE_SHARED_BASE / 4 * 4 ))
  ANNOUNCE_SENDER_UDP_PORT="$base"
  ANNOUNCE_RECEIVER_UDP_PORT=$((base + 1))
}

announce_write_rnsd_config() {
  local cfg_dir="$1"
  local instance_name="$2"
  local shared_port="$3"
  local control_port="$4"
  local if_name="$5"
  local role="$6"
  local listen_port="$ANNOUNCE_RECEIVER_UDP_PORT"
  local forward_port="$ANNOUNCE_SENDER_UDP_PORT"
  if [[ "$role" == "sender" ]]; then
    listen_port="$ANNOUNCE_SENDER_UDP_PORT"
    forward_port="$ANNOUNCE_RECEIVER_UDP_PORT"
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
    type = UDPInterface
    enabled = yes
    listen_ip = 127.0.0.1
    listen_port = $listen_port
    forward_ip = 127.0.0.1
    forward_port = $forward_port
EOF
}

announce_write_standalone_config() {
  local cfg_dir="$1"
  local instance_name="$2"
  local tool="$3"
  local if_name="UDP Listener"
  if [[ "$tool" == "rnid" ]]; then
    if_name="UDP Sender"
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
    type = UDPInterface
    enabled = yes
    listen_ip = 127.0.0.1
    listen_port = $ANNOUNCE_SENDER_UDP_PORT
    forward_ip = 127.0.0.1
    forward_port = $ANNOUNCE_RECEIVER_UDP_PORT
EOF
}

announce_write_rnsd_configs() {
  local py_dir="$1"
  local go_dir="$2"
  ANNOUNCE_SENDER_UDP_PORT=50000
  ANNOUNCE_RECEIVER_UDP_PORT=50001
  announce_write_rnsd_config "$py_dir" "parity_py_udp" 37432 37433 "UDP Py" receiver
  announce_write_rnsd_config "$go_dir" "parity_go_udp" 37430 37431 "UDP Go" sender
}

announce_wait_sender_ready() {
  local sender_log="$1"
  local label="$2"
  local receiver_log="$OUT_DIR/${label}.receiver.rnsd.log"
  local pattern="Valid announce for .* on UDPInterface|Destination .* is now .* on .*UDP"
  local start
  start="$(date +%s)"
  while true; do
    if [[ -f "$sender_log" ]] && rg -q "$pattern" "$sender_log"; then
      return 0
    fi
    if [[ -f "$receiver_log" ]] && rg -q "$pattern" "$receiver_log"; then
      return 0
    fi
    local now
    now="$(date +%s)"
    if [[ $((now - start)) -ge "$START_TIMEOUT_SECS" ]]; then
      echo "$(announce_prefix) $label: sender-side rnsd did not learn remote UDP path; see $sender_log and $receiver_log"
      return 1
    fi
    sleep 0.1
  done
}

announce_wait_before_reannounce() {
  local _label="$1"
  local mode="$2"
  local _sender_impl="$3"
  if [[ "$mode" == "local" ]]; then
    sleep 6
  fi
}
