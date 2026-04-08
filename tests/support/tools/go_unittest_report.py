#!/usr/bin/env python3
import json
import sys
import time


PYTHON_ORDER = [
    # tests/channel.py
    "tests.channel.TestChannel.test_buffer_big",
    "tests.channel.TestChannel.test_buffer_small_bidirectional",
    "tests.channel.TestChannel.test_buffer_small_with_callback",
    "tests.channel.TestChannel.test_multiple_handler",
    "tests.channel.TestChannel.test_send_one_retry",
    "tests.channel.TestChannel.test_send_receive_message_test",
    "tests.channel.TestChannel.test_send_timeout",
    "tests.channel.TestChannel.test_system_message_check",
    # tests/identity.py
    "tests.identity.TestIdentity.test_0_create_from_bytes",
    "tests.identity.TestIdentity.test_1_sign",
    "tests.identity.TestIdentity.test_2_encrypt",
    # tests/link.py
    "tests.link.TestLink.test_00_valid_announce",
    "tests.link.TestLink.test_01_invalid_announce",
    "tests.link.TestLink.test_02_establish",
    "tests.link.TestLink.test_03a_packets",
    "tests.link.TestLink.test_03b_packets",
    "tests.link.TestLink.test_04_micro_resource",
    "tests.link.TestLink.test_05_mini_resource",
    "tests.link.TestLink.test_06_small_resource",
    "tests.link.TestLink.test_07_medium_resource",
    "tests.link.TestLink.test_09_large_resource",
    "tests.link.TestLink.test_10_channel_round_trip",
    "tests.link.TestLink.test_11_buffer_round_trip",
    "tests.link.TestLink.test_12_buffer_round_trip_big",
    "tests.link.TestLink.test_13_buffer_round_trip_big_slow",
    # tests/hashes.py (sha256)
    "tests.hashes.TestSHA256.test_empty",
    "tests.hashes.TestSHA256.test_less_than_block_length",
    "tests.hashes.TestSHA256.test_block_length",
    "tests.hashes.TestSHA256.test_several_blocks",
    "tests.hashes.TestSHA256.test_random_blocks",
    # tests/hashes.py (sha512)
    "tests.hashes.TestSHA512.test_empty",
    "tests.hashes.TestSHA512.test_less_than_block_length",
    "tests.hashes.TestSHA512.test_block_length",
    "tests.hashes.TestSHA512.test_several_blocks",
    "tests.hashes.TestSHA512.test_random_blocks",
]

# Map Go test names (optionally with /subtests) to python unittest-style test ids.
GO_TO_PY = {
    # Channel (unit)
    "TestChannel_SendOneRetry": "tests.channel.TestChannel.test_send_one_retry",
    "TestChannel_SendTimeout": "tests.channel.TestChannel.test_send_timeout",
    "TestChannel_MultipleHandler": "tests.channel.TestChannel.test_multiple_handler",
    "TestChannel_SystemMessageCheck": "tests.channel.TestChannel.test_system_message_check",
    "TestChannel_SendReceiveMessageTest": "tests.channel.TestChannel.test_send_receive_message_test",
    "TestIntegration_ChannelRoundTrip": "tests.link.TestLink.test_10_channel_round_trip",
    # Buffer (integration)
    "TestIntegration_BufferRoundTrip_Small": "tests.link.TestLink.test_11_buffer_round_trip",
    "TestIntegration_BufferRoundTrip_Big": "tests.link.TestLink.test_12_buffer_round_trip_big",
    "TestIntegration_BufferRoundTrip_Big_Slow": "tests.link.TestLink.test_13_buffer_round_trip_big_slow",
    # Identity
    "TestIdentity_FromBytes_HashAndPrivateKey": "tests.identity.TestIdentity.test_0_create_from_bytes",
    "TestIdentity_Sign_KnownVector": "tests.identity.TestIdentity.test_1_sign",
    "TestIdentity_EncryptDecrypt_RandomSmallChunks": "tests.identity.TestIdentity.test_2_encrypt",
    # Link + announce
    "TestLink_ValidateAnnounce_Valid": "tests.link.TestLink.test_00_valid_announce",
    "TestLink_ValidateAnnounce_InvalidDestination": "tests.link.TestLink.test_01_invalid_announce",
    "TestIntegration_LinkEstablish_DefaultMode": "tests.link.TestLink.test_02_establish",
    "TestIntegration_LinkEstablish_AES256CBC_Mode": "tests.link.TestLink.test_02_establish",
    "TestIntegration_LinkEstablish_AES128CBC_ModeRejected": "tests.link.TestLink.test_02_establish",
    "TestIntegration_LinkPackets_WithReceipts": "tests.link.TestLink.test_03a_packets",
    # Resources
    "TestIntegration_Resource_MicroMiniSmall": "tests.link.TestLink.test_06_small_resource",
    "TestIntegration_Resource_MediumLarge_Slow": "tests.link.TestLink.test_09_large_resource",
    # Hashes
    "TestSHA256_RandomBlocks": "tests.hashes.TestSHA256.test_random_blocks",
    "TestSHA512_RandomBlocks": "tests.hashes.TestSHA512.test_random_blocks",
    "TestSHA256_KnownVectors": "tests.hashes.TestSHA256.test_several_blocks",
    "TestSHA512_KnownVectors": "tests.hashes.TestSHA512.test_several_blocks",
}


def split_py_id(py_id: str) -> tuple[str, str]:
    parts = py_id.rsplit(".", 1)
    if len(parts) != 2:
        return py_id, ""
    return parts[1], parts[0]


def main() -> int:
    started = time.time()
    py_results: dict[str, str] = {}
    extra_go_results: dict[tuple[str, str], str] = {}
    failures = 0
    skips = 0

    pkg_failed = {}
    test_errors: list[str] = []

    for raw in sys.stdin:
        raw = raw.strip()
        if not raw:
            continue
        try:
            ev = json.loads(raw)
        except json.JSONDecodeError:
            continue

        action = ev.get("Action")
        pkg = ev.get("Package", "")
        test = ev.get("Test", "")
        output = ev.get("Output", "")

        if test and action in ("pass", "fail", "skip"):
            if action == "pass":
                status = "ok"
            elif action == "skip":
                status = "skipped"
                skips += 1
            else:
                status = "FAIL"
                failures += 1

            base_name = test.split("/", 1)[0]
            py_id = GO_TO_PY.get(test) or GO_TO_PY.get(base_name)
            if py_id:
                prev = py_results.get(py_id)
                if prev == "FAIL":
                    pass
                elif status == "FAIL":
                    py_results[py_id] = status
                elif prev == "ok":
                    pass
                elif status == "ok":
                    py_results[py_id] = status
                else:
                    py_results[py_id] = status
            else:
                extra_go_results[(pkg, test)] = status
        else:
            if action == "fail":
                pkg_failed[pkg] = True
            if output and action == "output":
                if pkg_failed.get(pkg):
                    test_errors.append(output.rstrip("\n"))

    for py_id in PYTHON_ORDER:
        test_name, suite = split_py_id(py_id)
        if py_id in py_results:
            print(f"{test_name} ({suite}) ... {py_results[py_id]}")
        else:
            print(f"{test_name} ({suite}) ... skipped 'Not implemented in Go'")
            skips += 1

    for py_id in sorted(set(py_results.keys()) - set(PYTHON_ORDER)):
        test_name, suite = split_py_id(py_id)
        print(f"{test_name} ({suite}) ... {py_results[py_id]}")

    if extra_go_results:
        print()
        print("Additional Go-only tests:")
        for (pkg, test), status in sorted(extra_go_results.items()):
            print(f"{test} ({pkg}) ... {status}")

    elapsed = time.time() - started
    total = len(PYTHON_ORDER) + len(extra_go_results)
    print()
    print("-" * 70)
    print(f"Ran {total} tests in {elapsed:.3f}s")
    print()

    if failures == 0 and not any(pkg_failed.values()):
        if skips:
            print(f"OK (skipped={skips})")
        else:
            print("OK")
        return 0

    if test_errors:
        print("\n".join(test_errors))
    print(f"FAILED (failures={failures}, skipped={skips})")
    return 1


if __name__ == "__main__":
    raise SystemExit(main())

