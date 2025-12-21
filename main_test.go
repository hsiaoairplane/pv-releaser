package main

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcile_ReleasesConflictingPV(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)

	ctx := context.Background()

	// --- PVC with binding conflict ---
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
		},
		Status: corev1.PersistentVolumeClaimStatus{
			Phase: corev1.ClaimPending,
			Conditions: []corev1.PersistentVolumeClaimCondition{
				{
					Type:    corev1.PersistentVolumeClaimConditionType("Bound"),
					Message: "volume already bound to a different claim",
				},
			},
		},
	}

	// --- Released PV bound to another PVC ---
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv",
		},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				NFS: &corev1.NFSVolumeSource{
					Server: "10.0.0.1",
					Path:   "/data",
				},
			},
			ClaimRef: &corev1.ObjectReference{
				Name:            "old-pvc",
				Namespace:       "default",
				UID:             types.UID("old-uid"),
				ResourceVersion: "12345",
			},
		},
		Status: corev1.PersistentVolumeStatus{
			Phase: corev1.VolumeReleased,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pvc, pv).
		Build()

	reconciler := &PVCReclaimerReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      pvc.Name,
			Namespace: pvc.Namespace,
		},
	}

	// --- Run reconcile ---
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}

	// --- Expect requeue ---
	if !result.Requeue {
		t.Fatalf("expected reconcile to requeue")
	}

	// --- Fetch updated PV ---
	var updatedPV corev1.PersistentVolume
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: pv.Name}, &updatedPV); err != nil {
		t.Fatalf("failed to get updated pv: %v", err)
	}

	// --- Assert ClaimRef fields cleared ---
	if updatedPV.Spec.ClaimRef.UID != "" {
		t.Errorf("expected ClaimRef.UID to be cleared")
	}
	if updatedPV.Spec.ClaimRef.ResourceVersion != "" {
		t.Errorf("expected ClaimRef.ResourceVersion to be cleared")
	}
}
