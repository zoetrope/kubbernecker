package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	resourceUpdateCountDesc = prometheus.NewDesc(
		"kubbernecker_resource_updates_total",
		"",
		[]string{"group", "version", "kind", "namespace", "resource"}, nil)
	kindUpdateCountDesc = prometheus.NewDesc(
		"kubbernecker_kind_updates_total",
		"",
		[]string{"group", "version", "kind", "namespace", "type"}, nil)
)

func (m *WatcherManager) Describe(ch chan<- *prometheus.Desc) {
	ch <- resourceUpdateCountDesc
	ch <- kindUpdateCountDesc
}

func (m *WatcherManager) Collect(ch chan<- prometheus.Metric) {
	for _, watcher := range m.watchers {
		statistics := watcher.Statistics()
		for ns, nsStatistics := range statistics.Namespaces {
			ch <- prometheus.MustNewConstMetric(
				kindUpdateCountDesc,
				prometheus.CounterValue,
				float64(nsStatistics.AddCount),
				statistics.GroupVersionKind.Group, statistics.GroupVersionKind.Version, statistics.GroupVersionKind.Kind, ns, "add",
			)
			ch <- prometheus.MustNewConstMetric(
				kindUpdateCountDesc,
				prometheus.CounterValue,
				float64(nsStatistics.UpdateCount),
				statistics.GroupVersionKind.Group, statistics.GroupVersionKind.Version, statistics.GroupVersionKind.Kind, ns, "update",
			)
			ch <- prometheus.MustNewConstMetric(
				kindUpdateCountDesc,
				prometheus.CounterValue,
				float64(nsStatistics.DeleteCount),
				statistics.GroupVersionKind.Group, statistics.GroupVersionKind.Version, statistics.GroupVersionKind.Kind, ns, "delete",
			)
			for res, resStatistics := range nsStatistics.Resources {
				ch <- prometheus.MustNewConstMetric(
					resourceUpdateCountDesc,
					prometheus.CounterValue,
					float64(resStatistics.UpdateCount),
					statistics.GroupVersionKind.Group, statistics.GroupVersionKind.Version, statistics.GroupVersionKind.Kind, ns, res, "update",
				)
			}
		}
	}
}

func SetupMetrics(m *WatcherManager) error {
	return metrics.Registry.Register(m)
}
