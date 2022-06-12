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
func (r *NamespaceLabelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Processing NamespaceLabelReconciler")

	// fetch the NamespaceLabel using our client
	var namespaceLabel danaiov1alpha1.NamespaceLabel
	if err := r.Get(ctx, req.NamespacedName, &namespaceLabel); err != nil {
		if errors.IsNotFound(err) {
			// request object not found, could have been deleted after reconcile request
			// return and don't requeue
			log.Info("NamespaceLabel resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// error reading the object - requeue the request.
		log.Error(err, "unable to fetch namespaceLabel")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// fetch the current namespace using our client
	namespace := v1.Namespace{}
	nsNamespacedName := types.NamespacedName{
		Namespace: req.NamespacedName.Namespace,
		Name:      req.NamespacedName.Namespace,
	}

	if err := r.Get(ctx, nsNamespacedName, &namespace); err != nil {
		log.Error(err, "unable to fetch namespace")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// examine DeletionTimestamp to determine if object is under deletion
	if !namespaceLabel.ObjectMeta.DeletionTimestamp.IsZero() {
		// handle finalizer deletion on object
		if err := r.deleteFinalizer(ctx, &namespaceLabel, &namespace); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// The object is not being deleted, so if it does not have our finalizer,
	// then lets add the finalizer and update the object. This is equivalent
	// registering our finalizer
	if err := r.addFinalizer(ctx, &namespaceLabel, &namespace); err != nil {
		return ctrl.Result{}, nil
	}

	finalizerName := NamespaceLabelFinalizer
	if !controllerutil.ContainsFinalizer(&namespaceLabel, finalizerName) {
		controllerutil.AddFinalizer(&namespaceLabel, finalizerName)
		if err := r.Update(ctx, &namespaceLabel); err != nil {
			log.Error(err, "failed to update namespaceLabel")
			return ctrl.Result{}, err
		}
	}

	addLabels, delLabels := r.getNamespaceLabelsDiffs(&namespaceLabel)
	if err := r.updateNSLabels(ctx, &namespace, addLabels, delLabels); err != nil {
		return ctrl.Result{}, err
	}

	// update status of namespaceLabel to match request
	namespaceLabel.Status.ActiveLabels = namespaceLabel.Spec.Labels
	if err := r.Status().Update(ctx, &namespaceLabel); err != nil {
		log.Error(err, "unable to update namespaceLabel status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NamespaceLabelReconciler) deleteLabels(namespaceLabel *danaiov1alpha1.NamespaceLabel, namespace *v1.Namespace) {
	// delete the labels from the namespace
	actLabels := namespaceLabel.Status.ActiveLabels
	for key := range actLabels {
		delete(namespace.ObjectMeta.Labels, key)
	}
}

func (r *NamespaceLabelReconciler) deleteFinalizer(ctx context.Context, namespaceLabel *danaiov1alpha1.NamespaceLabel, namespace *v1.Namespace) error {
	log := log.FromContext(ctx)
	log.Info("Handling finalizer deletion")

	finalizerName := NamespaceLabelFinalizer
	if controllerutil.ContainsFinalizer(namespaceLabel, finalizerName) {
		// our finalizer is present, so lets handle any external dependency
		r.deleteLabels(namespaceLabel, namespace)

		// remove our finalizer from the list and update it
		controllerutil.RemoveFinalizer(namespaceLabel, finalizerName)
		if err := r.Update(ctx, namespaceLabel); err != nil {
			log.Error(err, "failed to update namespaceLabel")
			return err
		}

		if err := r.Update(ctx, namespace); err != nil {
			log.Error(err, "failed to update namespace")
			return err
		}
	}
	return nil
}

func (r *NamespaceLabelReconciler) addFinalizer(ctx context.Context, namespaceLabel *danaiov1alpha1.NamespaceLabel, namespace *v1.Namespace) error {
	// The object is not being deleted, so if it does not have our finalizer,
	// then lets add the finalizer and update the object. This is equivalent
	// registering our finalizer
	log := log.FromContext(ctx)
	log.Info("Handling finalizer addition")
	finalizerName := NamespaceLabelFinalizer
	if !controllerutil.ContainsFinalizer(namespaceLabel, finalizerName) {
		controllerutil.AddFinalizer(namespaceLabel, finalizerName)
		if err := r.Update(ctx, namespaceLabel); err != nil {
			log.Error(err, "failed to update namespaceLabel")
			return err
		}
	}
	return nil
}

// this function compares the status and the spec of a NamespaceLabel object
// and returns two maps: one map indicates which labels to add/amend
// the second map indicates which labels to delete from the namespace
func (r *NamespaceLabelReconciler) getNamespaceLabelsDiffs(namespaceLabel *danaiov1alpha1.NamespaceLabel) (map[string]string, map[string]string) {
	addLabels := make(map[string]string)
	delLabels := make(map[string]string)

	reqLabels := namespaceLabel.Spec.Labels
	actLabels := namespaceLabel.Status.ActiveLabels

	if actLabels == nil {
		addLabels = reqLabels
		return addLabels, delLabels
	}

	// loop over requested labels and check if they don't exist in the active labels
	// if so, add to addLabels map
	for reqKey, reqVal := range reqLabels {
		if actVal, ok := actLabels[reqKey]; !ok || actVal != reqVal {
			addLabels[reqKey] = reqVal
		}
	}

	// loop over active labels and check if they don't exist in the requested labels
	// if so, add to delLabels map
	for actKey, actVal := range actLabels {
		if _, ok := reqLabels[actKey]; !ok {
			delLabels[actKey] = actVal
		}
	}

	return addLabels, delLabels
}

func (r *NamespaceLabelReconciler) updateNSLabels(ctx context.Context, namespace *v1.Namespace, addLabels map[string]string, delLabels map[string]string) error {
	log := log.FromContext(ctx)
	log.Info("Updating namespace labels")

	// add the namespace labels to match the request
	if namespace.ObjectMeta.Labels == nil {
		namespace.ObjectMeta.Labels = make(map[string]string)
	}

	for key, val := range addLabels {
		namespace.ObjectMeta.Labels[key] = val
	}

	for key := range delLabels {
		delete(namespace.ObjectMeta.Labels, key)
	}

	// update the namespace with the new labels
	if err := r.Update(ctx, namespace); err != nil {
		log.Error(err, "failed to update namespace")
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceLabelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&danaiov1alpha1.NamespaceLabel{}).
		Complete(r)
}
