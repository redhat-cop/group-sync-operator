package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/api/v1alpha1"
	"github.com/redhat-cop/group-sync-operator/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	// "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	userv1 "github.com/openshift/api/user/v1"
)

var _ = Describe("MyController", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()
		namespace := "group-sync-operator"

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: namespace,
		}

		testProviderName := "test-provider"

		BeforeEach(func() {
			// Create namespace for isolation
			ns := &corev1.Namespace{}
			ns.Name = "group-sync-operator"
			err := k8sClient.Create(ctx, ns)
			Expect(err).NotTo(HaveOccurred())

			// Create the Custom Resource
			resource := &redhatcopv1alpha1.GroupSync{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			if err != nil {
				resource.SetName(resourceName)
				resource.SetNamespace(namespace)
				resource.Spec.Providers = []redhatcopv1alpha1.Provider{
					{
						Name: testProviderName,
						ProviderType: &redhatcopv1alpha1.ProviderType{
							Azure: &redhatcopv1alpha1.AzureProvider{},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		It("should contain the finalizer and disown a group when deleted", func() {
			// Reconcile logic is triggered by the controller manager running in BeforeSuite
			// or by calling the reconciler directly.

			gs := &redhatcopv1alpha1.GroupSync{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, gs)).To(Succeed())

			ocpGroup := &userv1.Group{}
			ocpGroup.SetName("group")
			ocpGroup.SetLabels(map[string]string{
				constants.SyncProvider: getProviderLabel(gs, testProviderName),
			})
			ocpGroup.Users = []string{}

			Expect(k8sClient.Create(ctx, ocpGroup)).To(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, gs)
				if err != nil {
					return false
				}

				return controllerutil.ContainsFinalizer(gs, finalizer)
			}, 10*time.Second, time.Second).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, gs)).To(Succeed())

			Eventually(func() bool {
				group := &userv1.Group{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: ocpGroup.Name}, group)
				Expect(err).NotTo(HaveOccurred())
				_, ok := group.GetLabels()[constants.SyncProvider]
				return !ok
			}, 10*time.Second, time.Second)

		})
	})
})
