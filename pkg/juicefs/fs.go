package juicefs

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func (j *JuiceFSHandler) handleFindMountPoint(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	j.log.Debugw("handleFindMountPoint", "request", request)
	cmd := j.exec.CommandContext(ctx, "df")
	res, err := cmd.CombinedOutput()
	if err != nil {
		j.log.Errorw("exec df error", "error", err)
		return nil, fmt.Errorf("exec df error: %w", err)
	}
	results := []string{}
	mounts := string(res)
	lines := strings.Split(mounts, "\n")
	for line := range lines {
		if strings.Contains(lines[line], "JuiceFS") {
			results = append(results, lines[line])
		}
	}
	j.log.Debugw("handleFindMountPoint", "results", results)
	return mcp.NewToolResultText(fmt.Sprintf("%+v", strings.Join(results, "\n"))), nil
}

func (j *JuiceFSHandler) handleFindMountOptions(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	j.log.Debugw("handleFindMountOptions", "request", request)
	mountpoint, ok := request.Params.Arguments["mountpoint"].(string)
	if !ok {
		j.log.Errorw("missing mountpoint", "request", request)
		return nil, fmt.Errorf("missing mountpoint")
	}
	cmd := j.exec.CommandContext(ctx, "ps", "-ef")
	res, err := cmd.CombinedOutput()
	if err != nil {
		j.log.Errorw("exec ps error", "error", err)
		return nil, fmt.Errorf("exec ps error: %w", err)
	}
	results := []string{}
	mounts := string(res)
	lines := strings.Split(mounts, "\n")
	for line := range lines {
		if strings.Contains(lines[line], "juicefs") && strings.Contains(lines[line], mountpoint) {
			results = append(results, lines[line])
		}
	}
	j.log.Debugw("handleFindMountOptions", "results", results)
	return mcp.NewToolResultText(fmt.Sprintf("%+v", strings.Join(results, "\n"))), nil
}
