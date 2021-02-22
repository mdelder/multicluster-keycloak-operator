package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	keycloakv1alpha1 "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	ocmworkv1 "github.com/open-cluster-management/api/work/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
)

var _ = Describe("ManagedCluster Controller", func() {
	const timeout = time.Second * 30
	const interval = time.Second * 1
	const keycloakNS = "keycloak"
	var authDomainName string

	BeforeEach(func() {
		authDomainName = fmt.Sprintf("test-%s", rand.String(6))
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
	})

	Context("Add a managedcluster", func() {
		It("Should cluster setup correctly", func() {
			clusterName := fmt.Sprintf("managedcluster-%s", rand.String(6))
			prepareCluster(clusterName, timeout, interval)

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
