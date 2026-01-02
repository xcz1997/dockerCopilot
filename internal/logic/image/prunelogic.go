package image

import (
	"context"

	"github.com/docker/docker/api/types/image"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/xcz1997/dockerCopilot/internal/utiles"
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

// Prune 清除所有未使用的镜像（与前端显示一致）
func (l *PruneLogic) Prune() (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 获取镜像列表（与前端显示一致的逻辑）
	imagesList, err := utiles.GetImagesList(l.svcCtx)
	if err != nil {
		logx.Errorf("获取镜像列表失败: %v", err)
		resp.Code = 500
		resp.Msg = "获取镜像列表失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 筛选出未使用的镜像（与前端 stats.unused 一致的判断逻辑）
	var unusedImages []types.Image
	for _, img := range imagesList {
		// 跳过 dangling 镜像（前端不显示这些）
		if img.ImageName == "None" || img.ImageTag == "None" {
			continue
		}
		// 只删除未使用的镜像
		if !img.InUsed {
			unusedImages = append(unusedImages, img)
		}
	}

	if len(unusedImages) == 0 {
		resp.Code = 200
		resp.Msg = "success"
		resp.Data = map[string]interface{}{
			"deletedCount":   0,
			"spaceReclaimed": uint64(0),
			"spaceStr":       "0 B",
		}
		return resp, nil
	}

	// 逐个删除未使用的镜像
	var deletedCount int
	var spaceReclaimed uint64
	var failedImages []string

	for _, img := range unusedImages {
		// 删除镜像
		deleteResp, err := l.svcCtx.DockerClient.ImageRemove(l.ctx, img.ID, image.RemoveOptions{
			Force:         false,
			PruneChildren: true, // 同时删除未被其他镜像引用的父层
		})
		if err != nil {
			logx.Errorf("删除镜像 %s:%s 失败: %v", img.ImageName, img.ImageTag, err)
			failedImages = append(failedImages, img.ImageName+":"+img.ImageTag)
			continue
		}

		deletedCount++
		// 统计释放的空间
		for _, item := range deleteResp {
			if item.Deleted != "" {
				spaceReclaimed += uint64(img.Size)
				break // 只计算一次主镜像大小
			}
		}
		logx.Infof("已删除镜像: %s:%s", img.ImageName, img.ImageTag)
	}

	// 格式化释放的空间大小
	spaceStr := formatSize(spaceReclaimed)

	logx.Infof("清除未使用镜像完成: 删除 %d 个镜像，释放 %s", deletedCount, spaceStr)

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{
		"deletedCount":   deletedCount,
		"spaceReclaimed": spaceReclaimed,
		"spaceStr":       spaceStr,
		"failedImages":   failedImages,
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
		return intToString(int64(bytes)) + " B"
	}
}

func formatFloat(f float64) string {
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
