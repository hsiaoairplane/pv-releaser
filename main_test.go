package main

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubernetes/pkg/controller/volume/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcilePVReleaseScenarios(t *testing.T) {
	tests := []struct {
		name                  string
		initialClaimRefName   string
		expectUIDCleared      bool
		expectResourceCleared bool
	}{
		{
			name:                  "release pv when claimref matches pvc",
			initialClaimRefName:   "test-pvc",
			expectUIDCleared:      true,
			expectResourceCleared: true,
		},
		{
			name:                  "do not release pv when claimref does not match pvc",
			initialClaimRefName:   "test-pvc-new",
			expectUIDCleared:      false,
			expectResourceCleared: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runReconcileTest(
				t,
				tt.initialClaimRefName,
				tt.expectUIDCleared,
				tt.expectResourceCleared,
			)
		})
	}
}

func runReconcileTest(
	t *testing.T,
	claimRefName string,
	expectUIDCleared bool,
	expectResourceCleared bool,
) {
	t.Helper()

	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)

	ctx := context.Background()

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
		},
		Status: corev1.PersistentVolumeClaimStatus{
			Phase: corev1.ClaimPending,
		},
	}

	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc-failedbinding",
			Namespace: "default",
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:      "PersistentVolumeClaim",
			Namespace: pvc.Namespace,
			Name:      pvc.Name,
			UID:       pvc.UID,
		},
		Reason:  events.FailedBinding,
		Message: "volume already bound to a different claim",
		Type:    corev1.EventTypeWarning,
	}

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
				Name:            claimRefName,
				Namespace:       "default",
				UID:             types.UID("foobar"),
				ResourceVersion: "12345",
			},
		},
		Status: corev1.PersistentVolumeStatus{
			Phase: corev1.VolumeReleased,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pvc, pv, event).
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

	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}

	var updatedPV corev1.PersistentVolume
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: pv.Name}, &updatedPV); err != nil {
		t.Fatalf("failed to get updated pv: %v", err)
	}

	if expectUIDCleared && updatedPV.Spec.ClaimRef.UID != "" {
		t.Errorf("expected ClaimRef.UID to be cleared")
	}
	if !expectUIDCleared && updatedPV.Spec.ClaimRef.UID == "" {
		t.Errorf("expected ClaimRef.UID to not be cleared")
	}

	if expectResourceCleared && updatedPV.Spec.ClaimRef.ResourceVersion != "" {
		t.Errorf("expected ClaimRef.ResourceVersion to be cleared")
	}
	if !expectResourceCleared && updatedPV.Spec.ClaimRef.ResourceVersion == "" {
		t.Errorf("expected ClaimRef.ResourceVersion to not be cleared")
	}
}
