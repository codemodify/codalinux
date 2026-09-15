#!/usr/bin/env python3
import importlib.util
import unittest
from pathlib import Path

_SPEC = importlib.util.spec_from_file_location(
    "coda_install_layout",
    Path(__file__).with_name("coda-install-layout.py"),
)
mod = importlib.util.module_from_spec(_SPEC)
assert _SPEC.loader is not None
_SPEC.loader.exec_module(mod)


def bytes_from_mib(n: int) -> int:
    return n * 1024 * 1024


class LayoutTests(unittest.TestCase):
    def test_minimum_constant(self):
        self.assertEqual(
            mod.MINIMUM_DISK_MIB,
            mod.ESP_MIB
            + 2 * mod.SLOT_FLOOR_MIB
            + mod.DATA_FLOOR_MIB
            + mod.GPT_SLACK_MIB,
        )
        # 1024 + 4096 + 4096 + 8192 + 4
        self.assertEqual(mod.MINIMUM_DISK_MIB, 17412)
        self.assertEqual(mod.SLOT_FLOOR_MIB, 4096)
        self.assertEqual(mod.DATA_FLOOR_MIB, 8192)
        self.assertEqual(mod.ESP_MIB, 1024)
        self.assertEqual(mod.CORE_ONLY_MINIMUM_DISK_MIB, mod.MINIMUM_DISK_MIB)

    def test_too_small(self):
        self.assertIsNone(mod.plan_layout(bytes_from_mib(mod.MINIMUM_DISK_MIB - 1)))
        self.assertIn("too small", mod.explain_too_small(bytes_from_mib(16 * 1024)))
        self.assertIn("core only", mod.explain_too_small(bytes_from_mib(16 * 1024)))

    def test_exact_minimum_uses_floors(self):
        plan = mod.plan_layout(bytes_from_mib(mod.MINIMUM_DISK_MIB))
        self.assertIsNotNone(plan)
        assert plan is not None
        self.assertEqual(plan["esp_mib"], 1024)
        self.assertEqual(plan["slot_a_mib"], mod.SLOT_FLOOR_MIB)
        self.assertEqual(plan["slot_b_mib"], mod.SLOT_FLOOR_MIB)
        self.assertEqual(plan["data_mib"], mod.DATA_FLOOR_MIB)

    def test_16g_rejected_data_needs_desktop(self):
        # 16 GiB < 17 GiB minimum (desktop now lives on data).
        self.assertIsNone(mod.plan_layout(16 * 1024 * 1024 * 1024))

    def test_22g_accepted_core_only_slots(self):
        plan = mod.plan_layout(22 * 1024 * 1024 * 1024)
        self.assertIsNotNone(plan)
        assert plan is not None
        self.assertEqual(plan["slot_a_mib"], 4096)
        self.assertEqual(plan["slot_b_mib"], 4096)
        self.assertGreaterEqual(plan["data_mib"], mod.DATA_FLOOR_MIB)

    def test_32g_gives_extra_to_data(self):
        plan = mod.plan_layout(32 * 1024 * 1024 * 1024)
        self.assertIsNotNone(plan)
        assert plan is not None
        self.assertEqual(plan["slot_a_mib"], mod.SLOT_PREFERRED_MIB)
        self.assertEqual(plan["slot_b_mib"], mod.SLOT_PREFERRED_MIB)
        self.assertGreaterEqual(plan["data_mib"], mod.DATA_FLOOR_MIB)
        self.assertGreaterEqual(plan["data_mib"], 14 * 1024)

    def test_parts_sum_with_slack(self):
        for size_g in (18, 22, 32, 64):
            plan = mod.plan_layout(size_g * 1024 * 1024 * 1024)
            self.assertIsNotNone(plan, size_g)
            assert plan is not None
            total = (
                plan["esp_mib"]
                + plan["slot_a_mib"]
                + plan["slot_b_mib"]
                + plan["data_mib"]
                + plan["slack_mib"]
            )
            self.assertLessEqual(total, plan["disk_mib"])


if __name__ == "__main__":
    unittest.main()
