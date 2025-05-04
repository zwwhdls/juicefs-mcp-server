package csi

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/mark3labs/mcp-go/mcp"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

type NodeWithStatus struct {
	Name   string
	Kind   string
	Status NodeStatus
}
type NodeStatus struct {
	Capacity    corev1.ResourceList
	Allocatable corev1.ResourceList
	Phase       corev1.NodePhase
	Conditions  []corev1.NodeCondition
}

func (c *CSIHandler) handleGetCSINodePod(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c.log.Debugw("handleGetCSINodePod", "argument", request.Params.Arguments)
	nodeName, ok := request.Params.Arguments["nodeName"].(string)
	if !ok {
		c.log.Errorw("Missing argument", "nodeName", nodeName)
		return nil, fmt.Errorf("missing nodeName")
	}

	csiNode, err := c.GetCSINode(ctx, nodeName)
	if err != nil {
		return nil, err
	}
	if csiNode == nil {
		return nil, fmt.Errorf("CSI node on %s not found", nodeName)
	}
	csists := PodWithStatus{
		Name:      csiNode.Name,
		Namespace: csiNode.Namespace,
		Kind:      "Pod",
		NodeName:  csiNode.Spec.NodeName,
		Status: PodStatus{
			Phase:             csiNode.Status.Phase,
			Conditions:        csiNode.Status.Conditions,
			Message:           csiNode.Status.Message,
			Reason:            csiNode.Status.Reason,
			ContainerStatuses: csiNode.Status.ContainerStatuses,
		},
	}
	c.log.Debugw("get csi node", "csiNode", csists)

	res, _ := json.Marshal(csists)
	return mcp.NewToolResultText(fmt.Sprintf("%+v", string(res))), nil
}

func (c *CSIHandler) handleGetNode(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c.log.Debugw("handleGetNode", "argument", request.Params.Arguments)
	nodeName, ok := request.Params.Arguments["nodeName"].(string)
	if !ok {
		c.log.Errorw("Missing argument", "nodeName", nodeName)
		return nil, fmt.Errorf("missing nodeName")
	}

	node, err := c.client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if node == nil {
		return nil, fmt.Errorf("node %s not found", nodeName)
	}
	nodeSts := NodeWithStatus{
		Name: node.Name,
		Kind: "Node",
		Status: NodeStatus{
			Capacity:    node.Status.Capacity,
			Allocatable: node.Status.Allocatable,
			Phase:       node.Status.Phase,
			Conditions:  node.Status.Conditions,
		},
	}

	res, _ := json.Marshal(nodeSts)
	return mcp.NewToolResultText(fmt.Sprintf("%+v", string(res))), nil
}

func (c *CSIHandler) handleGetMountPodByPV(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c.log.Debugw("handleGetMountPodByPV", "argument", request.Params.Arguments)
	pvName, ok := request.Params.Arguments["pvName"].(string)
	if !ok {
		c.log.Errorw("Missing argument", "pvName", pvName)
		return nil, fmt.Errorf("missing pvName")
	}
	nodeName, ok := request.Params.Arguments["nodeName"].(string)
	if !ok {
		c.log.Errorw("Missing argument", "nodeName", nodeName)
		return nil, fmt.Errorf("missing nodeName")
	}

	pv, err := c.client.CoreV1().PersistentVolumes().Get(ctx, pvName, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}
	if pv.Spec.CSI == nil || pv.Spec.CSI.Driver != DriverName {
		return nil, fmt.Errorf("PV %s is not JuiceFS PV", pvName)
	}

	uniqueId := pv.Spec.CSI.VolumeHandle
	csiNode, err := c.GetCSINode(ctx, nodeName)
	if err != nil {
		return nil, err
	}
	for _, v := range csiNode.Spec.Containers[0].Env {
		if v.Name == MountShare {
			uniqueId = pv.Spec.StorageClassName
		}
	}

	mountPodsList, err := c.GetMountPodOnNode(ctx, nodeName, uniqueId)
	if err != nil {
		return nil, err
	}

	mountPodNames := make([]string, 0)
	mountPodSts := []PodWithStatus{}
	for _, pod := range mountPodsList {
		mountPodNames = append(mountPodNames, pod.Name)
		mountPodSts = append(mountPodSts, PodWithStatus{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Kind:      "Pod",
			NodeName:  pod.Spec.NodeName,
			Status: PodStatus{
				Phase:             pod.Status.Phase,
				Conditions:        pod.Status.Conditions,
				Message:           pod.Status.Message,
				Reason:            pod.Status.Reason,
				ContainerStatuses: pod.Status.ContainerStatuses,
			},
		})
	}

	res, _ := json.Marshal(mountPodSts)
	c.log.Debugw("get mount pod", "uniqueId", uniqueId, "mountPods", mountPodSts)
	return mcp.NewToolResultText(fmt.Sprintf("%+v", string(res))), nil
}

func (c *CSIHandler) handleMountPodLogByPV(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c.log.Debugw("handleMountPodLogByPV", "argument", request.Params.Arguments)
	pvName, ok := request.Params.Arguments["pvName"].(string)
	if !ok {
		c.log.Errorw("Missing argument", "pvName", pvName)
		return nil, fmt.Errorf("missing pvName")
	}
	nodeName, ok := request.Params.Arguments["nodeName"].(string)
	if !ok {
		c.log.Errorw("Missing argument", "nodeName", nodeName)
		return nil, fmt.Errorf("missing nodeName")
	}
	tail, ok := request.Params.Arguments["tailLines"].(int64)
	if !ok {
		tail = int64(20)
	}

	pv, err := c.client.CoreV1().PersistentVolumes().Get(ctx, pvName, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}
	if pv.Spec.CSI == nil || pv.Spec.CSI.Driver != DriverName {
		return nil, fmt.Errorf("PV %s is not JuiceFS PV", pvName)
	}

	uniqueId := pv.Spec.CSI.VolumeHandle
	csiNode, err := c.GetCSINode(ctx, nodeName)
	if err != nil {
		return nil, err
	}
	for _, v := range csiNode.Spec.Containers[0].Env {
		if v.Name == MountShare {
			uniqueId = pv.Spec.StorageClassName
		}
	}

	mountPodsList, err := c.GetMountPodOnNode(ctx, nodeName, uniqueId)
	if err != nil {
		return nil, err
	}

	mountPodNames := make([]string, 0)
	for _, pod := range mountPodsList {
		mountPodNames = append(mountPodNames, pod.Name)
	}
	if len(mountPodNames) == 0 {
		return nil, fmt.Errorf("mount pod not found")
	}
	mountPod := mountPodsList[0]
	req := c.client.CoreV1().Pods(mountPod.Namespace).GetLogs(mountPod.Name, &corev1.PodLogOptions{
		Container: mountPod.Spec.Containers[0].Name,
		TailLines: &tail,
	})
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return nil, err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return nil, err
	}
	str := buf.String()

	c.log.Debugw("Pod Log", "mount pod name", mountPod.Name, "namespace", mountPod.Namespace, "tailLines", tail, "logs", str)
	return mcp.NewToolResultText(str), nil
}
