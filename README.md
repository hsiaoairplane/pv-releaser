# Static PV Releaser

A Kubernetes custom controller that automatically releases retained static PersistentVolumes when a new PersistentVolumeClaim fails to bind due to an existing claimRef.

## Problem

In Kubernetes clusters using static / manually managed PersistentVolumes, it is common to hit the following situation:
- A PVC is deleted and later recreated
- The underlying static PV uses:
  ```yaml
  persistentVolumeReclaimPolicy: Retain
  ```

- The static PV transitions to Released
- A new PVC tries to bind but fails with:
  ```bash
  volume already bound to a different claim
  ```


This leaves the PVC stuck in Pending, requiring manual static PV editing.

## What static-pv-releaser Does

`static-pv-releaser` automatically resolves this situation by:
1. Watching PVC Create and Update events only
1. Detecting PVCs stuck in Pending due to a bind conflict
1. Finding PVs that:
   - Have persistentVolumeReclaimPolicy: Retain
   - Are in Released phase
   - Are still bound to a different PVC
1. Clearing the stale fields:
   ```yaml
   spec:
     claimRef:
       name: foo-pvc
       namespace: foo
       uid: ""
       resourceVersion: ""
   ```
1. Letting Kubernetes re-bind the static PV to the new PVC
