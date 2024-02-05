/*
Copyright 2024.

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

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	webappv1 "test.com/managedsecret/api/v1"
)

const managedSecretFinalizer = "webapp.test.com/finalizer"

// ManagedSecretReconciler reconciles a ManagedSecret object
type ManagedSecretReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=webapp.test.com,resources=managedsecrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=webapp.test.com,resources=managedsecrets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=webapp.test.com,resources=managedsecrets/finalizers,verbs=update
//+kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ManagedSecret object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *ManagedSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var managedSecret webappv1.ManagedSecret
	if err := r.Get(ctx, req.NamespacedName, &managedSecret); err != nil {
		logger.Error(err, "Get Fail.")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check ManagedSecret Deletion
	if managedSecret.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&managedSecret, managedSecretFinalizer) {
			controllerutil.AddFinalizer(&managedSecret, managedSecretFinalizer)
			if err := r.Update(ctx, &managedSecret); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(&managedSecret, managedSecretFinalizer) {
			for _, ns := range managedSecret.Spec.TargetNamespaces {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      managedSecret.Spec.SecretName,
						Namespace: ns,
					},
				}
				if err := r.Delete(ctx, secret); client.IgnoreNotFound(err) != nil {
					return ctrl.Result{}, err
				}
			}

			controllerutil.RemoveFinalizer(&managedSecret, managedSecretFinalizer)
			if err := r.Update(ctx, &managedSecret); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Convert secretContent to the secretData
	secretData := make(map[string][]byte)
	for key, value := range managedSecret.Spec.SecretContent {
		secretData[key] = []byte(value)
	}

	// Create or Update Secrets to the required namespaces
	for _, ns := range managedSecret.Spec.TargetNamespaces {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managedSecret.Spec.SecretName,
				Namespace: ns,
			},
			Data: secretData,
		}
		if err := r.Create(ctx, secret); err != nil {
			if !errors.IsAlreadyExists(err) {
				logger.Error(err, "Create Fail.")
				return ctrl.Result{}, err
			}
			// Update it if already exists
			if err := r.Update(ctx, secret); err != nil {
				logger.Error(err, "Update Fail.")
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ManagedSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.ManagedSecret{}).
		Complete(r)
}
