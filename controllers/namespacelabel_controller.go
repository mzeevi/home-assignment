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

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	danaiov1alpha1 "home-assignment/apis/namespacelabel/v1alpha1"
)

const NamespaceLabelFinalizer = "dana.io/namespacelabel-finalizer"

// NamespaceLabelReconciler reconciles a NamespaceLabel object
type NamespaceLabelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=dana.io.dana.io,resources=namespacelabels,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dana.io.dana.io,resources=namespacelabels/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dana.io.dana.io,resources=namespacelabels/finalizers,verbs=update
//+kubebuilder:rbac:groups=*,resources=namespaces,verbs=get;list;watch;update;patch

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
	reqLabels := namespaceLabel.Spec.Labels

	// we'll fetch the current namespace using our client
	namespace := v1.Namespace{}
	nsNamespacedName := types.NamespacedName{
		Namespace: req.NamespacedName.Namespace,
		Name:      req.NamespacedName.Namespace,
	}

	if err := r.Get(ctx, nsNamespacedName, &namespace); err != nil {
		log.Error(err, "unable to fetch namespace")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	finalizerName := NamespaceLabelFinalizer
	// examine DeletionTimestamp to determine if object is under deletion
	if namespaceLabel.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer
		if !controllerutil.ContainsFinalizer(&namespaceLabel, finalizerName) {
			controllerutil.AddFinalizer(&namespaceLabel, finalizerName)
			if err := r.Update(ctx, &namespaceLabel); err != nil {
				log.Error(err, "failed to update namespaceLabel")
				return ctrl.Result{}, err
			}
		}
	} else {
		// the object is being deleted
		if controllerutil.ContainsFinalizer(&namespaceLabel, finalizerName) {
			// our finalizer is present, so lets handle any external dependency
			r.deleteLabels(&namespaceLabel, &namespace)

			// remove our finalizer from the list and update it
			controllerutil.RemoveFinalizer(&namespaceLabel, finalizerName)
			if err := r.Update(ctx, &namespaceLabel); err != nil {
				log.Error(err, "failed to update namespaceLabel")
				return ctrl.Result{}, err
			}

			if err := r.Update(ctx, &namespace); err != nil {
				log.Error(err, "failed to update namespace")
				return ctrl.Result{}, err
			}
		}
		// stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// add the namespace labels to match the request
	if namespace.ObjectMeta.Labels == nil {
		namespace.ObjectMeta.Labels = make(map[string]string)
	}

	for key, val := range reqLabels {
		namespace.ObjectMeta.Labels[key] = val
	}

	// update the namespace with the new labels
	if err := r.Update(ctx, &namespace); err != nil {
		log.Error(err, "failed to update namespace")
		return ctrl.Result{}, err
	}

	// update status of namespaceLabel
	namespaceLabel.Status.ActiveLabels = reqLabels
	if err := r.Status().Update(ctx, &namespaceLabel); err != nil {
		log.Error(err, "unable to update namespaceLabel status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NamespaceLabelReconciler) deleteLabels(namespaceLabel *danaiov1alpha1.NamespaceLabel, namespace *v1.Namespace) {
	// delete the labels from the namespace
	reqLabels := namespaceLabel.Spec.Labels
	for key := range reqLabels {
		delete(namespace.ObjectMeta.Labels, key)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceLabelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&danaiov1alpha1.NamespaceLabel{}).
		Complete(r)
}
