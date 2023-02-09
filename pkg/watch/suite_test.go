package watch

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	SetDefaultEventuallyTimeout(5 * time.Second)
	SetDefaultEventuallyPollingInterval(1 * time.Second)
	SetDefaultConsistentlyDuration(5 * time.Second)
	SetDefaultConsistentlyPollingInterval(1 * time.Second)

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

	kubeClient, err = client.MakeKubeClientFromRestConfig(cfg, "")
	Expect(err).NotTo(HaveOccurred())
	Expect(kubeClient).NotTo(BeNil())

	var ctx context.Context
	ctx, cancelCluster = context.WithCancel(context.Background())
	go kubeClient.Cluster.Start(ctx)

	cli := kubeClient.Cluster.GetClient()
	// wait for creating default namespace
	Eventually(func(g Gomega) {
		ns := &corev1.Namespace{}
		err = cli.Get(ctx, ctrlclient.ObjectKey{Name: "default"}, ns)
		g.Expect(err).ShouldNot(HaveOccurred())
	}).Should(Succeed())

	ns1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "admin-ns",
			Labels: map[string]string{
				"role": "admin",
			},
		},
	}
	err = cli.Create(ctx, ns1)
	Expect(err).ShouldNot(HaveOccurred())

	ns2 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "user-ns",
			Labels: map[string]string{
				"role": "user",
			},
		},
	}
	err = cli.Create(ctx, ns2)
	Expect(err).ShouldNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancelCluster()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
