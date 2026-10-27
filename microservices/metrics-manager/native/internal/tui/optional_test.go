// Copyright (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"regexp"
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
		"Per-CPU telemetry", "frequency", "temperature", "CPU 0", "2000.00 MHz", "52.00 °C",
		"Device interrupts (counters)", "IRQ 42 nvme0q1", "1234.00",
		"NIC driver enp1s0", "rx_errors",
	} {
		wantContains(t, view, want)
	}
	if strings.Contains(view, "99.00 °C") {
		t.Error("non-package sensor leaked into optional temperature panel")
	}
}

func TestPerCPUTelemetryRowsKeepMissingValuesInTheirColumns(t *testing.T) {
	view := renderFixture(t, `powerstat_core_cpu_frequency_mhz{cpu_id="10"} 2000
powerstat_core_cpu_temperature_celsius{cpu_id="8"} 42
powerstat_core_cpu_frequency_mhz{cpu_id="1"} 0
powerstat_core_cpu_temperature_celsius{cpu_id="1"} 51
`)

	patterns := []string{
		`CPU 1\s+0\.00 MHz\s+51\.00 °C`,
		`CPU 8\s+-\s+42\.00 °C`,
		`CPU 10\s+2000\.00 MHz\s+-`,
	}
	previous := -1
	for _, pattern := range patterns {
		location := regexp.MustCompile(pattern).FindStringIndex(view)
		if location == nil {
			t.Errorf("missing CPU row matching %q", pattern)
			continue
		}
		if location[0] <= previous {
			t.Errorf("CPU rows not sorted numerically: %q", pattern)
		}
		previous = location[0]
	}
}

func TestTurbostatPanelRequiresSamples(t *testing.T) {
	baseline := renderFixture(t, hostExposition)
	if strings.Contains(baseline, "Turbostat") {
		t.Fatal("turbostat panel is visible without samples")
	}

	view := renderFixture(t, hostExposition+`turbostat_ipc{host="itest",core="0",cpu="8"} 1.24
turbostat_smi{host="itest",core="-",cpu="-"} 3
`)
	wantContains(t, view, "Turbostat")
	wantContains(t, view, "IPC CPU 8")
	wantContains(t, view, "SMI system")
}

func TestNPUPowerStateInPanel(t *testing.T) {
	baseline := renderFixture(t, acceleratorExposition)
	if strings.Contains(baseline, "power state") {
		t.Fatal("NPU power state is visible before a collector publishes it")
	}
	view := renderFixture(t, acceleratorExposition+"npu_power_state{host=\"itest\"} 3\n")
	wantContains(t, view, "power state")
	wantContains(t, view, "D3hot")
	view = renderFixture(t, acceleratorExposition+"npu_power_state{host=\"itest\"} -1\n")
	wantContains(t, view, "unknown")
}
