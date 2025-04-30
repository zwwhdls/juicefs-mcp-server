package csi

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	k8sexec "k8s.io/utils/exec"

	"juicefs-mcp/pkg/tools"
	"juicefs-mcp/pkg/utils/logger"
)

const (
	DriverName          = "csi.juicefs.com"
	PodTypeKey          = "app.kubernetes.io/name"
	PodTypeValue        = "juicefs-mount"
	PodUniqueIdLabelKey = "volume-id"
	MountContainerName  = "jfs-mount"
	MountShare          = "STORAGE_CLASS_SHARE_MOUNT"
)

type CSIHandler struct {
	exec         k8sexec.Interface
	log          *zap.SugaredLogger
	sysNamespace string
	client       *kubernetes.Clientset
}

func NewCSIHandler(sysNamespace string, client *kubernetes.Clientset) *CSIHandler {
	return &CSIHandler{
		exec:         k8sexec.New(),
		log:          logger.NewLogger("csi"),
		sysNamespace: sysNamespace,
		client:       client,
	}
}

func RegisterJuiceCSITools(csiHandler *CSIHandler) {
	tools.RegistryTool(server.ServerTool{
		Tool: mcp.NewTool("get_handle_flow",
			mcp.WithDescription("获取排查 JuiceFS CSI 挂载问题的流程"),
		),
		Handler: csiHandler.handleGetHandleFlow,
	})
	tools.RegistryTool(server.ServerTool{
		Tool: mcp.NewTool("get_juicefs_pv_of_app_pod",
			mcp.WithDescription("获取应用 Pod 使用的 JuiceFS PV"),
			mcp.WithString("appName",
				mcp.Description("应用 Pod 名称"),
				mcp.Required(),
			),
			mcp.WithString("namespace",
				mcp.Description("应用 Pod 的 namespace"),
			),
		),
		Handler: csiHandler.handleGetJuiceFSPVOfApp,
	})
	tools.RegistryTool(server.ServerTool{
		Tool: mcp.NewTool("get_csi_node_pod",
			mcp.WithDescription("获取对应节点上的 CSI Node Pod"),
			mcp.WithString("nodeName",
				mcp.Description("节点名"),
				mcp.Required(),
			),
		),
		Handler: csiHandler.handleGetCSINodePod,
	})
	tools.RegistryTool(server.ServerTool{
		Tool: mcp.NewTool("get_pod",
			mcp.WithDescription("根据 pod 名获取 pod 的 yaml，可以查看 pod 的所有信息，包括 pod 使用的 PVC、所在节点等"),
			mcp.WithString("podName",
				mcp.Description("pod 名"),
				mcp.Required(),
			),
		),
		Handler: csiHandler.handleGetPod,
	})
	tools.RegistryTool(server.ServerTool{
		Tool: mcp.NewTool("get_node",
			mcp.WithDescription("根据 node 名获取 node yaml"),
			mcp.WithString("nodeName",
				mcp.Description("节点名"),
				mcp.Required(),
			),
		),
		Handler: csiHandler.handleGetNode,
	})
	tools.RegistryTool(server.ServerTool{
		Tool: mcp.NewTool("get_mount_pod_by_pv",
			mcp.WithDescription("根据 pv 获取对应节点上的 JuiceFS Mount Pod，可以查看 Mount Pod 的配置，包括 Mount Pod 的资源限制、挂载点、镜像等"),
			mcp.WithString("nodeName",
				mcp.Description("节点名"),
				mcp.Required(),
			),
			mcp.WithString("pvName",
				mcp.Description("PV 名称"),
				mcp.Required(),
			),
		),
		Handler: csiHandler.handleGetMountPodByPV,
	})
	tools.RegistryTool(server.ServerTool{
		Tool: mcp.NewTool("get_log_of_mount_pod",
			mcp.WithDescription("根据 pv 获取对应节点上 Mount Pod 的日志"),
			mcp.WithString("nodeName",
				mcp.Description("节点名"),
				mcp.Required(),
			),
			mcp.WithString("pvName",
				mcp.Description("PV 名称"),
				mcp.Required(),
			),
			mcp.WithNumber("tailLines",
				mcp.Description("获取的日志行数"),
			),
		),
		Handler: csiHandler.handleMountPodLogByPV,
	})
	tools.RegistryTool(server.ServerTool{
		Tool: mcp.NewTool("get_log_of_pod",
			mcp.WithDescription("获取 Pod 日志"),
			mcp.WithString("podName",
				mcp.Description("Pod 名称"),
				mcp.Required(),
			),
			mcp.WithString("namespace",
				mcp.Description("Pod 的 namespace"),
				mcp.Required(),
			),
			mcp.WithNumber("tailLines",
				mcp.Description("获取的日志行数"),
			),
		),
		Handler: csiHandler.handlePodLog,
	})
}
