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
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Test Watcher", func() {
	ctx := context.Background()
	logger := ctrl.Log.WithName("watcher-test")
	var watcher *Watcher

	var startWatcher = func(resourceType string, nsSelector, resSelector labels.Selector) {
		gvk, err := kubeClient.DetectGVK(resourceType)
		Expect(err).NotTo(HaveOccurred())
		watcher = NewWatcher(logger, kubeClient, *gvk, nsSelector, resSelector)

		err = watcher.Start(ctx)
		Expect(err).NotTo(HaveOccurred())

		time.Sleep(1 * time.Second)
	}

	AfterEach(func() {
		cli := kubeClient.Cluster.GetClient()
		err := cli.DeleteAllOf(ctx, &corev1.ConfigMap{}, ctrlclient.InNamespace("default"))
		Expect(err).ShouldNot(HaveOccurred())

		err = cli.DeleteAllOf(ctx, &corev1.ConfigMap{}, ctrlclient.InNamespace("admin-ns"))
		Expect(err).ShouldNot(HaveOccurred())

		err = cli.DeleteAllOf(ctx, &corev1.ConfigMap{}, ctrlclient.InNamespace("user-ns"))
		Expect(err).ShouldNot(HaveOccurred())

		err = watcher.Stop()
		Expect(err).ShouldNot(HaveOccurred())
	})

	Context("Watcher with everything", func() {
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

			Consistently(func(g Gomega) {
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

	Context("Watcher with namespace selector", func() {
		BeforeEach(func() {
			startWatcher("configmaps", labels.SelectorFromSet(map[string]string{"role": "admin"}), labels.Everything())
		})

		It("should only count configmap in admin-ns", func() {
			cm1 := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "admin-ns",
					Name:      "test1",
				},
				Data: map[string]string{
					"sample": "data",
				},
			}
			err := kubeClient.Cluster.GetClient().Create(ctx, cm1)
			Expect(err).NotTo(HaveOccurred())

			cm2 := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "user-ns",
					Name:      "test2",
				},
				Data: map[string]string{
					"sample": "data",
				},
			}
			err = kubeClient.Cluster.GetClient().Create(ctx, cm2)
			Expect(err).NotTo(HaveOccurred())

			Consistently(func(g Gomega) {
				statistics := watcher.Statistics()
				g.Expect(statistics.Namespaces).Should(MatchAllKeys(Keys{
					"admin-ns": PointTo(MatchAllFields(Fields{
						"Resources": MatchAllKeys(Keys{
							"test1": PointTo(MatchAllFields(Fields{
								"UpdateCount": Equal(0),
							})),
						}),
						"AddCount":    Equal(1),
						"UpdateCount": Equal(0),
						"DeleteCount": Equal(0),
					})),
					// user-ns should not appear
				}))
			}).Should(Succeed())
		})
	})

	Context("Watcher with resource selector", func() {
		BeforeEach(func() {
			selector, err := labels.Parse("ignored!=true")
			Expect(err).NotTo(HaveOccurred())
			startWatcher("configmaps", labels.Everything(), selector)
		})

		It("should only count configmap in admin-ns", func() {
			cm1 := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "admin-ns",
					Name:      "test1",
					Labels: map[string]string{
						"ignored": "true",
					},
				},
				Data: map[string]string{
					"sample": "data",
				},
			}
			err := kubeClient.Cluster.GetClient().Create(ctx, cm1)
			Expect(err).NotTo(HaveOccurred())

			cm2 := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "user-ns",
					Name:      "test2",
				},
				Data: map[string]string{
					"sample": "data",
				},
			}
			err = kubeClient.Cluster.GetClient().Create(ctx, cm2)
			Expect(err).NotTo(HaveOccurred())

			Consistently(func(g Gomega) {
				statistics := watcher.Statistics()
				g.Expect(statistics.Namespaces).Should(MatchAllKeys(Keys{
					"user-ns": PointTo(MatchAllFields(Fields{
						"Resources": MatchAllKeys(Keys{
							"test2": PointTo(MatchAllFields(Fields{
								"UpdateCount": Equal(0),
							})),
						}),
						"AddCount":    Equal(1),
						"UpdateCount": Equal(0),
						"DeleteCount": Equal(0),
					})),
					// admin-ns should not appear
				}))
			}).Should(Succeed())
		})
	})
})
