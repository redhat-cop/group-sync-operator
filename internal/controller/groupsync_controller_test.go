package controller

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	// "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
				resource := &redhatcopv1alpha1.GroupSync{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: namespace,
					},
					Spec: redhatcopv1alpha1.GroupSyncSpec{
						// Define your spec here
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		It("should contain the finalizer", func() {
			// Reconcile logic is triggered by the controller manager running in BeforeSuite
			// or by calling the reconciler directly.

			gs := &redhatcopv1alpha1.GroupSync{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, gs)).To(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, gs)
				if err != nil {
					return false
				}

				fmt.Printf("%+v\n", gs)
				labels := gs.GetLabels()
				_, ok := labels[finalizer]
				fmt.Printf("%v", labels)
				return ok
			}, 10*time.Second, time.Second).Should(BeTrue())
		})
	})
})
