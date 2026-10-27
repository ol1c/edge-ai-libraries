// Copyright (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"strings"
	"testing"
)

const optionalExposition = `temp_highres_temp{host="itest",sensor="coretemp_package_id_0"} 45
temp_highres_temp_min{host="itest",sensor="coretemp_package_id_0"} 40
temp_highres_temp_max{host="itest",sensor="coretemp_package_id_0"} 54
temp_highres_temp_mean{host="itest",sensor="coretemp_package_id_0"} 47
temp_highres_temp_max{host="itest",sensor="coretemp_core_id_0"} 99
powerstat_package_current_dram_power_consumption_watts{host="itest",package_id="0"} 1.5
powerstat_core_cpu_frequency_mhz{host="itest",cpu_id="0"} 2000
powerstat_core_cpu_temperature_celsius{host="itest",cpu_id="0"} 52
interrupts_total{host="itest",irq="42",device="nvme0q1"} 1234
ethtool_rx_errors{host="itest",interface="enp1s0"} 3
`

func TestOptionalPanelsAppearOnlyWithSamples(t *testing.T) {
	baseline := buildFixture(t, hostExposition)
	if len(baseline.Optional) != 0 {
		t.Fatalf("baseline unexpectedly has optional panels: %+v", baseline.Optional)
	}
	view := renderFixture(t, hostExposition)
	for _, title := range []string{"Package temperature (100 ms)", "DRAM power", "Per-CPU telemetry",
		"Device interrupts (counters)", "NIC driver enp1s0"} {
		if strings.Contains(view, title) {
			t.Errorf("baseline contains opt-in panel %q", title)
		}
	}

	view = renderFixture(t, hostExposition+optionalExposition)
	for _, want := range []string{
		"Package temperature (100 ms)", "54.00 °C", "DRAM power", "1.50 W",
		"Per-CPU telemetry", "CPU 0 frequency", "2000.00 MHz",
		"Device interrupts (counters)", "IRQ 42 nvme0q1", "1234.00",
		"NIC driver enp1s0", "rx_errors",
	} {
		wantContains(t, view, want)
	}
	if strings.Contains(view, "99.00 °C") {
		t.Error("non-package sensor leaked into optional temperature panel")
	}
}
