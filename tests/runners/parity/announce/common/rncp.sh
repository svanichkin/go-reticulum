#!/usr/bin/env bash
set -euo pipefail

ANNOUNCE_TOOL_NAME="rncp"
source "$ROOT/tests/runners/parity/announce/common/tool_base.sh"
announce_init_bins rnsd rncp rnid
announce_prepare_run

extract_listen_hash() {
  local path="$1"
  rg "rncp listening on|Listening on :" "$path" \
    | sed -E 's/^.*<([0-9a-fA-F]+)>.*/\1/' \
    | head -n 1
}

start_rncp_listener() {
  local impl="$1"
  local cfg_dir="$2"
  local identity_path="$3"
  local recv_dir="$4"
  local announce_interval="$5"
  local log_path="$6"
  : >"$log_path"
  if [[ "$impl" == "go" ]]; then
    env HOME="$cfg_dir" USERPROFILE="$cfg_dir" \
      "$GO_RNCP" --config "$cfg_dir" -i "$identity_path" -l -n -b "$announce_interval" -S -s "$recv_dir" >"$log_path" 2>&1 &
  else
    env HOME="$cfg_dir" USERPROFILE="$cfg_dir" \
      "${PY_RNS[@]}" "$ROOT/python/RNS/Utilities/rncp.py" --config "$cfg_dir" -i "$identity_path" -l -n -b "$announce_interval" -S -s "$recv_dir" >"$log_path" 2>&1 &
  fi
  local pid=$!
  PIDS+=("$pid")
  echo "$pid"
}

run_direction() {
  local sender_impl="$1"
  local mode="$2"

  local receiver_impl="go"
  if [[ "$sender_impl" == "go" ]]; then
    receiver_impl="py"
  fi

  local label="${sender_impl}_to_${receiver_impl}_${mode}"
  local scenario_dir="$WORK_DIR/$label"
  mkdir -p "$scenario_dir"
  announce_prepare_scenario "$ANNOUNCE_TOOL_NAME" "$label"

  local shared_base="$ANNOUNCE_SHARED_BASE"
  local receiver_cfg="$scenario_dir/receiver_cfg"
  local sender_cfg="$scenario_dir/sender_cfg"
  local sender_rnsd_cfg="$scenario_dir/sender_rnsd_cfg"
  mkdir -p "$receiver_cfg" "$sender_cfg" "$sender_rnsd_cfg"

  announce_write_rnsd_config "$receiver_cfg" "${label}_receiver_${receiver_impl}" \
    $((shared_base + 2)) $((shared_base + 3)) \
    "$ANNOUNCE_INTERFACE_SHORT_LABEL Receiver ${receiver_impl^}" receiver

  if [[ "$mode" == "standalone" ]]; then
    announce_write_standalone_config "$sender_cfg" "${label}_sender_${sender_impl}" "$ANNOUNCE_TOOL_NAME"
  else
    announce_write_rnsd_config "$sender_rnsd_cfg" "${label}_sender_rnsd_${sender_impl}" \
      "$shared_base" $((shared_base + 1)) "$ANNOUNCE_INTERFACE_SHORT_LABEL Sender Local" sender
    announce_write_local_client_config "$sender_cfg" "${label}_sender_client_${sender_impl}" \
      "$shared_base" $((shared_base + 1))
  fi

  local receiver_log="$OUT_DIR/${label}.receiver.rnsd.log"
  local sender_rnsd_log="$OUT_DIR/${label}.sender.rnsd.log"
  local listener_log="$OUT_DIR/${label}.listener.log"
  local receiver_pid="" sender_side_pid="" listener_pid=""

  echo "$(announce_prefix) $label: start receiver rnsd"
  if [[ "$receiver_impl" == "go" ]]; then
    receiver_pid="$(announce_start_go_rnsd "$receiver_cfg" "$receiver_log")"
  else
    receiver_pid="$(announce_start_py_rnsd "$receiver_cfg" "$receiver_log")"
  fi
  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$receiver_log" "Started rnsd version"; then
    echo "$(announce_prefix) $label: receiver rnsd did not start; see $receiver_log"
    stop_proc "$receiver_pid"
    return 1
  fi
  if [[ "$mode" == "local" ]]; then
    echo "$(announce_prefix) $label: start sender-side local rnsd"
    if [[ "$sender_impl" == "go" ]]; then
      sender_side_pid="$(announce_start_go_rnsd "$sender_rnsd_cfg" "$sender_rnsd_log")"
    else
      sender_side_pid="$(announce_start_py_rnsd "$sender_rnsd_cfg" "$sender_rnsd_log")"
    fi
    if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$sender_rnsd_log" "Started rnsd version"; then
      echo "$(announce_prefix) $label: sender-side rnsd did not start; see $sender_rnsd_log"
      stop_proc "$sender_side_pid"; stop_proc "$receiver_pid"
      return 1
    fi
    if ! announce_wait_sender_ready "$sender_rnsd_log" "$label"; then
      stop_proc "$sender_side_pid"; stop_proc "$receiver_pid"
      return 1
    fi
  fi

  local identity_path="$scenario_dir/listener.id"
  local recv_dir="$scenario_dir/recv"
  mkdir -p "$recv_dir"
  local generate_out="$OUT_DIR/${label}.generate.out"
  local code
  code="$(announce_run_generate_identity "$sender_impl" "$sender_cfg" "$identity_path" "$generate_out")"
  if [[ "$code" != "0" ]] || [[ ! -f "$identity_path" ]]; then
    echo "$(announce_prefix) $label: identity generation failed; see $generate_out"
    stop_proc "$sender_side_pid"; stop_proc "$receiver_pid"
    return 1
  fi

  echo "$(announce_prefix) $label: start rncp listener"
  listener_pid="$(start_rncp_listener "$sender_impl" "$sender_cfg" "$identity_path" "$recv_dir" 1 "$listener_log")"
  if ! wait_for_file_contains "$START_TIMEOUT_SECS" "$listener_log" "rncp listening on"; then
    echo "$(announce_prefix) $label: listener not ready; see $listener_log"
    stop_proc "$listener_pid"; stop_proc "$sender_side_pid"; stop_proc "$receiver_pid"
    return 1
  fi

  local listen_hash
  listen_hash="$(extract_listen_hash "$listener_log")"
  if [[ -z "$listen_hash" ]]; then
    echo "$(announce_prefix) $label: could not parse listener hash; see $listener_log"
    stop_proc "$listener_pid"; stop_proc "$sender_side_pid"; stop_proc "$receiver_pid"
    return 1
  fi

  local announce_pattern="Valid announce for <$listen_hash>|Destination <$listen_hash> is now"
  local receiver_from_line
  receiver_from_line=$(( $(wc -l <"$receiver_log") + 1 ))
  if ! wait_for_new_log_match "$START_TIMEOUT_SECS" "$receiver_log" "$receiver_from_line" "$announce_pattern"; then
    echo "$(announce_prefix) $label: receiver did not log initial announce; see $receiver_log"
    stop_proc "$listener_pid"; stop_proc "$sender_side_pid"; stop_proc "$receiver_pid"
    return 1
  fi

  receiver_from_line=$(( $(wc -l <"$receiver_log") + 1 ))
  if ! wait_for_new_log_match "$START_TIMEOUT_SECS" "$receiver_log" "$receiver_from_line" "$announce_pattern"; then
    echo "$(announce_prefix) $label: receiver did not log periodic reannounce; see $receiver_log"
    stop_proc "$listener_pid"; stop_proc "$sender_side_pid"; stop_proc "$receiver_pid"
    return 1
  fi

  {
    echo "sender_impl=$sender_impl"
    echo "receiver_impl=$receiver_impl"
    echo "mode=$mode"
    echo "listen_hash=$listen_hash"
    echo "announce_ok=yes"
    echo "reannounce_ok=yes"
  } >"$OUT_DIR/${label}.summary.txt"

  stop_proc "$listener_pid"
  stop_proc "$sender_side_pid"
  stop_proc "$receiver_pid"
  echo "$(announce_prefix) $label: OK"
  return 0
}

announce_run_four_directions

if [[ "$overall" -ne 0 ]]; then
  echo "$(announce_prefix) rncp $ANNOUNCE_INTERFACE_LABEL announce checks FAILED"
  exit 1
fi

echo "$(announce_prefix) rncp $ANNOUNCE_INTERFACE_LABEL announce checks passed"
