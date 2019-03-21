package monitors

import (
	"github.com/pkg/errors"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/utils/filter"
)

// If it's an included metric, continue to exclude processing.
// If metric is enabled via additionalMetrics, continue to exclude processing.
// If we don't know of the metric at all, continue to exclude processing.
// <drop>

// Exclude processing:
// If metric matches exclude rule <drop>
// <send>

type additionalMetricsFilter struct {
	metadata *Metadata
	filter   *filter.BasicStringFilter
}

// Used for filtering datapoints based on included status and user configuration.

func newAdditionalMetricsFilter(
	metadata *Metadata,
	additionalMetrics []config.AdditionalMetric) (*additionalMetricsFilter, error) {
	var items []string

	for _, add := range additionalMetrics {
		if add.MetricName == "" && add.Group == "" {
			return nil, errors.New("either metricName or group must be set")
		} else if add.MetricName != "" && add.Group != "" {
			return nil, errors.New("both metricName and group cannot be set")
		} else {
			// Good

			if add.MetricName != "" {
				// TODO: fix verify check if it's star
				if !metadata.Metrics[add.MetricName] {
					return nil, errors.Errorf("metric %s does not exist", add.MetricName)
				}
				items = append(items, add.MetricName)
			}

			if add.Group != "" {
				if metrics, ok := metadata.GroupMetricsMap[add.Group]; !ok {
					return nil, errors.Errorf("group %s does not exist", add.Group)
				} else {
					items = append(items, metrics...)
				}
			}
		}
	}

	basicFilter, err := filter.NewBasicStringFilter(items)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to construct filter with items %s", items)
	}

	return &additionalMetricsFilter{metadata, basicFilter}, nil
}

func (add *additionalMetricsFilter) shouldSend(dp *datapoint.Datapoint) bool {
	// TODO: check metric type (counter, etc.) against docs

	if add.metadata.SendAll {
		return true
	}

	if _, ok := add.metadata.IncludedMetrics[dp.Metric]; ok {
		// It's an included metric so send by default.
		return true
	}

	if !add.metadata.Metrics[dp.Metric] {
		// TODO: log/increase counter?
		// We don't know about this metric at all so send it to be safe. It might not have been documented
		// or it might be a dynamic metric name.
		return true
	}

	// User has explicitly chosen to send this metric (or a group that this metric belongs to).
	if add.filter.Matches(dp.Metric) {
		return true
	}

	// It's a known (but not included) metric and the user has not enabled it, so do not send it.
	return false
}
