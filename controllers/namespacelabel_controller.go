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
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	danaiov1alpha1 "home-assignment/api/v1alpha1"
)

const protectedLabelDomain = "kubernetes.io"

// NamespaceLabelReconciler reconciles a NamespaceLabel object
type NamespaceLabelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=dana.io.dana.io,resources=namespacelabels,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dana.io.dana.io,resources=namespacelabels/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dana.io.dana.io,resources=namespacelabels/finalizers,verbs=update
//+kubebuilder:rbac:groups=dana.io.dana.io,resources=namespaces,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NamespaceLabel object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *NamespaceLabelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Processing NamespaceLabelReconciler")

	// we'll fetch the NamespaceLabel using our client
	var namespaceLabel danaiov1alpha1.NamespaceLabel
	if err := r.Get(ctx, req.NamespacedName, &namespaceLabel); err != nil {
		if errors.IsNotFound(err) {
			// request object not found, could have been deleted after reconcile request
			// return and don't requeue
			log.Info("NamespaceLabel resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "unable to fetch namespaceLabel")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// get list of labels from request
	reqLabels := namespaceLabel.Labels

	// we'll fetch the current namespace using our client
	var namespace v1.Namespace
	if err := r.Get(ctx, req.NamespacedName, &namespace); err != nil {
		log.Error(err, "unable to fetch namespace")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// kubernetes reserves all labels and annotations in the kubernetes.io namespace
	// therefore we do not apply the changes the NamespaceLabel object requires
	// if the list of labels includes such labels

	// loop over the map to check if such labels exists
	for key := range reqLabels {
		labelDomain := strings.Split(key, "/")[0]
		if strings.HasSuffix(labelDomain, protectedLabelDomain) {
			errorMsg := fmt.Errorf("changing labels of the %s namespace is not allowed", protectedLabelDomain)
			return ctrl.Result{}, errorMsg
		}
	}

	// set the namespace labels to match the request
	namespace.ObjectMeta.Labels = reqLabels

	// update the namespace with the new labels
	if err := r.Update(ctx, &namespace); err != nil {
		log.Error(err, "Failed to update namespace", namespace.Name)
		return ctrl.Result{}, err
	}

	// update status of namespaceLabel
	if err := r.Status().Update(ctx, &namespaceLabel); err != nil {
		log.Error(err, "unable to update namespaceLabel status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceLabelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&danaiov1alpha1.NamespaceLabel{}).
		Complete(r)
}
