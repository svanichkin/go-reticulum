#!/usr/bin/env bash
set -euo pipefail

ANNOUNCE_TOOL_NAME="rnid"
ANNOUNCE_GENERATE_QUIET=1
source "$ROOT/tests/runners/parity/announce/common/tool_base.sh"
announce_init_bins rnsd rnid
announce_prepare_run

extract_dest_hash() {
  local path="$1"
  rg "destination for this Identity is|Announcing destination" "$path" \
    | sed -E 's/^.*<([0-9a-fA-F]+)>.*/\1/' \
    | head -n 1
}

run_hash_identity() {
  local impl="$1"
  local cfg_dir="$2"
  local identity_path="$3"
  local aspect="$4"
  local out="$5"
  if [[ "$impl" == "go" ]]; then
    run_capture "$out" env HOME="$cfg_dir" USERPROFILE="$cfg_dir" \
      "$GO_RNID" --config "$cfg_dir" -i "$identity_path" -H "$aspect"
  else
    run_capture "$out" env HOME="$cfg_dir" USERPROFILE="$cfg_dir" \
      "${PY_RNS[@]}" "$ROOT/python/RNS/Utilities/rnid.py" --config "$cfg_dir" -i "$identity_path" -H "$aspect"
  fi
}

run_announce_identity() {
  local impl="$1"
  local cfg_dir="$2"
  local identity_path="$3"
  local aspect="$4"
  local out="$5"
  if [[ "$impl" == "go" ]]; then
    run_capture "$out" env HOME="$cfg_dir" USERPROFILE="$cfg_dir" \
      "$GO_RNID" --config "$cfg_dir" -i "$identity_path" -a "$aspect"
  else
    run_capture "$out" env HOME="$cfg_dir" USERPROFILE="$cfg_dir" \
      "${PY_RNS[@]}" "$ROOT/python/RNS/Utilities/rnid.py" --config "$cfg_dir" -i "$identity_path" -a "$aspect"
  fi
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
  local receiver_pid="" sender_side_pid=""

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

  local identity_path="$scenario_dir/sender.id"
  local generate_out="$OUT_DIR/${label}.generate.out"
  local code
  code="$(announce_run_generate_identity "$sender_impl" "$sender_cfg" "$identity_path" "$generate_out")"
  if [[ "$code" != "0" ]] || [[ ! -f "$identity_path" ]]; then
    echo "$(announce_prefix) $label: identity generation failed; see $generate_out"
    stop_proc "$sender_side_pid"; stop_proc "$receiver_pid"
    return 1
  fi

  local hash_out="$OUT_DIR/${label}.hash.out"
  local dest_hash
  code="$(run_hash_identity "$sender_impl" "$sender_cfg" "$identity_path" "app.aspect" "$hash_out")"
  if [[ "$code" != "0" ]]; then
    echo "$(announce_prefix) $label: hash failed; see $hash_out"
    stop_proc "$sender_side_pid"; stop_proc "$receiver_pid"
    return 1
  fi
  dest_hash="$(extract_dest_hash "$hash_out")"
  if [[ -z "$dest_hash" ]]; then
    echo "$(announce_prefix) $label: could not parse destination hash; see $hash_out"
    stop_proc "$sender_side_pid"; stop_proc "$receiver_pid"
    return 1
  fi

  local announce_pattern="Valid announce for <$dest_hash>|Destination <$dest_hash> is now"
  local announce_one_out="$OUT_DIR/${label}.announce.one.out"
  local announce_two_out="$OUT_DIR/${label}.announce.two.out"
  local receiver_from_line

  receiver_from_line=$(( $(wc -l <"$receiver_log") + 1 ))
  code="$(run_announce_identity "$sender_impl" "$sender_cfg" "$identity_path" "app.aspect" "$announce_one_out")"
  if [[ "$code" != "0" ]] || ! rg -q "Created destination|Announcing destination" "$announce_one_out"; then
    echo "$(announce_prefix) $label: first announce failed; see $announce_one_out"
    stop_proc "$sender_side_pid"; stop_proc "$receiver_pid"
    return 1
  fi
  if ! wait_for_new_log_match "$START_TIMEOUT_SECS" "$receiver_log" "$receiver_from_line" "$announce_pattern"; then
    echo "$(announce_prefix) $label: receiver did not log first announce; see $receiver_log"
    stop_proc "$sender_side_pid"; stop_proc "$receiver_pid"
    return 1
  fi

  receiver_from_line=$(( $(wc -l <"$receiver_log") + 1 ))
  code="$(run_announce_identity "$sender_impl" "$sender_cfg" "$identity_path" "app.aspect" "$announce_two_out")"
  if [[ "$code" != "0" ]] || ! rg -q "Created destination|Announcing destination" "$announce_two_out"; then
    echo "$(announce_prefix) $label: second announce failed; see $announce_two_out"
    stop_proc "$sender_side_pid"; stop_proc "$receiver_pid"
    return 1
  fi
  if ! wait_for_new_log_match "$START_TIMEOUT_SECS" "$receiver_log" "$receiver_from_line" "$announce_pattern"; then
    echo "$(announce_prefix) $label: receiver did not log reannounce; see $receiver_log"
    stop_proc "$sender_side_pid"; stop_proc "$receiver_pid"
    return 1
  fi

  {
    echo "sender_impl=$sender_impl"
    echo "receiver_impl=$receiver_impl"
    echo "mode=$mode"
    echo "dest_hash=$dest_hash"
    echo "announce_ok=yes"
    echo "reannounce_ok=yes"
  } >"$OUT_DIR/${label}.summary.txt"

  stop_proc "$sender_side_pid"
  stop_proc "$receiver_pid"
  echo "$(announce_prefix) $label: OK"
  return 0
}

announce_run_four_directions

if [[ "$overall" -ne 0 ]]; then
  echo "$(announce_prefix) rnid $ANNOUNCE_INTERFACE_LABEL announce checks FAILED"
  exit 1
fi

echo "$(announce_prefix) rnid $ANNOUNCE_INTERFACE_LABEL announce checks passed"
