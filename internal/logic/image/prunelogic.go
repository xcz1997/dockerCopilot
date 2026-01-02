package image

import (
	"context"

	"github.com/docker/docker/api/types/filters"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type PruneLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPruneLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PruneLogic {
	return &PruneLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Prune 清除所有未使用的镜像
func (l *PruneLogic) Prune() (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 使用 Docker API 的 ImagesPrune 功能
	// 这会删除所有没有被任何容器使用的镜像（dangling 和 unused）
	pruneFilters := filters.NewArgs()
	// 添加 dangling=false 会包含所有未使用的镜像，不仅仅是 dangling 的
	pruneFilters.Add("dangling", "false")

	report, err := l.svcCtx.DockerClient.ImagesPrune(l.ctx, pruneFilters)
	if err != nil {
		logx.Errorf("清除未使用镜像失败: %v", err)
		resp.Code = 500
		resp.Msg = "清除失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 统计删除的镜像数量和释放的空间
	deletedCount := len(report.ImagesDeleted)
	spaceReclaimed := report.SpaceReclaimed

	// 格式化释放的空间大小
	var spaceStr string
	if spaceReclaimed >= 1024*1024*1024 {
		spaceStr = formatSize(spaceReclaimed)
	} else if spaceReclaimed >= 1024*1024 {
		spaceStr = formatSize(spaceReclaimed)
	} else if spaceReclaimed >= 1024 {
		spaceStr = formatSize(spaceReclaimed)
	} else {
		spaceStr = formatSize(spaceReclaimed)
	}

	logx.Infof("清除未使用镜像完成: 删除 %d 个镜像，释放 %s", deletedCount, spaceStr)

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{
		"deletedCount":   deletedCount,
		"spaceReclaimed": spaceReclaimed,
		"spaceStr":       spaceStr,
	}
	return resp, nil
}

// formatSize 格式化字节大小
func formatSize(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return formatFloat(float64(bytes)/float64(GB)) + " GB"
	case bytes >= MB:
		return formatFloat(float64(bytes)/float64(MB)) + " MB"
	case bytes >= KB:
		return formatFloat(float64(bytes)/float64(KB)) + " KB"
	default:
		return formatFloat(float64(bytes)) + " B"
	}
}

func formatFloat(f float64) string {
	if f == float64(int64(f)) {
		return string(rune(int(f))) + ""
	}
	// 使用简单的格式化，保留1位小数
	intPart := int64(f)
	decPart := int64((f - float64(intPart)) * 10)
	if decPart == 0 {
		return intToString(intPart)
	}
	return intToString(intPart) + "." + intToString(decPart)
}

func intToString(n int64) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToString(-n)
	}
	var result []byte
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	return string(result)
}
