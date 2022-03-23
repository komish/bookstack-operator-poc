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

// BookStackSecretReconciler reconciles the deployment resource.
type BookStackSecretReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=tools.opdev.io,resources=bookstacks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tools.opdev.io,resources=bookstacks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tools.opdev.io,resources=bookstacks/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=configmaps/finalizers,verbs=update

// Reconcile will ensure that the Kubernetes Secret for BookStack
// reaches the desired state.
func (r *BookStackSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("secret reconciliation initiated.")
	defer l.Info("secret reconciliation complete.")
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

	// app
	newAppSecret := instance.NewAppSecret()

	err = ctrl.SetControllerReference(&instance, &newAppSecret, r.Scheme)
	if err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	// If app secret exists, get it and patch it
	var existingAppSecret corev1.Secret
	err = r.Client.Get(ctx, client.ObjectKeyFromObject(&newAppSecret), &existingAppSecret)

	if apierrors.IsNotFound(err) {
		// create the resource because it does not exist.
		l.Info("creating resource", newAppSecret.Kind, newAppSecret.Name)
		if err := r.Client.Create(ctx, &newAppSecret); err != nil {
			return subrec.Evaluate(subrec.RequeueWithError(err))
		}
	}

	if err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	l.Info("updating resources if necessary", existingAppSecret.Kind, existingAppSecret.GetName())
	patchDiff := client.MergeFrom(&existingAppSecret)
	if err = mergo.Merge(&existingAppSecret, newAppSecret, mergo.WithOverride); err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	if err = r.Patch(ctx, &existingAppSecret, patchDiff); err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	// db
	newDBSecret := instance.NewDBSecret()

	err = ctrl.SetControllerReference(&instance, &newDBSecret, r.Scheme)
	if err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	// If db secret exists, get it and patch it
	var existingDBSecret corev1.Secret
	err = r.Client.Get(ctx, client.ObjectKeyFromObject(&newDBSecret), &existingDBSecret)

	if apierrors.IsNotFound(err) {
		// create the resource because it does not exist.
		l.Info("creating resource", newDBSecret.Kind, newDBSecret.Name)
		if err := r.Client.Create(ctx, &newDBSecret); err != nil {
			return subrec.Evaluate(subrec.RequeueWithError(err))
		}
	}

	if err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	l.Info("updating resources if necessary", existingDBSecret.Kind, existingDBSecret.GetName())
	DBCMpatchDiff := client.MergeFrom(&existingDBSecret)
	if err = mergo.Merge(&existingDBSecret, newDBSecret, mergo.WithOverride); err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	if err = r.Patch(ctx, &existingDBSecret, DBCMpatchDiff); err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	return subrec.Evaluate(subrec.DoNotRequeue()) // success
}

// SetupWithManager sets up the controller with the Manager.
func (r *BookStackSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&toolsv1alpha1.BookStack{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
