package monitors

import (
	"fmt"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"testing"
)

var metadata = &Metadata{
	"test-monitor",
	false,
	utils.MakeSet("cpu.idle", "cpu.min", "cpu.max", "mem.used"),
	utils.MakeSet("cpu.idle", "cpu.min", "cpu.max", "mem.used", "mem.free", "mem.available"),
	utils.MakeSet("cpu", "mem"),
	map[string][]string{
		"cpu": {"cpu.idle", "cpu.min", "cpu.max"},
		"mem": {"mem.used", "mem.free", "mem.available"},
	},
}

func TestAdditionalMetricsFilter(t *testing.T) {
	// test metricName, groups, negation, sendall

	// TODO: Test negation of metricnames.

	// Test included.
	if filter, err := newAdditionalMetricsFilter(metadata, nil); err != nil {
		t.Error(err)
	} else {
		// All included metrics should be sent.
		for metric, _ := range metadata.IncludedMetrics {
			t.Run(fmt.Sprintf("included metric %s should send", metric), func(t *testing.T) {
				dp := &datapoint.Datapoint{
					Metric:     metric,
					MetricType: datapoint.Counter,
				}
				if !filter.shouldSend(dp) {
					t.Error()
				}
			})
		}
	}

	// Test additional metric names
	if filter, err := newAdditionalMetricsFilter(metadata, []config.AdditionalMetric{
		{MetricName: "mem.used"}}); err != nil {
		t.Error(err)
	} else {
		for metric, shouldSend := range map[string]bool{
			"mem.used":      true,
			"mem.free":      false,
			"mem.available": false,
		} {
			dp := &datapoint.Datapoint{Metric: metric, MetricType: datapoint.Counter}
			sent := filter.shouldSend(dp)
			if sent && !shouldSend {
				t.Errorf("metric %s should not have sent", metric)
			}
			if !sent && shouldSend {
				t.Errorf("metric %s should have been sent", metric)
			}

		}

	}

	// Test groups
	if filter, err := newAdditionalMetricsFilter(metadata, []config.AdditionalMetric{
		{Group: "mem"}}); err != nil {
		t.Error(err)
	} else {
		for _, metric := range metadata.GroupMetricsMap["mem"] {
			dp := &datapoint.Datapoint{Metric: metric, MetricType: datapoint.Counter}

			if !filter.shouldSend(dp) {
				t.Errorf("metric %s should have been sent", metric)
			}
		}
	}
}
