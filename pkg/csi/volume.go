package csi

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

type PVWithStatus struct {
	Name      string
	Namespace string
	Kind      string
	Status    corev1.PersistentVolumeStatus
}

func (c *CSIHandler) handleGetJuiceFSPVOfApp(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c.log.Debugw("handleGetJuiceFSPVCOfApp", "argument", request.Params.Arguments)
	namespace, ok := request.Params.Arguments["namespace"].(string)
	if !ok {
		namespace = "default"
	}
	appName, ok := request.Params.Arguments["appName"].(string)
	if !ok {
		c.log.Errorw("Missing argument", "appName", appName)
		return nil, fmt.Errorf("missing appName")
	}

	var (
		pod   *corev1.Pod
		pvSts = []PVWithStatus{}
		err   error
	)
	if pod, err = c.client.CoreV1().Pods(namespace).Get(ctx, appName, metav1.GetOptions{}); err != nil {
		c.log.Errorw("Get Pod Error", "err", err)
		return nil, err
	}

	for _, volume := range pod.Spec.Volumes {
		var (
			pvc *corev1.PersistentVolumeClaim
			pv  *corev1.PersistentVolume
		)
		if volume.PersistentVolumeClaim == nil {
			continue
		}
		pvc, err = c.client.CoreV1().PersistentVolumeClaims(pod.Namespace).Get(ctx, volume.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return nil, err
		}
		if pvc.Status.Phase == corev1.ClaimBound {
			pv, err = c.client.CoreV1().PersistentVolumes().Get(ctx, pvc.Spec.VolumeName, metav1.GetOptions{})
			if err != nil && !k8serrors.IsNotFound(err) {
				return nil, err
			}
			if pv.Spec.CSI != nil && pv.Spec.CSI.Driver == DriverName {
				pvSts = append(pvSts, PVWithStatus{
					Name:      pv.Name,
					Namespace: pv.Namespace,
					Kind:      "PersistentVolume",
					Status:    pv.Status,
				})
			}
		}
	}

	c.log.Debugw("get pv", "pvs", pvSts)

	res, _ := json.Marshal(pvSts)
	return mcp.NewToolResultText(fmt.Sprintf("%+v", string(res))), nil
}
