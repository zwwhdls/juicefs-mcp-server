package tools

import (
	"sync"

	"github.com/mark3labs/mcp-go/server"
)

var (
	ToolRegistry = []server.ServerTool{}
	lock         = &sync.Mutex{}
)

func RegistryTool(tool server.ServerTool) {
	lock.Lock()
	ToolRegistry = append(ToolRegistry, tool)
	lock.Unlock()
}
