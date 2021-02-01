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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ocmclusterv1 "github.com/open-cluster-management/api/cluster/v1"
	ocmworkv1 "github.com/open-cluster-management/api/work/v1"

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

	isAuthzDomainMarkedToBeDeleted := authzDomain.GetDeletionTimestamp() != nil
	if isAuthzDomainMarkedToBeDeleted {
		r.Log.Info("AuthorizationDomain marked for deletion", "AuthorizationDomain", authzDomain)
	}

	// Create the Realm
	found := &keycloakv1alpha1.KeycloakRealm{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: authzDomain.ObjectMeta.Name, Namespace: "keycloak"}, found)
	if err != nil && errors.IsNotFound(err) {
		realm := r.createKeycloakRealm(authzDomain)
		r.Log.Info("Creating KeycloakRealm", "KeycloakRealm", realm)
		if err := r.Client.Create(context.TODO(), realm); err != nil {
			return ctrl.Result{}, err
		}
		if err := controllerutil.SetControllerReference(authzDomain, realm, r.Scheme); err != nil {
			r.Log.Error(err, "Could not set controller reference for KeycloakRealm")
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	} // else update if necessary

	// For each ManagedCluster, create the KeycloakClient

	baseDomain := "demo.red-chesterfield.com"
	managedClusterList := &ocmclusterv1.ManagedClusterList{}
	// clients := &keycloakv1alpha1.KeycloakClientList{}
	err = r.Client.List(context.Background(), managedClusterList)
	for _, cluster := range managedClusterList.Items {
		r.Log.Info("Discovered ManagedCluster", "ManagedClusterName", cluster.ObjectMeta.Name)

		keycloakClient := &keycloakv1alpha1.KeycloakClient{}
		keycloakClientName := fmt.Sprintf("%s-%s", authzDomain.ObjectMeta.Name, cluster.ObjectMeta.Name)
		if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: keycloakClientName, Namespace: "keycloak"}, keycloakClient); err != nil {
			if errors.IsNotFound(err) {
				keycloakClient = r.createKeycloakClient(authzDomain, &cluster, baseDomain)
				r.Log.Info("Creating KeycloakClient", "KeycloakClient", keycloakClient)
				if err := r.Client.Create(context.TODO(), keycloakClient); err != nil {
					return ctrl.Result{}, err
				}
				if err := controllerutil.SetControllerReference(authzDomain, keycloakClient, r.Scheme); err != nil {
					r.Log.Error(err, "Could not set controller reference for KeycloakClient")
					return ctrl.Result{}, err
				}
			}
			return ctrl.Result{}, err
		} // else update the KeycloakClient if needed

		manifestWork := &ocmworkv1.ManifestWork{}
		manifestWorkName := fmt.Sprintf("%s-%s-oauth", cluster.ObjectMeta.Name, authzDomain.ObjectMeta.Name)
		if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: manifestWorkName, Namespace: cluster.ObjectMeta.Name}, manifestWork); err != nil {
			if errors.IsNotFound(err) {
				manifestWork = r.createManifestWork(authzDomain, &cluster, baseDomain, keycloakClient)
				r.Log.Info("Creating ManifestWork", "ManifestWork", manifestWork)
				if err := r.Client.Create(context.TODO(), manifestWork); err != nil {
					return ctrl.Result{}, err
				}
				// Cross Namespace ownerRefs are BAD
				// if err := controllerutil.SetControllerReference(authzDomain, manifestWork, r.Scheme); err != nil {
				// 	r.Log.Error(err, "Could not set controller reference for KeycloakClient")
				// 	return ctrl.Result{}, err
				// }
			}
			return ctrl.Result{}, err
		} // else update the ManifestWork if needed

	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AuthorizationDomainReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multiclusterkeycloakv1alpha1.AuthorizationDomain{}).
		Owns(&keycloakv1alpha1.KeycloakRealm{}).
		Owns(&keycloakv1alpha1.KeycloakClient{}).
		Complete(r)
}

func (r *AuthorizationDomainReconciler) createKeycloakRealm(authzDomain *multiclusterkeycloakv1alpha1.AuthorizationDomain) *keycloakv1alpha1.KeycloakRealm {
	secret := &corev1.Secret{}
	githubClientID, githubClientSecret := "", ""
	if authzDomain.Spec.IdentityProviders != nil {
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
			Name:      authzDomain.ObjectMeta.Name,
			Namespace: "keycloak",
			Labels: map[string]string{
				"app": "sso",
				// "realm": authzDomain.ObjectMeta.Name,
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
					// "realm": authzDomain.ObjectMeta.Name,
				},
			},
		},
	}

	return realm
}

func (r *AuthorizationDomainReconciler) createKeycloakClient(authzDomain *multiclusterkeycloakv1alpha1.AuthorizationDomain, cluster *ocmclusterv1.ManagedCluster, baseDomain string) *keycloakv1alpha1.KeycloakClient {
	rootURL := fmt.Sprintf("https://oauth-openshift.apps.%s.%s", cluster.ObjectMeta.Name, baseDomain)
	clientID := fmt.Sprintf("%s-%s", authzDomain.ObjectMeta.Name, cluster.ObjectMeta.Name)
	clientSecret := "SAMPLE-CDE6024A-0225-4FF9-B04E-058E95A1095C"
	if cluster.Status.ClusterClaims != nil {
		for _, claim := range cluster.Status.ClusterClaims {
			if claim.Name == "consoleurl.cluster.open-cluster-management.io" {
				rootURL = strings.Replace(claim.Value, "console-openshift-console", "oauth-openshift", 1)
			} else if claim.Name == "id.openshift.io" {
				clientSecret = claim.Value
			}
		}
	}

	client := &keycloakv1alpha1.KeycloakClient{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: clientID, Namespace: "keycloak", Labels: map[string]string{
			"app": "sso",
			// "realm": authzDomain.ObjectMeta.Name
		}},
		Spec: keycloakv1alpha1.KeycloakClientSpec{
			RealmSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "sso",
					// "realm": authzDomain.ObjectMeta.Name,
				},
			},
			Client: &keycloakv1alpha1.KeycloakAPIClient{
				ClientID:    clientID,
				Enabled:     true,
				Secret:      clientSecret,
				BaseURL:     "/oauth2callback/oidcidp",
				RootURL:     rootURL,
				Description: "Managed Multicluster Keycloak Client",
				// DefaultRoles:              []string{},
				RedirectUris:        []string{"/oauth2callback/oidcidp"},
				StandardFlowEnabled: true,
			},
		},
	}
	return client
}

func (r *AuthorizationDomainReconciler) createManifestWork(authzDomain *multiclusterkeycloakv1alpha1.AuthorizationDomain, cluster *ocmclusterv1.ManagedCluster, baseDomain string, keycloakClient *keycloakv1alpha1.KeycloakClient) *ocmworkv1.ManifestWork {

	oauthCredentials := &corev1.Secret{
		TypeMeta:   metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-%s-oauth-credentials", cluster.Name, authzDomain.ObjectMeta.Name), Namespace: "openshift-config", Labels: map[string]string{"app": "sso"}},
		Data: map[string][]byte{
			"clientID":     []byte(keycloakClient.Spec.Client.ClientID),
			"clientSecret": []byte(keycloakClient.Spec.Client.Secret),
		},
		// StringData: map[string]string{},
		Type: "Opaque",
	}
	oauthCredentialsManifest := &ocmworkv1.Manifest{}
	oauthCredentialsJSON, err := json.Marshal(oauthCredentials)
	if err != nil {
		r.Log.Error(err, "Could create Secret for OAuth Credentails.")
		return nil
	}
	oauthCredentialsManifest.RawExtension = runtime.RawExtension{Raw: oauthCredentialsJSON}

	// oauthClusterManifest := ocmworkv1.Manifest{
	// 	runtime.RawExtension{Raw: []byte(oauthClusterJSON)},
	// }

	manifestwork := &ocmworkv1.ManifestWork{
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-%s-oauth", cluster.ObjectMeta.Name, authzDomain.ObjectMeta.Name), Namespace: cluster.ObjectMeta.Name, Labels: map[string]string{"app": "sso"}},
		Spec: ocmworkv1.ManifestWorkSpec{
			Workload: ocmworkv1.ManifestsTemplate{
				Manifests: []ocmworkv1.Manifest{
					// oauthClusterManifest,
					*oauthCredentialsManifest,
				},
			},
		},
	}

	return manifestwork
}
