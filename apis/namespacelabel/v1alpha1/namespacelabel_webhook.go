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
	"fmt"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var namespacelabellog = logf.Log.WithName("namespacelabel-resource")

func (r *NamespaceLabel) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-dana-io-dana-io-v1alpha1-namespacelabel,mutating=false,failurePolicy=fail,sideEffects=None,groups=dana.io.dana.io,resources=namespacelabels,verbs=create;update;delete,versions=v1alpha1,name=vnamespacelabel.kb.io,admissionReviewVersions=v1
//+kubebuilder:rbac:groups=*,resources=namespaces,verbs=get;list

var _ webhook.Validator = &NamespaceLabel{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NamespaceLabel) ValidateCreate() error {
	namespacelabellog.Info("validate create", "name", r.Name)

	if res := r.CheckNamespaceLabelName(); !res {
		err := fmt.Errorf("NamespaceLabel name must be equal to the name of its namespace")
		namespacelabellog.Error(err, "unable to crate namespacelabel")

		return err
	}

	if err := r.CheckLabelNS(); err != nil {
		return err
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NamespaceLabel) ValidateUpdate(old runtime.Object) error {
	namespacelabellog.Info("validate update", "name", r.Name)

	return r.CheckLabelNS()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NamespaceLabel) ValidateDelete() error {
	namespacelabellog.Info("validate delete", "name", r.Name)

	return nil
}

func (r *NamespaceLabel) CheckNamespaceLabelName() bool {
	nsLabelName := r.Name
	nsLabelNamespace := r.Namespace

	return nsLabelName == nsLabelNamespace
}

func (r *NamespaceLabel) CheckLabelNS() error {
	// get controller config map values from environment variable
	controllerConfigMapKey := os.Getenv("PROTECTED_MANAGEMENT_LABELS_DOMAINS")

	if controllerConfigMapKey != "" {

		protectedDomains := strings.Split(controllerConfigMapKey, ",")

		for key := range r.Spec.Labels {
			reqlabelDomain := strings.Split(key, "/")[0]
			for _, dom := range protectedDomains {
				if strings.HasSuffix(reqlabelDomain, dom) {
					errorMsg := fmt.Errorf("setting labels of the %s domain is not allowed", dom)
					return errorMsg
				}
			}
		}
	}
	return nil
}
