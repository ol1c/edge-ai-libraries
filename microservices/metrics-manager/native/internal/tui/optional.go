// Copyright (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/open-edge-platform/edge-ai-libraries/microservices/metrics-manager/native/internal/promtext"
)

// OptionalPanel is built only when an opt-in collector actually publishes data.
type OptionalPanel struct {
	Title        string
	Rows         []OptionalRow
	TableColumns []column
	TableRows    [][]string
}

type OptionalRow struct {
	Label, Value string
}

func buildOptionalPanels(idx *Index) []OptionalPanel {
	var panels []OptionalPanel
	appendPanel := func(panel OptionalPanel) {
		if len(panel.Rows) > 0 || len(panel.TableRows) > 0 {
			panels = append(panels, panel)
		}
	}

	temperature := OptionalPanel{Title: "Package temperature (100 ms)"}
	for _, metric := range []struct{ name, label string }{
		{"temp_highres_temp", "sample"},
		{"temp_highres_temp_min", "min"},
		{"temp_highres_temp_max", "max"},
		{"temp_highres_temp_mean", "mean"},
	} {
		for _, sample := range idx.All(metric.name) {
			if isPackageSensor(sample.Labels["sensor"]) {
				addOptionalSample(&temperature, sample.Labels["sensor"]+" "+metric.label, sample, " °C")
			}
		}
	}
	appendPanel(temperature)

	power := OptionalPanel{Title: "DRAM power"}
	for _, sample := range idx.All("powerstat_package_current_dram_power_consumption_watts") {
		addOptionalSample(&power, "package "+sample.Labels["package_id"], sample, " W")
	}
	appendPanel(power)

	appendPanel(buildPerCPUPanel(idx))

	interrupts := OptionalPanel{Title: "Device interrupts (counters)"}
	for _, sample := range idx.All("interrupts_total") {
		irq := sample.Labels["irq"]
		if irq == "" {
			continue
		}
		label := "IRQ " + irq
		if device := sample.Labels["device"]; device != "" {
			label += " " + device
		}
		addOptionalSample(&interrupts, label, sample, "")
	}
	appendPanel(interrupts)

	interfaces := make(map[string]*OptionalPanel)
	for _, name := range idx.NamesWithPrefix("ethtool_") {
		for _, sample := range idx.All(name) {
			iface := sample.Labels["interface"]
			if iface == "" {
				iface = sample.Labels["name"]
			}
			if iface == "" {
				continue
			}
			key := sample.Labels["host"] + "/" + iface
			panel := interfaces[key]
			if panel == nil {
				panel = &OptionalPanel{Title: "NIC driver " + iface}
				interfaces[key] = panel
			}
			addOptionalSample(panel, strings.TrimPrefix(name, "ethtool_"), sample, "")
		}
	}
	keys := make([]string, 0, len(interfaces))
	for key := range interfaces {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		appendPanel(*interfaces[key])
	}

	turbostat := OptionalPanel{Title: "Turbostat"}
	for _, metric := range []struct{ name, label, unit string }{
		{"turbostat_ipc", "IPC", ""},
		{"turbostat_smi", "SMI", ""},
		{"turbostat_irq", "IRQ", ""},
		{"turbostat_nmi", "NMI", ""},
		{"turbostat_busy_percent", "busy", " %"},
		{"turbostat_busy_frequency_mhz", "busy frequency", " MHz"},
		{"turbostat_core_temperature_celsius", "core temperature", " °C"},
	} {
		for _, sample := range idx.All(metric.name) {
			place := "system"
			if cpu := sample.Labels["cpu"]; cpu != "" && cpu != "-" {
				place = "CPU " + cpu
			} else if core := sample.Labels["core"]; core != "" && core != "-" {
				place = "core " + core
			}
			addOptionalSample(&turbostat, metric.label+" "+place, sample, metric.unit)
		}
	}
	appendPanel(turbostat)

	return panels
}

func buildPerCPUPanel(idx *Index) OptionalPanel {
	panel := OptionalPanel{
		Title: "Per-CPU telemetry",
		TableColumns: []column{
			{title: "CPU", width: 10, left: true},
			{title: "frequency", width: 18},
			{title: "temperature", width: 16},
		},
	}
	type readings struct {
		frequency, temperature Reading
	}
	cpus := make(map[string]*readings)
	for _, metric := range []struct {
		name        string
		temperature bool
	}{
		{"powerstat_core_cpu_frequency_mhz", false},
		{"powerstat_core_cpu_temperature_celsius", true},
	} {
		for _, sample := range idx.All(metric.name) {
			id := sample.Labels["cpu_id"]
			if id == "" || math.IsNaN(sample.Value) || math.IsInf(sample.Value, 0) {
				continue
			}
			cpu := cpus[id]
			if cpu == nil {
				cpu = &readings{}
				cpus[id] = cpu
			}
			if metric.temperature {
				cpu.temperature = reading(sample.Value)
			} else {
				cpu.frequency = reading(sample.Value)
			}
		}
	}

	ids := make([]string, 0, len(cpus))
	for id := range cpus {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, aErr := strconv.Atoi(ids[i])
		b, bErr := strconv.Atoi(ids[j])
		if aErr == nil && bErr == nil && a != b {
			return a < b
		}
		return ids[i] < ids[j]
	})
	for _, id := range ids {
		cpu := cpus[id]
		frequency, temperature := "-", "-"
		if cpu.frequency.OK {
			frequency = fmt.Sprintf("%.2f MHz", cpu.frequency.Value)
		}
		if cpu.temperature.OK {
			temperature = fmt.Sprintf("%.2f °C", cpu.temperature.Value)
		}
		panel.TableRows = append(panel.TableRows, []string{"CPU " + sanitize(id, 20), frequency, temperature})
	}
	return panel
}

func addOptionalSample(panel *OptionalPanel, label string, sample promtext.Sample, unit string) {
	if math.IsNaN(sample.Value) || math.IsInf(sample.Value, 0) {
		return
	}
	panel.Rows = append(panel.Rows, OptionalRow{
		Label: label,
		Value: fmt.Sprintf("%.2f%s", sample.Value, unit),
	})
}

func (m Model) optionalSection(panel OptionalPanel) string {
	var b strings.Builder
	b.WriteString(headingStyle.Render(sanitize(panel.Title, 100)) + "\n")
	if len(panel.TableRows) > 0 {
		b.WriteString(responsiveTable(panel.TableColumns, panel.TableRows, m.panelWidth()))
	}
	for _, row := range panel.Rows {
		b.WriteString(field(sanitize(row.Label, 60), row.Value))
	}
	return b.String()
}
