package client

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("Test KubeClient", func() {
	It("should detect GroupVersionKind from resources arguments", func() {
		gvk, err := kubeClient.DetectGVK("pods")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(gvk).Should(PointTo(MatchAllFields(Fields{
			"Kind":    Equal("Pod"),
			"Version": Equal("v1"),
			"Group":   Equal(""),
		})))

		gvk, err = kubeClient.DetectGVK("horizontalpodautoscalers")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(gvk).Should(PointTo(MatchAllFields(Fields{
			"Kind":    Equal("HorizontalPodAutoscaler"),
			"Version": Equal("v2"),
			"Group":   Equal("autoscaling"),
		})))

		gvk, err = kubeClient.DetectGVK("horizontalpodautoscalers.autoscaling")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(gvk).Should(PointTo(MatchAllFields(Fields{
			"Kind":    Equal("HorizontalPodAutoscaler"),
			"Version": Equal("v2"),
			"Group":   Equal("autoscaling"),
		})))

		gvk, err = kubeClient.DetectGVK("horizontalpodautoscalers.v1.autoscaling")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(gvk).Should(PointTo(MatchAllFields(Fields{
			"Kind":    Equal("HorizontalPodAutoscaler"),
			"Version": Equal("v1"),
			"Group":   Equal("autoscaling"),
		})))

		gvk, err = kubeClient.DetectGVK("mutatingwebhookconfigurations.v1.admissionregistration.k8s.io")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(gvk).Should(PointTo(MatchAllFields(Fields{
			"Kind":    Equal("MutatingWebhookConfiguration"),
			"Version": Equal("v1"),
			"Group":   Equal("admissionregistration.k8s.io"),
		})))
	})
})
