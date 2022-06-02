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

package v1alpha1

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var namespacelabellog = logf.Log.WithName("namespacelabel-resource")

const protectedLabelDomain = "kubernetes.io"

func (r *NamespaceLabel) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-dana-io-dana-io-v1alpha1-namespacelabel,mutating=false,failurePolicy=fail,sideEffects=None,groups=dana.io.dana.io,resources=namespacelabels,verbs=create;update,versions=v1alpha1,name=vnamespacelabel.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &NamespaceLabel{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NamespaceLabel) ValidateCreate() error {
	namespacelabellog.Info("validate create", "name", r.Name)

	cl, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		namespacelabellog.Error(err, "failed to create client")
		return err
	}

	// fail if there are multiple (> 1) NamespaceLabel objects running at the same time
	var namespaceLabelList NamespaceLabelList
	if err := cl.List(context.Background(), &namespaceLabelList, client.InNamespace(r.ObjectMeta.Namespace)); err != nil {
		namespacelabellog.Error(err, "unable to list namespaceLabels")
		return err
	}

	if len(namespaceLabelList.Items) > 1 {
		errorMsg := fmt.Errorf("only one namespaceLabel object can be set on a namespace")
		return errorMsg
	}

	if r.CheckLabelNS() {
		errorMsg := fmt.Errorf("changing labels of the %s namespace is not allowed", protectedLabelDomain)
		return errorMsg
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NamespaceLabel) ValidateUpdate(old runtime.Object) error {
	namespacelabellog.Info("validate update", "name", r.Name)

	if r.CheckLabelNS() {
		errorMsg := fmt.Errorf("changing labels of the %s namespace is not allowed", protectedLabelDomain)
		return errorMsg
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NamespaceLabel) ValidateDelete() error {
	namespacelabellog.Info("validate delete", "name", r.Name)

	return nil
}

func (r *NamespaceLabel) CheckLabelNS() bool {
	for key := range r.Spec.Labels {
		labelDomain := strings.Split(key, "/")[0]
		if strings.HasSuffix(labelDomain, protectedLabelDomain) {
			return true
		}
	}
	return false
}
