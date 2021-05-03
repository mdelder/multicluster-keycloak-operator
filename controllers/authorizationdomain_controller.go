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
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	ocmclusterv1 "github.com/open-cluster-management/api/cluster/v1"
	ocmworkv1 "github.com/open-cluster-management/api/work/v1"

	// openshiftconfigv1 "github.com/openshift/api/config/v1"

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

type managedClusterSSOContext struct {
	AuthorizationDomain   *multiclusterkeycloakv1alpha1.AuthorizationDomain
	ManagedCluster        *ocmclusterv1.ManagedCluster
	KeycloakClient        *keycloakv1alpha1.KeycloakClient
	RootURL               string
	ClientID              string
	ClientSecret          string
	OAuthConfigSecretName string
	BaseDomain            string
}

func (r *AuthorizationDomainReconciler) newManagedClusterSSOContext(authzDomain *multiclusterkeycloakv1alpha1.AuthorizationDomain, cluster *ocmclusterv1.ManagedCluster, defaultBaseDomain string) (*managedClusterSSOContext, error) {

	if authzDomain == nil {
		return nil, errors.NewBadRequest("AuthorizationDomain may not be nil.")
	}
	if cluster == nil {
		return nil, errors.NewBadRequest("ManagedCluster may not be nil.")
	}
	c := &managedClusterSSOContext{
		AuthorizationDomain:   authzDomain,
		ManagedCluster:        cluster,
		ClientID:              fmt.Sprintf("%s-%s", authzDomain.Name, cluster.Name),
		OAuthConfigSecretName: fmt.Sprintf("%s-%s-oauth-credentials", cluster.Name, authzDomain.Name),
	}
	c.BaseDomain = defaultBaseDomain
	if c.ManagedCluster.Status.ClusterClaims != nil {
		for _, claim := range c.ManagedCluster.Status.ClusterClaims {
			if claim.Name == "consoleurl.cluster.open-cluster-management.io" {
				c.RootURL = strings.Replace(claim.Value, "console-openshift-console", "oauth-openshift", 1)
				// derive updated c.BaseDomain = from RootURL
			} else if claim.Name == "id.openshift.io" {
				c.ClientSecret = claim.Value
			}
		}
	}
	if c.ClientSecret == "" {
		return nil, errors.NewNotFound(schema.GroupResource{Group: "cluster.open-cluster-management.io", Resource: "ManagedClusterClaim"}, "ManagedClusterClaim for \"id.openshift.io\" not found. Attempt to reconcile deferred.")
	}
	if c.RootURL == "" {
		return nil, errors.NewNotFound(schema.GroupResource{Group: "cluster.open-cluster-management.io", Resource: "ManagedClusterClaim"}, "ManagedClusterClaim for \"consoleurl.cluster.open-cluster-management.io\" not found. Attempt to reconcile deferred.")
	}
	return c, nil
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
		return r.Delete(ctx, authzDomain)

	}

	// Create the Realm
	r.Log.Info("Creating KeycloakRealm", "AuthorizationDomain.Name", authzDomain.Name, "AuthorizationDomain.Namespace", authzDomain.Namespace)
	found := &keycloakv1alpha1.KeycloakRealm{}
	realm, err := r.createKeycloakRealm(authzDomain)
	if err != nil {
		r.Log.Info("Failed to create KeycloakRealm", "KeycloakRealm", realm)
		return ctrl.Result{}, err
	}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: authzDomain.Name, Namespace: authzDomain.Namespace}, found); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("Creating KeycloakRealm", "KeycloakRealm", realm)
			if err := r.Client.Create(context.TODO(), realm); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	} else if !reflect.DeepEqual(found.Spec, realm.Spec) {
		realm.Spec.DeepCopyInto(&found.Spec)
		r.Log.Info("Updating KeycloakRealm", "KeycloakRealm", found)
		r.Client.Update(context.TODO(), found)
	}

	baseDomain := "demo.red-chesterfield.com" // temporary

	r.Log.Info("Retrieving ManagedClusters attached to the ClusterManager", "AuthorizationDomain.Name", authzDomain.Name, "AuthorizationDomain.Namespace", authzDomain.Namespace)
	// For each ManagedCluster, create the KeycloakClient, ManifestWork
	managedClusterList := &ocmclusterv1.ManagedClusterList{}
	err = r.Client.List(context.Background(), managedClusterList)
	errs := []error{}
	for _, cluster := range managedClusterList.Items {
		r.Log.Info("Discovered ManagedCluster", "ManagedClusterName", cluster.Name)
		clusterContext, err := r.newManagedClusterSSOContext(authzDomain, &cluster, baseDomain)
		if err != nil {
			errs = append(errs, err)
		} else {
			if err := r.createOrUpdateKeycloakClient(clusterContext); err != nil {
				return ctrl.Result{}, err
			}
			if err := r.createOrUpdateManifestWork(clusterContext); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	if len(errs) > 0 {
		return ctrl.Result{}, errs[0]
	}
	return ctrl.Result{}, nil
}

func (r *AuthorizationDomainReconciler) Delete(ctx context.Context, authzDomain *multiclusterkeycloakv1alpha1.AuthorizationDomain) (ctrl.Result, error) {
	r.Log.Error(errors.NewBadRequest("WARNING: No specialized delete process is currently implemented to clean up created resources."), "AuthorizationDomain.Namespace", authzDomain.Namespace, "AuthorizationDomain.Name", authzDomain.Name)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AuthorizationDomainReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multiclusterkeycloakv1alpha1.AuthorizationDomain{}).
		// watch the Secret / ConfigMap referenced by AuthorizationDomain
		Watches(&source.Kind{Type: &ocmclusterv1.ManagedCluster{}}, &handler.EnqueueRequestForObject{}).
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
				if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: provider.SecretRef, Namespace: authzDomain.Namespace}, secret); err != nil {
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
			Namespace: authzDomain.Namespace,
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

func (r *AuthorizationDomainReconciler) createManifestWork(clusterContext *managedClusterSSOContext) (*ocmworkv1.ManifestWork, error) {

	oauthCredentialsManifest, err := clusterContext.createOAuthSecretManifest()
	if err != nil {
		r.Log.Error(err, "Could create Secret for OAuth client.")
		return nil, err
	}
	oauthClientManifest, err := clusterContext.createOAuthClientManifest()
	if err != nil {
		r.Log.Error(err, "Could create OAuth cluster.")
		return nil, err
	}
	oauthClusterRole, err := clusterContext.createClusterRole()
	if err != nil {
		r.Log.Error(err, "Could create OAuth cluster.")
		return nil, err
	}
	oauthClusterRoleBinding, err := clusterContext.createClusterRoleBinding()
	if err != nil {
		r.Log.Error(err, "Could create OAuth cluster.")
		return nil, err
	}

	issuerCertificateConfigMap, err := clusterContext.createIssuerCertificateConfigMap(r.Client)
	if err != nil || issuerCertificateConfigMap == nil {
		r.Log.Error(err, "Could not find IssuerCertificate.", "issuerCerficate.configMapRef", clusterContext.AuthorizationDomain.Spec.IssuerCertificate.ConfigMapRef)
		return nil, err
	}
	r.Log.Info("Found IssuerCertificate ConfigMap", "IssuerCertificateName", clusterContext.AuthorizationDomain.Spec.IssuerCertificate.ConfigMapRef)
	manifestwork := &ocmworkv1.ManifestWork{
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-oauth", clusterContext.ClientID), Namespace: clusterContext.ManagedCluster.Name, Labels: map[string]string{"app": "sso"}},
		Spec: ocmworkv1.ManifestWorkSpec{
			Workload: ocmworkv1.ManifestsTemplate{
				Manifests: []ocmworkv1.Manifest{
					*oauthClusterRole,
					*oauthClusterRoleBinding,
					*oauthClientManifest,
					*oauthCredentialsManifest,
					*issuerCertificateConfigMap,
				},
			},
		},
	}
	return manifestwork, nil
}

func (r *AuthorizationDomainReconciler) createOrUpdateKeycloakClient(clusterContext *managedClusterSSOContext) error {
	found := &keycloakv1alpha1.KeycloakClient{}
	keycloakClient, err := r.createKeycloakClient(clusterContext)
	if err != nil {
		return err
	}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: clusterContext.ClientID, Namespace: clusterContext.AuthorizationDomain.Namespace}, found); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("Creating KeycloakClient", "KeycloakClient", keycloakClient)
			if err := r.Client.Create(context.TODO(), keycloakClient); err != nil {
				return err
			}
		}
		return err
	} else if !reflect.DeepEqual(found.Spec, keycloakClient.Spec) {
		keycloakClient.Spec.DeepCopyInto(&found.Spec)
		err := r.Client.Update(context.TODO(), found)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *AuthorizationDomainReconciler) createKeycloakClient(clusterContext *managedClusterSSOContext) (*keycloakv1alpha1.KeycloakClient, error) {
	client := &keycloakv1alpha1.KeycloakClient{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: clusterContext.ClientID, Namespace: clusterContext.AuthorizationDomain.Namespace, Labels: map[string]string{
			"app": "sso",
			// "realm": authzDomain.Name
		}},
		Spec: keycloakv1alpha1.KeycloakClientSpec{
			RealmSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "sso",
					// "realm": authzDomain.Name,
				},
			},
			Client: &keycloakv1alpha1.KeycloakAPIClient{
				ClientAuthenticatorType: "client-secret",
				Name:                    clusterContext.ClientID,
				ClientID:                clusterContext.ClientID,
				Enabled:                 true,
				Secret:                  clusterContext.ClientSecret,
				BaseURL:                 "/oauth2callback/oidcidp",
				RootURL:                 clusterContext.RootURL,
				Description:             "Managed Multicluster Keycloak Client",
				// DefaultRoles:              []string{},
				RedirectUris:        []string{"/oauth2callback/oidcidp"},
				StandardFlowEnabled: true,
			},
		},
	}
	if err := controllerutil.SetControllerReference(clusterContext.AuthorizationDomain, client, r.Scheme); err != nil {
		return nil, err
	}
	return client, nil
}
func (r *AuthorizationDomainReconciler) createOrUpdateManifestWork(clusterContext *managedClusterSSOContext) error {
	manifestWork := &ocmworkv1.ManifestWork{}
	manifestWorkName := fmt.Sprintf("%s-oauth", clusterContext.ClientID)
	r.Log.Info("Looking for existing ManifestWork", "ManifestWorkName", manifestWorkName)
	manifestWork, err := r.createManifestWork(clusterContext)
	if err != nil {
		return err
	}
	found := &ocmworkv1.ManifestWork{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: manifestWorkName, Namespace: clusterContext.ManagedCluster.Name}, found); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("Creating ManifestWork", "ManifestWork", manifestWork)
			if err := r.Client.Create(context.TODO(), manifestWork); err != nil {
				return err
			}
			// Cross Namespace ownerRefs are BAD
			// if err := controllerutil.SetControllerReference(authzDomain, manifestWork, r.Scheme); err != nil {
			// 	r.Log.Error(err, "Could not set controller reference for KeycloakClient")
			// 	return ctrl.Result{}, err
			// }
		} else {
			r.Log.Error(err, "Could not fetch ManifestWork", "ManifestWorkName", manifestWorkName)
			return err
		}
	} else if !reflect.DeepEqual(found.Spec, manifestWork.Spec) {
		manifestWork.Spec.DeepCopyInto(&found.Spec)
		err := r.Client.Update(context.TODO(), found)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *managedClusterSSOContext) createOAuthSecretManifest() (*ocmworkv1.Manifest, error) {
	return c.createManifest(&corev1.Secret{
		TypeMeta:   metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: c.OAuthConfigSecretName, Namespace: "openshift-config", Labels: map[string]string{"app": "sso"}},
		Data: map[string][]byte{
			"clientID":     []byte(c.ClientID),
			"clientSecret": []byte(c.ClientSecret),
		},
		// StringData: map[string]string{},
		Type: "Opaque",
	})
}

func (c *managedClusterSSOContext) createOAuthClientManifest() (*ocmworkv1.Manifest, error) {
	oauthClientJSON := []byte(fmt.Sprintf(`
	{
		"apiVersion": "config.openshift.io/v1",
		"kind": "OAuth",
		"metadata": {
		   "name": "cluster"
		},
		"spec": {
		   "identityProviders": [
			  {
				 "name": "oidcidp",
				 "mappingMethod": "claim",
				 "type": "OpenID",
				 "openID": {
					"clientID": "%s",
					"clientSecret": {
					   "name": "%s"
					},
					"ca": {
					   "name": "ca-config-map"
					},
					"claims": {
					   "preferredUsername": [
						  "email"
					   ],
					   "name": [
						  "name"
					   ],
					   "email": [
						  "email"
					   ]
					},
					"issuer": "%s"
				 }
			  }
		   ]
		}
	 }
	`, c.ClientID, c.OAuthConfigSecretName, c.AuthorizationDomain.Spec.IssuerURL))
	oauthClientManifest := &ocmworkv1.Manifest{}
	oauthClientManifest.RawExtension = runtime.RawExtension{Raw: oauthClientJSON}
	return oauthClientManifest, nil
}

func (c *managedClusterSSOContext) createClusterRole() (*ocmworkv1.Manifest, error) {
	return c.createManifest(&rbacv1.ClusterRole{
		TypeMeta:   metav1.TypeMeta{Kind: "ClusterRole", APIVersion: "rbac.authorization.k8s.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "open-cluster-management:klusterlet-work-sa:agent:oauth-edit"},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"config.openshift.io"},
				Resources: []string{"oauths"},
				Verbs:     []string{"create", "get", "list", "update", "delete"},
			},
		},
		// AggregationRule: &rbacv1.AggregationRule{ClusterRoleSelectors: []metav1.LabelSelector{}},
	})
}

func (c *managedClusterSSOContext) createClusterRoleBinding() (*ocmworkv1.Manifest, error) {
	return c.createManifest(&rbacv1.ClusterRoleBinding{
		TypeMeta:   metav1.TypeMeta{Kind: "ClusterRoleBinding", APIVersion: "rbac.authorization.k8s.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "open-cluster-management:klusterlet-work-sa:agent:oauth-edit"},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: "klusterlet-work-sa", Namespace: "open-cluster-management-agent"}},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "open-cluster-management:klusterlet-work-sa:agent:oauth-edit"},
	})
}

func (c *managedClusterSSOContext) createIssuerCertificateConfigMap(client client.Client) (*ocmworkv1.Manifest, error) {
	configMap := &corev1.ConfigMap{}
	if c.AuthorizationDomain.Spec.IssuerCertificate.ConfigMapRef != "" {
		err := client.Get(context.TODO(), types.NamespacedName{Name: c.AuthorizationDomain.Spec.IssuerCertificate.ConfigMapRef, Namespace: c.AuthorizationDomain.Namespace}, configMap)
		if err != nil {
			return nil, err
		}
		// The IssuerCertificate should be created in the "openshift-config" namespace
		configMap.ObjectMeta.Namespace = "openshift-config"
		configMap.ObjectMeta.ResourceVersion = ""
		configMap.ObjectMeta.SelfLink = ""
		configMap.ObjectMeta.UID = ""
		configMap.ObjectMeta.ManagedFields = []metav1.ManagedFieldsEntry{}
		return c.createManifest(configMap)
	}
	return nil, nil
}

func (c *managedClusterSSOContext) createManifest(obj interface{}) (*ocmworkv1.Manifest, error) {
	objJSON, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	manifest := &ocmworkv1.Manifest{}
	manifest.RawExtension = runtime.RawExtension{Raw: objJSON}
	return manifest, nil
}
