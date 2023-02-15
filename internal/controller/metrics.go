package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	resourceEventsCountDesc = prometheus.NewDesc(
		"kubbernecker_resource_events_total",
		"Total number of Kubernetes events by resource type.",
		[]string{"resource_type", "namespace", "event_type"}, nil)
	resourceUpdateCountDesc = prometheus.NewDesc(
		"kubbernecker_resource_updates_total",
		"Total number of updates for a Kubernetes resource instance.",
		[]string{"resource_type", "namespace", "resource_name"}, nil)
)

func (m *WatcherManager) Describe(ch chan<- *prometheus.Desc) {
	ch <- resourceEventsCountDesc
	ch <- resourceUpdateCountDesc
}

func resourceType(gvk metav1.GroupVersionKind) string {
	if gvk.Group == "" {
		return "core." + gvk.Version + "." + gvk.Kind
	}

	return gvk.Group + "." + gvk.Version + "." + gvk.Kind
}

func (m *WatcherManager) Collect(ch chan<- prometheus.Metric) {
	for _, watcher := range m.watchers {
		statistics := watcher.Statistics()
		for ns, nsStatistics := range statistics.Namespaces {
			ch <- prometheus.MustNewConstMetric(
				resourceEventsCountDesc,
				prometheus.CounterValue,
				float64(nsStatistics.AddCount),
				resourceType(statistics.GroupVersionKind), ns, "add",
			)
			ch <- prometheus.MustNewConstMetric(
				resourceEventsCountDesc,
				prometheus.CounterValue,
				float64(nsStatistics.UpdateCount),
				resourceType(statistics.GroupVersionKind), ns, "update",
			)
			ch <- prometheus.MustNewConstMetric(
				resourceEventsCountDesc,
				prometheus.CounterValue,
				float64(nsStatistics.DeleteCount),
				resourceType(statistics.GroupVersionKind), ns, "delete",
			)
			for res, resStatistics := range nsStatistics.Resources {
				ch <- prometheus.MustNewConstMetric(
					resourceUpdateCountDesc,
					prometheus.CounterValue,
					float64(resStatistics.UpdateCount),
					resourceType(statistics.GroupVersionKind), ns, res,
				)
			}
		}
	}
}

func SetupMetrics(m *WatcherManager) error {
	return metrics.Registry.Register(m)
}
