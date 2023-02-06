package watch

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("Test Watcher", func() {
	ctx := context.Background()
	var stopFunc func()
	logger := ctrl.Log.WithName("watcher-test")
	var watcher *Watcher

	var startWatcher = func(resourceType string, nsSelector, resSelector labels.Selector) {
		gvk, err := kubeClient.DetectGVK(resourceType)
		Expect(err).NotTo(HaveOccurred())
		watcher = NewWatcher(logger, kubeClient, *gvk, nsSelector, resSelector)

		ctx, cancel := context.WithCancel(ctx)
		stopFunc = cancel
		go func() {
			err := watcher.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
		time.Sleep(1 * time.Second)
	}

	AfterEach(func() {
		stopFunc()
	})

	Context("ConfigMap Watcher", func() {
		BeforeEach(func() {
			startWatcher("configmaps", labels.Everything(), labels.Everything())
		})

		It("should be success", func() {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test",
				},
				Data: map[string]string{
					"sample": "data",
				},
			}
			err := kubeClient.Cluster.GetClient().Create(ctx, cm)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func(g Gomega) {
				statistics := watcher.Statistics()
				g.Expect(statistics.Namespaces).Should(MatchAllKeys(Keys{
					"default": PointTo(MatchAllFields(Fields{
						"Resources": MatchAllKeys(Keys{
							"test": PointTo(MatchAllFields(Fields{
								"UpdateCount": Equal(0),
							})),
						}),
						"AddCount":    Equal(1),
						"UpdateCount": Equal(0),
						"DeleteCount": Equal(0),
					})),
				}))
			}).Should(Succeed())
		})
	})
})
