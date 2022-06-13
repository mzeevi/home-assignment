/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	danaiov1alpha1 "home-assignment/apis/namespacelabel/v1alpha1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

// var cfg *rest.Config
var (
	cfg       *rest.Config
	k8sClient client.Client // You'll be using this client in your tests.
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = danaiov1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())

	err = (&NamespaceLabelReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme()}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

}, 60)

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Namespacelabel Controller", func() {

	// define utility constants for object names and testing timeouts/durations and intervals
	const (
		NamespaceLabelName      = "test-namespacelabel"
		NamespaceLabelNamespace = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When updating NamespaceLabel Status", func() {
		It("Should change NamespaceLabel Status.ActiveLabels to match new namespace labels", func() {
			By("By creating a new NamespaceLabel")
			ctx := context.Background()
			namespaceLabel := danaiov1alpha1.NamespaceLabel{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "dana.io.dana.io/v1alpha1",
					Kind:       "NamespaceLabel",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      NamespaceLabelName,
					Namespace: NamespaceLabelNamespace,
				},
				Spec: danaiov1alpha1.NamespaceLabelSpec{
					Labels: map[string]string{
						"labelA": "testlabel",
					},
				},
			}
			Expect(k8sClient.Create(ctx, &namespaceLabel)).Should(Succeed())

			namespaceLabelLookupKey := types.NamespacedName{
				Name:      NamespaceLabelName,
				Namespace: NamespaceLabelNamespace,
			}
			createdNamespaceLabel := danaiov1alpha1.NamespaceLabel{}

			// we'll need to retry getting this newly created namespaceLabel, given that creation may not immediately happen
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespaceLabelLookupKey, &createdNamespaceLabel)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// let's make sure our labels map value was properly converted/handled.
			expectedLabels := map[string]string{
				"labelA": "testlabel",
			}
			Expect(createdNamespaceLabel.Spec.Labels).Should(Equal(expectedLabels))

			By("By checking that the namespace has the new labels")
			Eventually(func() (map[string]string, error) {
				err := k8sClient.Get(ctx, namespaceLabelLookupKey, &createdNamespaceLabel)
				if err != nil {
					return nil, err
				}

				labels := map[string]string{}
				for key, val := range createdNamespaceLabel.Status.ActiveLabels {
					labels[key] = val
				}
				return labels, nil
			}, timeout, interval).Should(ContainElements("testlabel"))
		})
	})
})
