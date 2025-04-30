package csi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *CSIHandler) handleGetHandleFlow(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c.log.Debugw("handleGetHandleFlow", "request", request.Params.Arguments)
	return mcp.NewToolResultText(`
排查业务容器挂载问题时，可以通过以下步骤进行：
1. 判断 PVC 是否和 PV 绑定成功，使用 tool get_juicefs_pv_of_app_pod;
2. 判断 Mount Pod 是否创建成功并正常运行，使用 tool get_mount_pod_by_pv;
3. 如果 Mount Pod 已经创建，查看 Mount Pod 的日志，使用 tool get_log_of_pod;
4. 如果 Mount Pod 没有创建，查看 CSI Node Pod 的日志，使用 tool get_csi_node_pod + get_log_of_pod;
`), nil
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
		pod     *corev1.Pod
		pvs     = []corev1.PersistentVolume{}
		pvNames = []string{}
		err     error
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
				pvs = append(pvs, *pv)
				pvNames = append(pvNames, pv.Name)
			}
		}
	}

	c.log.Debugw("get pv", "pvs", strings.Join(pvNames, ","))

	res, _ := yaml.Marshal(pvs)
	return mcp.NewToolResultText(fmt.Sprintf("%+v", string(res))), nil
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
	c.log.Debugw("get csi node", "csiNode", csiNode.Name)

	res, _ := yaml.Marshal(csiNode)
	return mcp.NewToolResultText(fmt.Sprintf("%+v", string(res))), nil
}

func (c *CSIHandler) handleGetPod(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c.log.Debugw("handleGetName", "argument", request.Params.Arguments)
	podName, ok := request.Params.Arguments["podName"].(string)
	if !ok {
		c.log.Errorw("Missing argument")
		return nil, fmt.Errorf("missing podName")
	}
	namespace, ok := request.Params.Arguments["namespace"].(string)
	if !ok {
		namespace = "default"
	}

	pod, err := c.client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	res, _ := yaml.Marshal(pod)
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

	res, _ := yaml.Marshal(node)
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
	for _, pod := range mountPodsList {
		mountPodNames = append(mountPodNames, pod.Name)
	}

	c.log.Debugw("get mount pod", "uniqueId", uniqueId, "mountPodNames", strings.Join(mountPodNames, ","))
	res, _ := yaml.Marshal(mountPodsList)
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

func (c *CSIHandler) handlePodLog(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c.log.Debugw("handlePodLog", "argument", request.Params.Arguments)
	podName, ok := request.Params.Arguments["podName"].(string)
	if !ok {
		return nil, fmt.Errorf("missing podName")
	}
	namespace, ok := request.Params.Arguments["namespace"].(string)
	if !ok {
		return nil, fmt.Errorf("missing namespace")
	}
	tail, ok := request.Params.Arguments["tailLines"].(int64)
	if !ok {
		tail = int64(20)
	}

	pod, err := c.client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}
	if pod == nil {
		return nil, fmt.Errorf("pod %s not found", podName)
	}

	req := c.client.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: pod.Spec.Containers[0].Name,
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

	c.log.Debugw("Pod Log", "podName", podName, "namespace", namespace, "tailLines", tail, "logs", str)
	return mcp.NewToolResultText(str), nil
}
