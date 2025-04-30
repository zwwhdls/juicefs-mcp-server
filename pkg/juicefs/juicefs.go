package juicefs

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func (j *JuiceFSHandler) getJuiceFSWorkflow(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	j.log.Debugw("get juicefs workflow")
	return mcp.NewToolResultText(`
对于 JuiceFS 的任何性能问题，可以遵循以下步骤：
1. 需要先获取到 JuiceFS 的 mountpoint，使用 tool find_mountpoint;
2. 通过 mountpoint 来进行性能测试，使用 tool bench_in_juicefs;
`), nil
}

func (j *JuiceFSHandler) handleBench(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	mountpoint, ok := request.Params.Arguments["mountpoint"].(string)
	if !ok {
		j.log.Errorw("missing mountpoint", "request", request)
		return nil, fmt.Errorf("missing mountpoint")
	}
	j.log.Debugw("handleBench", "mountpoint", mountpoint)

	cmd := j.exec.CommandContext(ctx, "juicefs", "bench", mountpoint)
	res, err := cmd.CombinedOutput()
	if err != nil {
		j.log.Errorw("exec bench error", "mountpoint", mountpoint, "err", err)
		return nil, fmt.Errorf("bench error: %w", err)
	}
	fmt.Println(string(res))
	return mcp.NewToolResultText(fmt.Sprintf("%+v", string(res))), nil
}

func (j *JuiceFSHandler) handleStats(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	mountpoint, ok := request.Params.Arguments["mountpoint"].(string)
	if !ok {
		j.log.Errorw("missing mountpoint", "request", request)
		return nil, fmt.Errorf("missing mountpoint")
	}
	interval, ok := request.Params.Arguments["interval"].(int)
	if !ok {
		interval = 3
	}
	j.log.Debugw("handleStats", "mountpoint", mountpoint)
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(interval))
	defer cancel()

	cmd := j.exec.CommandContext(timeoutCtx, "juicefs", "stats", mountpoint, "-l", "1")
	res, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(err.Error(), "signal: killed") {
		j.log.Errorw("exec juicefs stats error", "mountpoint", mountpoint, "err", err, "res", string(res))
		return nil, fmt.Errorf("stats error: %w", err)
	}
	fmt.Println(string(res))
	return mcp.NewToolResultText(fmt.Sprintf("%+v", string(res))), nil
}

func (j *JuiceFSHandler) handleAccessLog(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	mountpoint, ok := request.Params.Arguments["mountpoint"].(string)
	if !ok {
		j.log.Errorw("missing mountpoint", "request", request)
		return nil, fmt.Errorf("missing mountpoint")
	}
	interval, ok := request.Params.Arguments["interval"].(int)
	if !ok {
		interval = 3
	}
	j.log.Debugw("handleStats", "mountpoint", mountpoint)
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(interval))
	defer cancel()

	cmd := j.exec.CommandContext(timeoutCtx, "cat", filepath.Join(mountpoint, ".accesslog"))
	res, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(err.Error(), "signal: killed") {
		j.log.Errorw("exec cat accesslog error", "mountpoint", mountpoint, "err", err)
		return nil, fmt.Errorf("accesslog error: %w", err)
	}
	fmt.Println(string(res))
	return mcp.NewToolResultText(fmt.Sprintf("%+v", string(res))), nil
}
