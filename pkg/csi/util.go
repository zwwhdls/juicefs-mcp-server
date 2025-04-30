package csi

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

func (c *CSIHandler) GetCSINode(ctx context.Context, nodeName string) (*corev1.Pod, error) {
	fieldSelector := fields.Set{"spec.nodeName": nodeName}
	nodeLabelMap, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{PodTypeKey: "juicefs-csi-driver", "app": "juicefs-csi-node"},
	})
	csiNodeList, err := c.client.CoreV1().Pods(c.sysNamespace).List(ctx,
		metav1.ListOptions{
			LabelSelector: nodeLabelMap.String(),
			FieldSelector: fieldSelector.String(),
		})
	if err != nil {
		return nil, err
	}
	if csiNodeList == nil || len(csiNodeList.Items) == 0 {
		return nil, nil
	}
	return &csiNodeList.Items[0], nil
}

func (c *CSIHandler) GetMountPodOnNode(ctx context.Context, nodeName, volumeHandle string) ([]corev1.Pod, error) {
	fieldSelector := fields.Set{"spec.nodeName": nodeName}
	labels := map[string]string{PodTypeKey: PodTypeValue}
	if volumeHandle != "" {
		labels[PodUniqueIdLabelKey] = volumeHandle
	}
	mountLabelMap, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: labels})
	mountList, err := c.client.CoreV1().Pods(c.sysNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: mountLabelMap.String(),
		FieldSelector: fieldSelector.String(),
	})
	if err != nil {
		return nil, err
	}
	return mountList.Items, nil
}
