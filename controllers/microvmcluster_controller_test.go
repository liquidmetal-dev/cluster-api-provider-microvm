// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0.

package controllers_test

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"

	infrav1 "github.com/weaveworks/cluster-api-provider-microvm/api/v1alpha1"
)

func TestClusterReconciliationNoEndpoint(t *testing.T) {
	g := NewWithT(t)

	objects := []runtime.Object{
		createCluster(testClusterName, testClusterNamespace),
		createMicrovmCluster(testClusterName, testClusterNamespace),
	}

	client := createFakeClient(g, objects)
	result, err := reconcileCluster(client)

	g.Expect(err).To(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(Equal(time.Duration(0)))

	reconciled, err := getMicrovmCluster(context.TODO(), client, testClusterName, testClusterNamespace)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(reconciled.Status.Ready).To(BeFalse())

	c := conditions.Get(reconciled, infrav1.LoadBalancerAvailableCondition)
	g.Expect(c).To(BeNil())
}

func TestClusterReconciliationWithMvmClusterEndpoint(t *testing.T) {
	g := NewWithT(t)

	mvmCluster := createMicrovmCluster(testClusterName, testClusterNamespace)
	mvmCluster.Spec.EndpointRef = &corev1.ObjectReference{
		Kind: "ExternalLoadBalancerEndpoint",
		Name: "tenant1-elb-endpoint",
	}

	endpoint := &infrav1.ExternalLoadBalancer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tenant1-elb-endpoint",
			Namespace: "ns1",
		},
		Spec: infrav1.ExternalLoadBalancerSpec{
			Endpoint: infrav1.ExternalLoadBalancerEndpoint{
				Host: "localhost",
				Port: 6443,
			},
		},
		Status: infrav1.ExternalLoadBalancerStatus{
			Ready: true,
		},
	}

	objects := []runtime.Object{
		createCluster(testClusterName, testClusterNamespace),
		mvmCluster,
		endpoint,
	}

	client := createFakeClient(g, objects)
	result, err := reconcileCluster(client)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(Equal(time.Duration(0)))

	reconciled, err := getMicrovmCluster(context.TODO(), client, testClusterName, testClusterNamespace)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(reconciled.Status.Ready).To(BeTrue())
	g.Expect(reconciled.Status.FailureDomains).To(HaveLen(1))

	c := conditions.Get(reconciled, infrav1.LoadBalancerAvailableCondition)
	g.Expect(c).ToNot(BeNil())
	g.Expect(c.Status).To(Equal(corev1.ConditionTrue))

	c = conditions.Get(reconciled, clusterv1.ReadyCondition)
	g.Expect(c).ToNot(BeNil())
	g.Expect(c.Status).To(Equal(corev1.ConditionTrue))
}

func TestClusterReconciliationWithClusterEndpointAPIServerNotReady(t *testing.T) {
	g := NewWithT(t)

	cluster := createCluster(testClusterName, testClusterNamespace)
	mvmCluster := createMicrovmCluster(testClusterName, testClusterNamespace)
	mvmCluster.Spec.EndpointRef = &corev1.ObjectReference{
		Kind: "ExternalLoadBalancerEndpoint",
		Name: "tenant1-elb-endpoint",
	}

	endpoint := &infrav1.ExternalLoadBalancer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tenant1-elb-endpoint",
			Namespace: "ns1",
		},
		Spec: infrav1.ExternalLoadBalancerSpec{
			Endpoint: infrav1.ExternalLoadBalancerEndpoint{
				Host: "localhost",
				Port: 6443,
			},
		},
		Status: infrav1.ExternalLoadBalancerStatus{
			Ready: false,
		},
	}

	objects := []runtime.Object{
		cluster,
		mvmCluster,
		endpoint,
	}

	client := createFakeClient(g, objects)
	result, err := reconcileCluster(client)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(Equal(time.Duration(30 * time.Second)))

	reconciled, err := getMicrovmCluster(context.TODO(), client, testClusterName, testClusterNamespace)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(reconciled.Status.Ready).To(BeTrue())
	g.Expect(reconciled.Status.FailureDomains).To(HaveLen(1))

	c := conditions.Get(reconciled, infrav1.LoadBalancerAvailableCondition)
	g.Expect(c).ToNot(BeNil())
	g.Expect(c.Status).To(Equal(corev1.ConditionFalse))

	c = conditions.Get(reconciled, clusterv1.ReadyCondition)
	g.Expect(c).ToNot(BeNil())
	g.Expect(c.Status).To(Equal(corev1.ConditionFalse))
}

func TestClusterReconciliationMicrovmAlreadyDeleted(t *testing.T) {
	g := NewWithT(t)

	objects := []runtime.Object{}

	client := createFakeClient(g, objects)
	result, err := reconcileCluster(client)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(Equal(time.Duration(0)))

	_, err = getMicrovmCluster(context.TODO(), client, testClusterName, testClusterNamespace)
	g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
}

func TestClusterReconciliationNotOwner(t *testing.T) {
	g := NewWithT(t)

	mvmCluster := createMicrovmCluster(testClusterName, testClusterNamespace)
	mvmCluster.ObjectMeta.OwnerReferences = nil

	objects := []runtime.Object{
		createCluster(testClusterName, testClusterNamespace),
		mvmCluster,
	}

	client := createFakeClient(g, objects)
	result, err := reconcileCluster(client)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(Equal(time.Duration(0)))

	reconciled, err := getMicrovmCluster(context.TODO(), client, testClusterName, testClusterNamespace)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(reconciled.Status.Ready).To(BeFalse())

	c := conditions.Get(reconciled, infrav1.LoadBalancerAvailableCondition)
	g.Expect(c).To(BeNil())
}

func TestClusterReconciliationWhenPaused(t *testing.T) {
	g := NewWithT(t)

	mvmCluster := createMicrovmCluster(testClusterName, testClusterNamespace)
	mvmCluster.ObjectMeta.Annotations = map[string]string{
		clusterv1.PausedAnnotation: "true",
	}

	objects := []runtime.Object{
		createCluster(testClusterName, testClusterNamespace),
		mvmCluster,
	}

	client := createFakeClient(g, objects)
	result, err := reconcileCluster(client)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(Equal(time.Duration(0)))

	reconciled, err := getMicrovmCluster(context.TODO(), client, testClusterName, testClusterNamespace)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(reconciled.Status.Ready).To(BeFalse())

	c := conditions.Get(reconciled, infrav1.LoadBalancerAvailableCondition)
	g.Expect(c).To(BeNil())
}

func TestClusterReconciliationDelete(t *testing.T) {
	g := NewWithT(t)

	mvmCluster := createMicrovmCluster(testClusterName, testClusterNamespace)
	mvmCluster.ObjectMeta.DeletionTimestamp = &metav1.Time{
		Time: time.Now(),
	}

	objects := []runtime.Object{
		createCluster(testClusterName, testClusterNamespace),
		mvmCluster,
	}

	client := createFakeClient(g, objects)
	result, err := reconcileCluster(client)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(Equal(time.Duration(0)))

	// TODO: when we move to envtest this should return an NotFound error. #30
	_, err = getMicrovmCluster(context.TODO(), client, testClusterName, testClusterNamespace)
	g.Expect(err).NotTo(HaveOccurred())
}
