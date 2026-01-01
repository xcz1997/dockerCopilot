package image

import (
	"context"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/xcz1997/dockerCopilot/internal/utiles"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateSourceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateSourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSourceLogic {
	return &UpdateSourceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateSourceLogic) UpdateSource(req *types.UpdateImageSourceReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 验证来源类型
	validTypes := map[string]bool{
		model.SourceTypeRemote:  true,
		model.SourceTypeLocal:   true,
		model.SourceTypePrivate: true,
	}
	if !validTypes[req.SourceType] {
		resp.Code = 400
		resp.Msg = "无效的来源类型，只支持 remote、local 或 private"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 私有类型必须指定 Registry
	if req.SourceType == model.SourceTypePrivate && req.RegistryHost == "" {
		resp.Code = 400
		resp.Msg = "私有镜像必须指定 Registry 地址"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 获取镜像信息
	imageList, err := utiles.GetImagesList(l.svcCtx)
	if err != nil {
		resp.Code = 500
		resp.Msg = "获取镜像列表失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 查找镜像
	var found bool
	var imageName, imageTag string
	for _, img := range imageList {
		if img.ID == req.Id {
			found = true
			imageName = img.ImageName
			imageTag = img.ImageTag
			break
		}
	}

	if !found {
		resp.Code = 404
		resp.Msg = "镜像不存在"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 更新或创建镜像元数据
	meta := &model.ImageMetadata{
		ImageID:      req.Id,
		ImageName:    imageName,
		ImageTag:     imageTag,
		SourceType:   req.SourceType,
		RegistryHost: req.RegistryHost,
	}

	err = model.UpsertImageMetadata(meta)
	if err != nil {
		resp.Code = 500
		resp.Msg = "更新镜像来源类型失败: " + err.Error()
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	// 如果设置为 local，从更新检查缓存中移除
	if req.SourceType == model.SourceTypeLocal {
		l.svcCtx.HubImageInfo.RemoveImage(req.Id)
		logx.Infof("镜像 %s 已标记为本地镜像，从更新检查中排除", req.Id[:12])
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{
		"imageId":      req.Id,
		"sourceType":   req.SourceType,
		"registryHost": req.RegistryHost,
	}
	return resp, nil
}
