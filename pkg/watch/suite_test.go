package watch

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/zoetrope/kubbernecker/pkg/client"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var cfg *rest.Config
var kubeClient *client.KubeClient
var testEnv *envtest.Environment
var scheme = runtime.NewScheme()
var cancelCluster context.CancelFunc

func TestWatch(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Watcher Suite", Label("envtest", "watcher"))
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true), zap.Level(zapcore.Level(-10))))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())
	err = clientgoscheme.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	kubeClient, err = client.MakeKubeClientFromRestConfig(cfg, "default")
	Expect(err).NotTo(HaveOccurred())
	Expect(kubeClient).NotTo(BeNil())

	var ctx context.Context
	ctx, cancelCluster = context.WithCancel(context.Background())
	go kubeClient.Cluster.Start(ctx)

	// wait for creating default namespace
	Eventually(func(g Gomega) {
		ns := &corev1.Namespace{}
		err = kubeClient.Cluster.GetClient().Get(context.Background(), ctrlclient.ObjectKey{Name: "default"}, ns)
		g.Expect(err).ShouldNot(HaveOccurred())
	}).Should(Succeed())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancelCluster()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
