package controllers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	keycloakv1alpha1 "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	multiclusterkeycloakv1alpha1 "github.com/mdelder/multicluster-keycloak-operator/api/v1alpha1"
	"github.com/mdelder/multicluster-keycloak-operator/controllers/bindata"
	ocmclusterv1 "github.com/open-cluster-management/api/cluster/v1"
	ocmworkv1 "github.com/open-cluster-management/api/work/v1"
	"github.com/openshift/library-go/pkg/assets"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type managedClusterSSOContext struct {
	client                  client.Client
	ManagedCluster          *ocmclusterv1.ManagedCluster
	IssuerCertificateConfig *corev1.ConfigMap
	RootURL                 string
	ClientID                string
	ClientSecret            string
	Issuer                  string
	EncodedClientID         string
	EncodedClientSecret     string
	OAuthConfigSecretName   string
	BaseDomain              string
}

var workFiles = []string{
	"controllers/manifests/clusterrole.yaml",
	"controllers/manifests/clusterrolebinding.yaml",
	"controllers/manifests/oauthclient.yaml",
	"controllers/manifests/secret.yaml",
}

func newManagedClusterSSOContext(
	client client.Client,
	authzDomain *multiclusterkeycloakv1alpha1.AuthorizationDomain,
	cluster *ocmclusterv1.ManagedCluster,
	defaultBaseDomain string,
) (*managedClusterSSOContext, error) {
	if authzDomain == nil {
		return nil, errors.NewBadRequest("AuthorizationDomain may not be nil.")
	}
	if cluster == nil {
		return nil, errors.NewBadRequest("ManagedCluster may not be nil.")
	}
	c := &managedClusterSSOContext{
		client:                client,
		ManagedCluster:        cluster,
		Issuer:                authzDomain.Spec.IssuerURL,
		ClientID:              fmt.Sprintf("%s-%s", authzDomain.Name, cluster.Name),
		OAuthConfigSecretName: fmt.Sprintf("%s-%s-oauth-credentials", cluster.Name, authzDomain.Name),
		BaseDomain:            defaultBaseDomain,
	}

	if authzDomain.Spec.IssuerCertificate.ConfigMapRef != "" {
		configMap := &corev1.ConfigMap{}
		err := client.Get(context.TODO(), types.NamespacedName{Name: authzDomain.Spec.IssuerCertificate.ConfigMapRef, Namespace: "keycloak"}, configMap)
		if err != nil {
			return nil, err
		}
		c.IssuerCertificateConfig = configMap
	}
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
	c.EncodedClientID = base64.StdEncoding.EncodeToString([]byte(c.ClientID))
	c.EncodedClientSecret = base64.StdEncoding.EncodeToString([]byte(c.ClientSecret))
	return c, nil
}

func (c *managedClusterSSOContext) manifestWork() (*ocmworkv1.ManifestWork, error) {
	work := &ocmworkv1.ManifestWork{
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-oauth", c.ClientID), Namespace: c.ManagedCluster.Name, Labels: map[string]string{"app": "sso"}},
		Spec: ocmworkv1.ManifestWorkSpec{
			Workload: ocmworkv1.ManifestsTemplate{
				Manifests: []ocmworkv1.Manifest{},
			},
		},
	}
	// Put manifests into work
	for _, file := range workFiles {
		rawData := assets.MustCreateAssetFromTemplate(file, bindata.MustAsset(filepath.Join("", file)), c).Data
		rawJSON, err := yaml.YAMLToJSON(rawData)
		if err != nil {
			return nil, err
		}
		work.Spec.Workload.Manifests = append(work.Spec.Workload.Manifests, ocmworkv1.Manifest{
			RawExtension: runtime.RawExtension{Raw: rawJSON},
		})
	}

	if c.IssuerCertificateConfig != nil {
		c.IssuerCertificateConfig.ObjectMeta = metav1.ObjectMeta{
			Name:      c.IssuerCertificateConfig.Name,
			Namespace: c.IssuerCertificateConfig.Namespace,
		}
		objJSON, err := json.Marshal(c.IssuerCertificateConfig)
		if err != nil {
			return nil, err
		}
		work.Spec.Workload.Manifests = append(work.Spec.Workload.Manifests, ocmworkv1.Manifest{
			RawExtension: runtime.RawExtension{Raw: objJSON},
		})

	}
	return work, nil
}

func (c *managedClusterSSOContext) keycloakClient() *keycloakv1alpha1.KeycloakClient {
	client := &keycloakv1alpha1.KeycloakClient{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: c.ClientID, Namespace: "keycloak", Labels: map[string]string{
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
				ClientID:    c.ClientID,
				Enabled:     true,
				Secret:      c.ClientSecret,
				BaseURL:     "/oauth2callback/oidcidp",
				RootURL:     c.RootURL,
				Description: "Managed Multicluster Keycloak Client",
				// DefaultRoles:              []string{},
				RedirectUris:        []string{"/oauth2callback/oidcidp"},
				StandardFlowEnabled: true,
			},
		},
	}

	return client
}

func (c *managedClusterSSOContext) createOrUpdateKeycloakClient() error {
	found := &keycloakv1alpha1.KeycloakClient{}
	keycloakClient := c.keycloakClient()
	err := c.client.Get(context.TODO(), types.NamespacedName{Name: c.ClientID, Namespace: "keycloak"}, found)
	switch {
	case errors.IsNotFound(err):
		return c.client.Create(context.TODO(), keycloakClient)
	case err != nil:
		return err
	}
	if !reflect.DeepEqual(found.Spec, keycloakClient.Spec) {
		keycloakClient.Spec.DeepCopyInto(&found.Spec)
		return c.client.Update(context.TODO(), found)
	}
	return nil
}

func (c *managedClusterSSOContext) createOrUpdateManifestWork() error {
	manifestWork, err := c.manifestWork()
	if err != nil {
		return err
	}
	found := &ocmworkv1.ManifestWork{}
	err = c.client.Get(context.TODO(), types.NamespacedName{Name: manifestWork.Name, Namespace: c.ManagedCluster.Name}, found)
	switch {
	case errors.IsNotFound(err):
		return c.client.Create(context.TODO(), manifestWork)
	case err != nil:
		return err
	}

	if !reflect.DeepEqual(found.Spec, manifestWork.Spec) {
		manifestWork.Spec.DeepCopyInto(&found.Spec)
		return c.client.Update(context.TODO(), found)
	}

	return nil
}
