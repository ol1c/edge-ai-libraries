// Copyright (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/open-edge-platform/edge-ai-libraries/microservices/metrics-manager/native/internal/promtext"
)

// OptionalPanel is built only when an opt-in collector actually publishes data.
type OptionalPanel struct {
	Title string
	Rows  []OptionalRow
}

type OptionalRow struct {
	Label, Value string
}

func buildOptionalPanels(idx *Index) []OptionalPanel {
	var panels []OptionalPanel
	appendPanel := func(panel OptionalPanel) {
		if len(panel.Rows) > 0 {
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

	cores := OptionalPanel{Title: "Per-CPU telemetry"}
	for _, metric := range []struct{ name, label, unit string }{
		{"powerstat_core_cpu_frequency_mhz", "frequency", " MHz"},
		{"powerstat_core_cpu_temperature_celsius", "temperature", " °C"},
	} {
		for _, sample := range idx.All(metric.name) {
			addOptionalSample(&cores, "CPU "+sample.Labels["cpu_id"]+" "+metric.label, sample, metric.unit)
		}
	}
	appendPanel(cores)

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

	return panels
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

func optionalSection(panel OptionalPanel) string {
	var b strings.Builder
	b.WriteString(headingStyle.Render(sanitize(panel.Title, 100)) + "\n")
	for _, row := range panel.Rows {
		b.WriteString(field(sanitize(row.Label, 60), row.Value))
	}
	return b.String()
}
