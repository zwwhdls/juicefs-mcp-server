package juicefs

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
	k8sexec "k8s.io/utils/exec"

	"juicefs-mcp/pkg/tools"
	"juicefs-mcp/pkg/utils/logger"
)

type JuiceFSHandler struct {
	exec    k8sexec.Interface
	log     *zap.SugaredLogger
	binPath string
	isCE    bool
}

func NewJuiceFSHandler() *JuiceFSHandler {
	return &JuiceFSHandler{
		exec:    k8sexec.New(),
		log:     logger.NewLogger("juicefs"),
		binPath: "/usr/bin/juicefs",
		isCE:    true,
	}
}

func RegisterJuiceFSTools(jfsHandler *JuiceFSHandler) {
	// fs
	tools.RegistryTool(server.ServerTool{
		Tool: mcp.NewTool("find_mountpoint",
			mcp.WithDescription("查看机器上的挂载点"),
		),
		Handler: jfsHandler.handleFindMountPoint,
	})
	// juicefs
	tools.RegistryTool(server.ServerTool{
		Tool: mcp.NewTool("bench_in_juicefs",
			mcp.WithDescription("通过挂载点进行 juicefs 的性能测试"),
			mcp.WithString("mountpoint",
				mcp.Description("挂载点"),
				mcp.Required(),
			),
		),
		Handler: jfsHandler.handleBench,
	})
	tools.RegistryTool(server.ServerTool{
		Tool: mcp.NewTool("stats_in_juicefs",
			mcp.WithDescription("通过挂载点实时统计 JuiceFS 性能指标"),
			mcp.WithString("mountpoint",
				mcp.Description("挂载点"),
				mcp.Required(),
			),
			mcp.WithNumber("interval",
				mcp.Description("统计间隔"),
				mcp.Required(),
			),
		),
		Handler: jfsHandler.handleStats,
	})
	tools.RegistryTool(server.ServerTool{
		Tool: mcp.NewTool("accesslog_in_juicefs",
			mcp.WithDescription("通过挂载点获取基于文件系统访问日志的实时监控数据"),
			mcp.WithString("mountpoint",
				mcp.Description("挂载点"),
				mcp.Required(),
			),
			mcp.WithNumber("interval",
				mcp.Description("统计间隔"),
				mcp.Required(),
			),
		),
		Handler: jfsHandler.handleAccessLog,
	})
	tools.RegistryTool(server.ServerTool{
		Tool: mcp.NewTool("get_mount_options",
			mcp.WithDescription("通过挂载点查看客户端的载参数"),
			mcp.WithString("mountpoint",
				mcp.Description("挂载点"),
				mcp.Required(),
			),
		),
		Handler: jfsHandler.handleFindMountOptions,
	})
	//// docs
	//tools.RegistryTool(server.ServerTool{
	//	Tool: mcp.NewTool("how_to_diagnosis_read_amplification",
	//		mcp.WithDescription("什么是 JuiceFS 的读放大问题，以及应该如何优化"),
	//	),
	//	Handler: handleDocsOfReadAmplification,
	//})
	//tools.RegistryTool(server.ServerTool{
	//	Tool: mcp.NewTool("how_to_diagnosis_write_amplification",
	//		mcp.WithDescription("什么是 JuiceFS 的写放大问题，以及应该如何优化"),
	//	),
	//	Handler: handleDocsOfWriteAmplification,
	//})
	//tools.RegistryTool(server.ServerTool{
	//	Tool: mcp.NewTool("what_is_meta_cache",
	//		mcp.WithDescription("元数据缓存的作用，以及如何调优"),
	//	),
	//	Handler: handleDocsOfMetaCache,
	//})
	//tools.RegistryTool(server.ServerTool{
	//	Tool: mcp.NewTool("what_is_data_cache",
	//		mcp.WithDescription("数据缓存的作用，以及如何调优，包括内核页缓存、内核回写模式、客户端读缓存、客户端写缓存。"),
	//	),
	//	Handler: handleDocsOfDataCache,
	//})
	//tools.RegistryTool(server.ServerTool{
	//	Tool: mcp.NewTool("what_is_read_ahead_and_prefetch",
	//		mcp.WithDescription("什么是 JuiceFS 的预读和预取"),
	//	),
	//	Handler: handleDocsOfReadAheadAndPrefetch,
	//})
	//
	//tools.RegistryTool(server.ServerTool{
	//	Tool: mcp.NewTool("what_is_buffer_size",
	//		mcp.WithDescription("buffer-size 的对读写数据的作用及其调优方法"),
	//	),
	//	Handler: handleDocsOfBufferSize,
	//})
	//tools.RegistryTool(server.ServerTool{
	//	Tool: mcp.NewTool("what_is_read_ahead_and_prefetch",
	//		mcp.WithDescription("什么是 JuiceFS 的预读和预取"),
	//	),
	//	Handler: handleDocsOfReadAheadAndPrefetch,
	//})
}
