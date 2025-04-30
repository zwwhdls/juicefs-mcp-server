package main

import (
	"context"
	"flag"

	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"

	ctrl "sigs.k8s.io/controller-runtime"

	"juicefs-mcp/pkg/csi"
	"juicefs-mcp/pkg/juicefs"
	"juicefs-mcp/pkg/tools"
	"juicefs-mcp/pkg/utils"
	"juicefs-mcp/pkg/utils/logger"
)

const (
	version = "0.1.0"
)

var (
	transport    string
	sseUrl       string
	debug        bool
	sysNamespace string
	handlerName  string
)

var JuiceMCPServer = server.NewMCPServer(
	"juicefs-mcp-server",
	version,
)

func init() {
	flag.StringVar(&transport, "t", "sse", "transport protocol")
	flag.StringVar(&sseUrl, "sseurl", "0.0.0.0:8088", "sse url")
	flag.BoolVar(&debug, "debug", false, "debug mode")
	flag.StringVar(&sysNamespace, "sysnamespace", "kube-system", "namespace of JuiceFS CSI driver")
	flag.StringVar(&handlerName, "handler", "csi", "handler kind")
}

func initTools(log *zap.SugaredLogger) {
	switch handlerName {
	case "juicefs":
		// Register JuiceFS tools
		log.Infow("init juicefs handler")
		juicefsHandler := juicefs.NewJuiceFSHandler()
		juicefs.RegisterJuiceFSTools(juicefsHandler)
	case "csi":
		// Register JuiceFS CSI tools
		config := ctrl.GetConfigOrDie()
		clientSet, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err)
		}
		log.Infow("init csi handler")
		csiHandler := csi.NewCSIHandler(sysNamespace, clientSet)
		csi.RegisterJuiceCSITools(csiHandler)
	}

	for _, tool := range tools.ToolRegistry {
		JuiceMCPServer.AddTool(tool.Tool, tool.Handler)
	}
}

func main() {
	flag.Parse()
	logger.InitLogger()
	logger.SetDebug(debug)
	defer logger.Sync()
	log := logger.NewLogger("main")

	initTools(log)

	switch transport {
	case "stdio":
		if err := server.ServeStdio(JuiceMCPServer); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	case "sse":
		sseServer := server.NewSSEServer(JuiceMCPServer, server.WithBaseURL(sseUrl))

		stop := utils.HandleTerminalSignal()
		go func() {
			<-stop
			ctx, cancel := context.WithTimeout(context.Background(), 5)
			defer cancel()
			if err := sseServer.Shutdown(ctx); err != nil {
				log.Fatalw("Failed to gracefully shutdown sse", "error", err)
			}
		}()

		log.Infow("SSE server start", "address", sseUrl)
		if err := sseServer.Start(sseUrl); err != nil {
			log.Fatalw("Failed to start sse", "error", err)
		}
	default:
		log.Fatalw("Invalid transport type. Must be 'stdio' or 'sse'.", "transport", transport)
	}
}
