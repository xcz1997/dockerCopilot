package image

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/image"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type PullLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPullLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PullLogic {
	return &PullLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PullLogic) Pull(req *types.GetNewImageReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}

	if req.ImageNameAndTag == "" {
		resp.Code = 400
		resp.Msg = "镜像名称不能为空"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	imageName := req.ImageNameAndTag
	logx.Infof("开始拉取镜像: %s", imageName)

	// 拉取镜像
	pullOut, err := l.svcCtx.DockerClient.ImagePull(l.ctx, imageName, image.PullOptions{})
	if err != nil {
		resp.Code = 500
		resp.Msg = fmt.Sprintf("拉取镜像失败: %v", err)
		resp.Data = map[string]interface{}{}
		return resp, nil
	}
	defer pullOut.Close()

	// 读取完成
	_, err = io.Copy(io.Discard, pullOut)
	if err != nil {
		resp.Code = 500
		resp.Msg = fmt.Sprintf("读取拉取结果失败: %v", err)
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	logx.Infof("镜像拉取完成: %s", imageName)

	resp.Code = 200
	resp.Msg = "镜像拉取成功"
	resp.Data = map[string]interface{}{
		"imageName": imageName,
	}
	return resp, nil
}
