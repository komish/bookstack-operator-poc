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
	"fmt"

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

// BookStackConfigMapReconciler reconciles the deployment resource.
type BookStackConfigMapReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=tools.opdev.io,resources=bookstacks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tools.opdev.io,resources=bookstacks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tools.opdev.io,resources=bookstacks/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=configmaps/finalizers,verbs=update

// Reconcile will ensure that the Kubernetes Configmap for BookStack
// reaches the desired state.
func (r *BookStackConfigMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("configmap reconciliation initiated.")
	defer l.Info("configmap reconciliation complete.")
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
	newAppCM := instance.NewAppConfigMap()

	// get service persisted to API and extract node port for use in config
	newService := instance.NewService()
	var existingSvc corev1.Service
	if err = r.Client.Get(ctx, client.ObjectKeyFromObject(&newService), &existingSvc); err != nil {
		// we didn't find the service, so block requeue. This blocks the creation
		// of the config map until the service is up.
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	nodePort := existingSvc.Spec.Ports[0].NodePort
	newAppCM.Data["APP_URL"] = fmt.Sprintf("http://127.0.0.1:%d/", nodePort)

	err = ctrl.SetControllerReference(&instance, &newAppCM, r.Scheme)
	if err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	// If app configmap exists, get it and patch it
	var existingAppCM corev1.ConfigMap
	err = r.Client.Get(ctx, client.ObjectKeyFromObject(&newAppCM), &existingAppCM)

	if apierrors.IsNotFound(err) {
		// create the resource because it does not exist.
		l.Info("creating resource", newAppCM.Kind, newAppCM.Name)
		if err := r.Client.Create(ctx, &newAppCM); err != nil {
			return subrec.Evaluate(subrec.RequeueWithError(err))
		}
	}

	if err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	l.Info("updating resources if necessary", existingAppCM.Kind, existingAppCM.GetName())
	patchDiff := client.MergeFrom(&existingAppCM)
	if err = mergo.Merge(&existingAppCM, newAppCM, mergo.WithOverride); err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	if err = r.Patch(ctx, &existingAppCM, patchDiff); err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	// db
	newDBCM := instance.NewDBConfigMap()

	err = ctrl.SetControllerReference(&instance, &newDBCM, r.Scheme)
	if err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	// If app configmap exists, get it and patch it
	var existingDBCM corev1.ConfigMap
	err = r.Client.Get(ctx, client.ObjectKeyFromObject(&newDBCM), &existingDBCM)

	if apierrors.IsNotFound(err) {
		// create the resource because it does not exist.
		l.Info("creating resource", newDBCM.Kind, newDBCM.Name)
		if err := r.Client.Create(ctx, &newDBCM); err != nil {
			return subrec.Evaluate(subrec.RequeueWithError(err))
		}
	}

	if err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	l.Info("updating resources if necessary", existingDBCM.Kind, existingDBCM.GetName())
	DBCMpatchDiff := client.MergeFrom(&existingDBCM)
	if err = mergo.Merge(&existingDBCM, newDBCM, mergo.WithOverride); err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	if err = r.Patch(ctx, &existingDBCM, DBCMpatchDiff); err != nil {
		return subrec.Evaluate(subrec.RequeueWithError(err))
	}

	return subrec.Evaluate(subrec.DoNotRequeue()) // success
}

// SetupWithManager sets up the controller with the Manager.
func (r *BookStackConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&toolsv1alpha1.BookStack{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
