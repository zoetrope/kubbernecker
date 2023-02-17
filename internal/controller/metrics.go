package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	resourceEventsCountDesc = prometheus.NewDesc(
		"kubbernecker_resource_events_total",
		"Total number of events for Kubernetes resources",
		[]string{"group", "version", "kind", "namespace", "event_type", "resource_name"}, nil)
)

func (m *WatcherManager) Describe(ch chan<- *prometheus.Desc) {
	ch <- resourceEventsCountDesc
}

func (m *WatcherManager) Collect(ch chan<- prometheus.Metric) {
	for _, watcher := range m.watchers {
		statistics := watcher.Statistics()
		for ns, nsStatistics := range statistics.Namespaces {
			for res, resStatistics := range nsStatistics.Resources {
				ch <- prometheus.MustNewConstMetric(
					resourceEventsCountDesc,
					prometheus.CounterValue,
					float64(resStatistics.UpdateCount),
					statistics.GroupVersionKind.Group, statistics.GroupVersionKind.Version, statistics.GroupVersionKind.Kind, ns, "update", res,
				)
				ch <- prometheus.MustNewConstMetric(
					resourceEventsCountDesc,
					prometheus.CounterValue,
					float64(resStatistics.AddCount),
					statistics.GroupVersionKind.Group, statistics.GroupVersionKind.Version, statistics.GroupVersionKind.Kind, ns, "add", res,
				)
				ch <- prometheus.MustNewConstMetric(
					resourceEventsCountDesc,
					prometheus.CounterValue,
					float64(resStatistics.DeleteCount),
					statistics.GroupVersionKind.Group, statistics.GroupVersionKind.Version, statistics.GroupVersionKind.Kind, ns, "delete", res,
				)
			}
		}
	}
}

func SetupMetrics(m *WatcherManager) error {
	return metrics.Registry.Register(m)
}
