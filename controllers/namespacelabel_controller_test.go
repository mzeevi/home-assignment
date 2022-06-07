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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	danaiov1alpha1 "home-assignment/apis/namespacelabel/v1alpha1"
)

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
