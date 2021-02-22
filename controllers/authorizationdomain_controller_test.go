package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	keycloakv1alpha1 "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	multiclusterkeycloakv1alpha1 "github.com/mdelder/multicluster-keycloak-operator/api/v1alpha1"
	ocmclusterv1 "github.com/open-cluster-management/api/cluster/v1"
	ocmworkv1 "github.com/open-cluster-management/api/work/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
)

var _ = Describe("AuthorizationDomain Controller", func() {
	const timeout = time.Second * 30
	const interval = time.Second * 1
	var clusterName string

	BeforeEach(func() {
		clusterName = fmt.Sprintf("managedcluster-%s", rand.String(6))
		prepareCluster(clusterName, timeout, interval)
	})

	Context("Deploy an authorization domain", func() {
		It("Should have a authorization domain deployed correctly", func() {
			authDomainName := fmt.Sprintf("test-%s", rand.String(6))
			prepareSecret(authDomainName)
			prepareAuthDomain(authDomainName, authDomainName)

			// Check if cloakrealm is created
			Eventually(func() error {
				cloakRealm := &keycloakv1alpha1.KeycloakRealm{}
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: authDomainName, Namespace: keycloakNS}, cloakRealm)
				if err != nil {
					return err
				}
				return nil
			}, timeout, interval).Should(Succeed())
			// Check if cloakclient is created
			Eventually(func() error {
				cloakClient := &keycloakv1alpha1.KeycloakClient{}
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("%s-%s", authDomainName, clusterName), Namespace: keycloakNS}, cloakClient)
				if err != nil {
					return err
				}
				return nil
			}, timeout, interval).Should(Succeed())
			// Check if manifestwork is created
			Eventually(func() error {
				work := &ocmworkv1.ManifestWork{}
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("%s-%s-oauth", authDomainName, clusterName), Namespace: clusterName}, work)
				if err != nil {
					return err
				}
				return nil
			}, timeout, interval).Should(Succeed())
		})
	})
})

func prepareCluster(clusterName string, timeout, interval time.Duration) {
	cluster := &ocmclusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
	}
	err := k8sClient.Create(context.Background(), cluster)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		err = k8sClient.Get(context.Background(), types.NamespacedName{Name: clusterName}, cluster)
		if err != nil {
			return err
		}
		return nil
	}, timeout, interval).Should(Succeed())

	cluster.Status = ocmclusterv1.ManagedClusterStatus{
		ClusterClaims: []ocmclusterv1.ManagedClusterClaim{
			{
				Name:  "consoleurl.cluster.open-cluster-management.io",
				Value: "https://test",
			},
			{
				Name:  "id.openshift.io",
				Value: clusterName,
			},
		},
		Conditions: []metav1.Condition{},
	}
	err = k8sClient.Status().Update(context.Background(), cluster)
	Expect(err).ToNot(HaveOccurred())

	clusterNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
	}
	err = k8sClient.Create(context.Background(), clusterNs)
	Expect(err).ToNot(HaveOccurred())
}

func prepareAuthDomain(authDomainName, secretName string) {
	authDomain := &multiclusterkeycloakv1alpha1.AuthorizationDomain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      authDomainName,
			Namespace: keycloakNS,
		},
		Spec: multiclusterkeycloakv1alpha1.AuthorizationDomainSpec{
			IdentityProviders: []multiclusterkeycloakv1alpha1.IdentityProvider{
				{
					Type:      "github",
					SecretRef: secretName,
				},
			},
			IssuerURL: "https://test",
		},
	}
	err := k8sClient.Create(context.Background(), authDomain)
	Expect(err).ToNot(HaveOccurred())
}

func prepareSecret(secretName string) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: keycloakNS,
		},
		Data: map[string][]byte{
			"clientId":     []byte("clientid"),
			"clientSecret": []byte("clientsecret"),
		},
	}
	err := k8sClient.Create(context.Background(), secret)
	Expect(err).ToNot(HaveOccurred())
}
