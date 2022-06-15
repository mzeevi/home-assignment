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
	"reflect"
	"testing"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	danaiov1alpha1 "home-assignment/apis/namespacelabel/v1alpha1"
)

const (
	LabelKey = "label-key"
	LabelVal = "label-value"
)

func setupClient(obj []client.Object) (client.Client, *runtime.Scheme, error) {

	s := scheme.Scheme
	if err := danaiov1alpha1.AddToScheme(s); err != nil {
		return nil, s, err
	}

	// create fake client
	cl := fake.NewClientBuilder().WithObjects(obj...).Build()

	return cl, s, nil

}

func generateNamespacelabelObject() *danaiov1alpha1.NamespaceLabel {
	namespaceLabel := &danaiov1alpha1.NamespaceLabel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "namespacelabel-test",
			Namespace: "default",
		},
		Spec: danaiov1alpha1.NamespaceLabelSpec{
			Labels: map[string]string{
				LabelKey: LabelVal,
			},
		},
		Status: danaiov1alpha1.NamespaceLabelStatus{
			ActiveLabels: map[string]string{
				LabelKey: LabelVal,
			},
		},
	}

	return namespaceLabel
}

func generateNamespaceObject() *v1.Namespace {
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "default",
			Labels: map[string]string{
				"kubernetes.io/name": "default",
				LabelKey:             LabelVal,
			},
		},
	}

	return namespace
}

func TestDeleteLabels(t *testing.T) {
	g := NewGomegaWithT(t)
	RegisterFailHandler(ginkgo.Fail)

	namespaceLabel := generateNamespacelabelObject()
	namespace := generateNamespaceObject()

	// objects to track in the fake client
	obj := []client.Object{namespaceLabel, namespace}

	cl, s, err := setupClient(obj)
	if err != nil {
		t.Fatalf("Unable to add to scheme: %v", err)
	}

	// create a NamespaceLabelReconciler object with the scheme and fake client
	r := &NamespaceLabelReconciler{cl, s}

	// run function to test
	r.deleteLabels(namespaceLabel, namespace)

	// set expected result and check result matches expected
	g.Expect(func() bool {
		expectedLabels := map[string]string{
			"kubernetes.io/name": "default",
		}
		actualLabels := namespace.ObjectMeta.Labels
		return reflect.DeepEqual(expectedLabels, actualLabels)
	}()).To(BeTrue())
}

func TestDeleteFinalizer(t *testing.T) {
	g := NewGomegaWithT(t)
	RegisterFailHandler(ginkgo.Fail)

	namespaceLabel := generateNamespacelabelObject()
	namespace := generateNamespaceObject()

	obj := []client.Object{namespaceLabel, namespace}
	cl, s, err := setupClient(obj)
	if err != nil {
		t.Fatalf("Unable to add to scheme: %v", err)
	}

	// create a NamespaceLabelReconciler object with the scheme and fake client
	r := &NamespaceLabelReconciler{cl, s}

	// add finalizer to namespacelabel object
	controllerutil.AddFinalizer(namespaceLabel, NamespaceLabelFinalizer)

	// run function to test
	r.deleteFinalizer(context.TODO(), namespaceLabel, namespace)

	// set expected result and check result matches expected
	g.Expect(func() bool {
		return !controllerutil.ContainsFinalizer(namespaceLabel, NamespaceLabelFinalizer)
	}()).To(BeTrue())
}

func TestAddFinalizer(t *testing.T) {
	g := NewGomegaWithT(t)
	RegisterFailHandler(ginkgo.Fail)

	namespaceLabel := generateNamespacelabelObject()

	obj := []client.Object{namespaceLabel}
	cl, s, err := setupClient(obj)
	if err != nil {
		t.Fatalf("Unable to add to scheme: %v", err)
	}

	// create a NamespaceLabelReconciler object with the scheme and fake client
	r := &NamespaceLabelReconciler{cl, s}

	// run function to test
	r.addFinalizer(context.TODO(), namespaceLabel)

	// set expected result and check result matches expected
	g.Expect(func() bool {
		return controllerutil.ContainsFinalizer(namespaceLabel, NamespaceLabelFinalizer)
	}()).To(BeTrue())
}

func TestGetNamespaceLabelsDiffs(t *testing.T) {
	g := NewGomegaWithT(t)
	RegisterFailHandler(ginkgo.Fail)

	namespaceLabel := generateNamespacelabelObject()

	newSpecLabels := map[string]string{
		"labelA": "testlabelA",
		"labelB": "testlabelB",
		"labelC": "testlabelC2",
	}

	newStatusLabels := map[string]string{
		"labelA": "testlabelA",
		"labelC": "testlabelC",
		"labelD": "testlabelD",
	}

	namespaceLabel.Spec.Labels = newSpecLabels
	namespaceLabel.Status.ActiveLabels = newStatusLabels

	obj := []client.Object{namespaceLabel}
	cl, s, err := setupClient(obj)
	if err != nil {
		t.Fatalf("Unable to add to scheme: %v", err)
	}

	// create a NamespaceLabelReconciler object with the scheme and fake client
	r := &NamespaceLabelReconciler{cl, s}

	// run function to test
	addLabels, delLabels := r.getNamespaceLabelsDiffs(namespaceLabel)

	// set expected result and check result matches expected
	g.Expect(func() bool {
		expectedAddLabels := map[string]string{
			"labelB": "testlabelB",
			"labelC": "testlabelC2",
		}
		expectedDelLabels := map[string]string{
			"labelD": "testlabelD",
		}

		return reflect.DeepEqual(addLabels, expectedAddLabels) && reflect.DeepEqual(delLabels, expectedDelLabels)
	}()).To(BeTrue())
}

func TestUpdateNSLabels(t *testing.T) {
	g := NewGomegaWithT(t)
	RegisterFailHandler(ginkgo.Fail)

	namespace := generateNamespaceObject()

	addLabels := map[string]string{
		"labelB": "testlabelB",
		"labelC": "testlabelC2",
	}
	delLabels := map[string]string{
		"labelD": "testlabelD",
	}

	obj := []client.Object{namespace}
	cl, s, err := setupClient(obj)
	if err != nil {
		t.Fatalf("Unable to add to scheme: %v", err)
	}

	// create a NamespaceLabelReconciler object with the scheme and fake client
	r := &NamespaceLabelReconciler{cl, s}

	// run function to test
	if err := r.updateNSLabels(context.TODO(), namespace, addLabels, delLabels); err != nil {
		t.Fatalf("Unable to add to update NS Labels: %v", err)
	}

	// set expected result and check result matches expected
	g.Expect(func() bool {
		expectedLabels := map[string]string{
			"kubernetes.io/name": "default",
			LabelKey:             LabelVal,
			"labelB":             "testlabelB",
			"labelC":             "testlabelC2",
		}

		return !reflect.DeepEqual(namespace.ObjectMeta.Labels, expectedLabels)
	}()).To(BeTrue())
}
