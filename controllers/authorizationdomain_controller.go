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
	"reflect"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ocmclusterv1 "github.com/open-cluster-management/api/cluster/v1"

	// openshiftconfigv1 "github.com/openshift/api/config/v1"

	keycloakv1alpha1 "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"

	multiclusterkeycloakv1alpha1 "github.com/mdelder/multicluster-keycloak-operator/api/v1alpha1"
)

const baseDomain = "demo.red-chesterfield.com" // temporary

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

	isAuthzDomainMarkedToBeDeleted := authzDomain.GetDeletionTimestamp() != nil
	if isAuthzDomainMarkedToBeDeleted {
		r.Log.Info("AuthorizationDomain marked for deletion", "AuthorizationDomain", authzDomain)
		// Add a finalizer to AD to remove the generated ManifestWork _per_ ManagedCluster
	}

	// Create the Realm
	found := &keycloakv1alpha1.KeycloakRealm{}
	realm, err := r.createKeycloakRealm(authzDomain)
	if err != nil {
		r.Log.Info("Failed to create KeycloakRealm", "KeycloakRealm", realm)
		return ctrl.Result{}, err
	}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: authzDomain.Name, Namespace: "keycloak"}, found)
	switch {
	case errors.IsNotFound(err):
		r.Log.Info("Creating KeycloakRealm", "KeycloakRealm", realm)
		return ctrl.Result{}, r.Client.Create(context.TODO(), realm)
	case err != nil:
		return ctrl.Result{}, err
	}
	if !reflect.DeepEqual(found.Spec, realm.Spec) {
		realm.Spec.DeepCopyInto(&found.Spec)
		r.Log.Info("Updating KeycloakRealm", "KeycloakRealm", found)
		if err := r.Client.Update(context.TODO(), found); err != nil {
			return ctrl.Result{}, err
		}
	}

	// For each ManagedCluster, create the KeycloakClient, ManifestWork
	managedClusterList := &ocmclusterv1.ManagedClusterList{}
	err = r.Client.List(context.Background(), managedClusterList)
	errs := []error{}
	for _, cluster := range managedClusterList.Items {
		r.Log.Info("Discovered ManagedCluster", "ManagedClusterName", cluster.Name)

		clusterContext, err := newManagedClusterSSOContext(r.Client, authzDomain, &cluster, baseDomain)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := clusterContext.createOrUpdateKeycloakClient(); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := clusterContext.createOrUpdateManifestWork(); err != nil {
			errs = append(errs, err)
			continue
		}
	}
	if len(errs) > 0 {
		return ctrl.Result{}, utilerrors.NewAggregate(errs)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AuthorizationDomainReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multiclusterkeycloakv1alpha1.AuthorizationDomain{}).
		// watch the Secret / ConfigMap referenced by AuthorizationDomain
		Owns(&keycloakv1alpha1.KeycloakRealm{}).
		Owns(&keycloakv1alpha1.KeycloakClient{}).
		Complete(r)
}

func (r *AuthorizationDomainReconciler) createKeycloakRealm(authzDomain *multiclusterkeycloakv1alpha1.AuthorizationDomain) (*keycloakv1alpha1.KeycloakRealm, error) {
	secret := &corev1.Secret{}
	githubClientID, githubClientSecret := "", ""
	if authzDomain.Spec.IdentityProviders != nil {
		// There is only 1 supported IdentityProvider; so the for loop should never have more than 1 iteration at present
		for _, provider := range authzDomain.Spec.IdentityProviders {
			if provider.Type != "github" {
				r.Log.Info("Missing provider \"type\". Currently supported types: {\"github\"}.", "Type", provider.Type)
			} else if provider.SecretRef == "" {
				r.Log.Info("Missing provider \"secretRef\". Currently supported types: {\"github\"}.")
			} else {
				if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: provider.SecretRef, Namespace: "keycloak"}, secret); err != nil {
					r.Log.Error(err, "Could not find the referenced secret for IdentityProvider", "AuthorizationDomain", authzDomain, "SecretRef", provider.SecretRef)
				} else {
					githubClientID = string(secret.Data["clientId"])
					githubClientSecret = string(secret.Data["clientSecret"])
					break
				}
			}
		}
	}
	realm := &keycloakv1alpha1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      authzDomain.Name,
			Namespace: "keycloak",
			Labels: map[string]string{
				"app": "sso",
				// "realm": authzDomain.Name,
			},
		},
		Spec: keycloakv1alpha1.KeycloakRealmSpec{
			Realm: &keycloakv1alpha1.KeycloakAPIRealm{
				ID:          authzDomain.Name,
				Realm:       authzDomain.Name,
				Enabled:     true,
				DisplayName: fmt.Sprintf("%s Realm", authzDomain.Name),
				IdentityProviders: []*keycloakv1alpha1.KeycloakIdentityProvider{
					{
						Alias:                     "github",
						TrustEmail:                true,
						ProviderID:                "github",
						FirstBrokerLoginFlowAlias: "first broker login",
						Config: map[string]string{
							"clientId":     githubClientID,
							"clientSecret": githubClientSecret,
							"useJwksUrl":   "true",
						},
					},
				},
			},
			InstanceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "sso",
					// "realm": authzDomain.Name,
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(authzDomain, realm, r.Scheme); err != nil {
		return nil, err
	}
	return realm, nil
}
