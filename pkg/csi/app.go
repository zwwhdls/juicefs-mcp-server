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

type PodWithStatus struct {
	Name      string
	Namespace string
	Kind      string
	NodeName  string
	Status    PodStatus
}

type PodStatus struct {
	Phase                 corev1.PodPhase
	Conditions            []corev1.PodCondition
	Message               string
	Reason                string
	InitContainerStatuses []corev1.ContainerStatus
	ContainerStatuses     []corev1.ContainerStatus
}

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
	if pod == nil {
		return nil, fmt.Errorf("pod %s not found", podName)
	}
	podSts := PodWithStatus{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Kind:      "Pod",
		NodeName:  pod.Spec.NodeName,
		Status: PodStatus{
			Phase:                 pod.Status.Phase,
			Conditions:            pod.Status.Conditions,
			Message:               pod.Status.Message,
			Reason:                pod.Status.Reason,
			InitContainerStatuses: pod.Status.InitContainerStatuses,
			ContainerStatuses:     pod.Status.ContainerStatuses,
		},
	}

	res, _ := json.Marshal(podSts)
	c.log.Debugw("get pod", "pod", podSts)
	return mcp.NewToolResultText(fmt.Sprintf("%+v", string(res))), nil
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
