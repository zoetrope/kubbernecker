package watch

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("Test SampleController", func() {
	ctx := context.Background()
	var stopFunc func()
	logger := ctrl.Log.WithName("watcher-test")
	var watcher *Watcher

	BeforeEach(func() {
		gvk, err := kubeClient.DetectGVK("configmaps")
		Expect(err).NotTo(HaveOccurred())
		logger.Info("gvk", "gvk", *gvk)
		watcher = NewWatcher(&logger, kubeClient, *gvk)

		ctx, cancel := context.WithCancel(ctx)
		stopFunc = cancel
		go func() {
			err := watcher.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
	})

	AfterEach(func() {
		stopFunc()
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

		err := kubeClient.Client.Create(ctx, cm)
		Expect(err).NotTo(HaveOccurred())

		time.Sleep(1 * time.Second)
		statistics := watcher.PrintStatistics()
		Expect(statistics).Should(Equal("hoge"))
	})
})
