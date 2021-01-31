/*
Copyright 2021.

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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakv1alpha1 "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"

	multiclusterkeycloakv1alpha1 "github.com/mdelder/multicluster-keycloak-operator/api/v1alpha1"
)

var applyFiles = []string{
	"manifests/keycloak-realm.yaml",
	"manifests/ca-config-map-manifestwork.yaml",
	"manifests/client-secret-manifestwork.yaml",
	"manifests/oauth-manifestwork.yaml",
}

// AuthorizationDomainReconciler reconciles a AuthorizationDomain object
type AuthorizationDomainReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=keycloak.open-cluster-management.io,resources=authorizationdomains,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=keycloak.open-cluster-management.io,resources=authorizationdomains/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=keycloak.open-cluster-management.io,resources=authorizationdomains/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AuthorizationDomain object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *AuthorizationDomainReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("authorizationdomain", req.NamespacedName)

	authzDomain := &multiclusterkeycloakv1alpha1.AuthorizationDomain{}
	if err := r.Get(ctx, req.NamespacedName, authzDomain); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
	}
	r.Log.Info("Reconciling", "AuthorizationDomain", authzDomain)

	// Check if this deployment already exists in the specified namespace
	found := &keycloakv1alpha1.KeycloakRealm{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: authzDomain.ObjectMeta.Name, Namespace: "keycloak"}, found)

	// If not exists, then create it
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating KeycloakRealm", "KeycloakRealm", found)
		realm := r.createKeycloakRealm(authzDomain)
		err = r.Client.Create(context.TODO(), realm)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AuthorizationDomainReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multiclusterkeycloakv1alpha1.AuthorizationDomain{}).
		Complete(r)
}

func (r *AuthorizationDomainReconciler) createKeycloakRealm(authzDomain *multiclusterkeycloakv1alpha1.AuthorizationDomain) *keycloakv1alpha1.KeycloakRealm {
	realm := &keycloakv1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      authzDomain.ObjectMeta.Name,
			Namespace: "keycloak",
			Labels: map[string]string{
				"app": "sso",
			},
		},
		Spec: keycloakv1alpha1.KeycloakRealmSpec{
			Realm: &keycloakv1alpha1.KeycloakAPIRealm{
				ID:          authzDomain.ObjectMeta.Name,
				Realm:       authzDomain.ObjectMeta.Name,
				Enabled:     true,
				DisplayName: fmt.Sprintf("%s Realm", authzDomain.ObjectMeta.Name),
				IdentityProviders: []*keycloakv1alpha1.KeycloakIdentityProvider{
					{
						Alias:                     "github",
						TrustEmail:                true,
						ProviderID:                "github",
						FirstBrokerLoginFlowAlias: "first broker login",
						Config: map[string]string{
							"clientId":     authzDomain.ObjectMeta.Name,
							"clientSecret": authzDomain.ObjectMeta.Name,
							"useJwksUrl":   "true",
						},
					},
				},
			},
			InstanceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "sso",
				},
			},
		},
	}
	return realm
}
