/*
Copyright 2022 The OpDev Team.

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

	"github.com/imdario/mergo"
	toolsv1alpha1 "github.com/opdev/bookstack-operator/api/v1alpha1"
	subrec "github.com/opdev/subreconciler"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// BookStackAppStorageReconciler reconciles the deployment resource.
type BookStackAppStorageReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=tools.opdev.io,resources=bookstacks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tools.opdev.io,resources=bookstacks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tools.opdev.io,resources=bookstacks/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=persistentvolumes,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=persistentvolumes/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaim/finalizers,verbs=update

// Reconcile will ensure that the Kubernetes Secret for BookStack
// reaches the desired state.
func (r *BookStackAppStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("app storage reconciliation initiated.")
	defer l.Info("app storage reconciliation complete.")
	bookstackInstanceKey := req.NamespacedName

	// Get the BookStack instance to make sure it still exists.
	var instance toolsv1alpha1.BookStack
	err := r.Client.Get(ctx, bookstackInstanceKey, &instance)

	if apierrors.IsNotFound(err) {
		return subrec.Evaluate(subrec.DoNotRequeue())
	}

	if err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	// app pv
	appPV := instance.NewAppPersistentVolume()

	err = ctrl.SetControllerReference(&instance, &appPV, r.Scheme)
	if err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	// If app secret exists, get it and patch it
	var existingPV corev1.PersistentVolume
	err = r.Client.Get(ctx, client.ObjectKeyFromObject(&appPV), &existingPV)

	if apierrors.IsNotFound(err) {
		// create the resource because it does not exist.
		l.Info("creating resource", appPV.Kind, appPV.Name)
		if err := r.Client.Create(ctx, &appPV); err != nil {
			return subrec.Evaluate(subrec.RequeueWithError(err))
		}
	}

	if err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	l.Info("updating resources if necessary", existingPV.Kind, existingPV.GetName())
	patchDiff := client.MergeFrom(&existingPV)
	if err = mergo.Merge(&existingPV, appPV, mergo.WithOverride); err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	if err = r.Patch(ctx, &existingPV, patchDiff); err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	// app pvc
	newAppPVC := instance.NewAppPersistentVolumeClaim()

	err = ctrl.SetControllerReference(&instance, &newAppPVC, r.Scheme)
	if err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	// If db secret exists, get it and patch it
	var existingAppPVC corev1.PersistentVolumeClaim
	err = r.Client.Get(ctx, client.ObjectKeyFromObject(&newAppPVC), &existingAppPVC)

	if apierrors.IsNotFound(err) {
		// create the resource because it does not exist.
		l.Info("creating resource", newAppPVC.Kind, newAppPVC.Name)
		if err := r.Client.Create(ctx, &newAppPVC); err != nil {
			return subrec.Evaluate(subrec.RequeueWithError(err))
		}
	}

	if err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	l.Info("updating resources if necessary", existingAppPVC.Kind, existingAppPVC.GetName())
	appPVCpatchDiff := client.MergeFrom(&existingAppPVC)
	if err = mergo.Merge(&existingAppPVC, newAppPVC, mergo.WithOverride); err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	if err = r.Patch(ctx, &existingAppPVC, appPVCpatchDiff); err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	return subrec.Evaluate(subrec.DoNotRequeue()) // success
}

// SetupWithManager sets up the controller with the Manager.
func (r *BookStackAppStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&toolsv1alpha1.BookStack{}).
		Owns(&corev1.PersistentVolume{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}
