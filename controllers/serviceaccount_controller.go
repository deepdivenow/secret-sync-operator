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
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	authv1alpha1 "github.com/deepdivenow/secret-sync-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServiceAccountReconciler reconciles a ServiceAccount object
type ServiceAccountReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=auth.itsumma.ru,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=auth.itsumma.ru,resources=serviceaccounts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=auth.itsumma.ru,resources=serviceaccounts/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ServiceAccount object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ServiceAccountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	//instance := &corev1.ServiceAccount{}
	authSA := &authv1alpha1.ServiceAccount{}
	err := r.Get(context.TODO(), req.NamespacedName, authSA)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("auth.ServiceAccount resource not found. Ignoring since object must be deleted")
			return reconcile.Result{}, nil
		}
		log.Error(err, "Failed to get auth.ServiceAccount ")
		return reconcile.Result{}, err
	}
	// Check if the SA already exists, if not create a new one
	foundSA := &corev1.ServiceAccount{}
	err = r.Get(ctx, types.NamespacedName{Name: authSA.Name, Namespace: authSA.Namespace}, foundSA)
	if err != nil && errors.IsNotFound(err) {
		// Define a new coreSA
		coreSA := r.coreSAForauthSA(authSA)
		log.Info("Creating a new corev1.ServiceAccount", "coreSA.Namespace", coreSA.Namespace, "coreSA.Name", coreSA.Name)
		err = r.Create(ctx, coreSA)
		if err != nil {
			log.Error(err, "Failed to create new corev1.SA", "coreSA.Namespace", coreSA.Namespace, "coreSA.Name", coreSA.Name)
			return ctrl.Result{}, err
		}
		// coreSA created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get coreSA")
		return ctrl.Result{}, err
	}
	if len(foundSA.Secrets) > 1 {
		var newSecrets []string
		for _, s := range foundSA.Secrets {
			if s.Kind == "Secret" {
				newSecrets = append(newSecrets, s.Name)
			}
		}
		if len(newSecrets) > 1 {
			authSA.Status.Secrets = newSecrets
			r.Update(ctx, authSA)
		}
	}

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceAccountReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.ServiceAccount{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Complete(r)
}

func (r *ServiceAccountReconciler) coreSAForauthSA(sa *authv1alpha1.ServiceAccount) *corev1.ServiceAccount {
	coreSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sa.Name,
			Namespace: sa.Namespace,
		},
	}
	// Set authSA instance as the owner and controller
	ctrl.SetControllerReference(sa, coreSA, r.Scheme)
	return coreSA
}

// labelsForApp creates a simple set of labels for Memcached.
func labelsForApp(name string) map[string]string {
	return map[string]string{"cr_name": name}
}
