package image

import (
	"context"
	"encoding/json"
	"time"

	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/module"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/xcz1997/dockerCopilot/internal/utiles"

	"github.com/zeromicro/go-zero/core/logx"
)

type ImagesListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type Info struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	Tag            string `json:"tag"`
	Size           string `json:"size"`
	InUsed         bool   `json:"inUsed"`
	CreateTime     string `json:"createTime"`
	HaveUpdate     bool   `json:"haveUpdate"`
	SourceType     string `json:"sourceType,omitempty"`     // remote/local/private
	RegistryHost   string `json:"registryHost,omitempty"`   // 私有 Registry 地址
	LastCheckAt    string `json:"lastCheckAt,omitempty"`    // 最后检查时间
	LastCheckError string `json:"lastCheckError,omitempty"` // 最后检查错误
}

func NewImagesListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ImagesListLogic {
	return &ImagesListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ImagesListLogic) ImagesList() (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 检查是否为远程环境
	if l.svcCtx.CurrentEnvironment != nil && l.svcCtx.CurrentEnvironment.EnvType == model.EnvTypeRemote {
		return l.getRemoteImages()
	}

	// 本地环境
	list, err := utiles.GetImagesList(l.svcCtx)
	if err != nil {
		resp.Code = 500
		resp.Msg = err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}

	// 获取所有镜像元数据
	metadataMap, err := model.GetImageMetadataMap()
	if err != nil {
		logx.Errorf("获取镜像元数据失败: %v", err)
		metadataMap = make(map[string]*model.ImageMetadata)
	}

	resp.Code = 200
	resp.Msg = "success"
	var imageInfoList []Info
	for _, v := range list {
		// 跳过 dangling 镜像（没有 RepoTags 的孤立镜像）
		if v.ImageName == "None" || v.ImageTag == "None" {
			continue
		}
		var imageInfo Info
		imageInfo.Id = v.ID
		imageInfo.Name = v.ImageName
		imageInfo.Tag = v.ImageTag
		imageInfo.Size = v.SizeFormat
		imageInfo.InUsed = v.InUsed
		t := time.Unix(v.Created, 0)
		imageInfo.CreateTime = t.Format("2006-01-02 15:04:05")
		// 检查镜像是否有更新
		if hubInfo, ok := l.svcCtx.HubImageInfo.Data[v.ID]; ok {
			imageInfo.HaveUpdate = hubInfo.NeedUpdate
		}
		// 获取镜像元数据（来源类型、绑定的 Registry）
		if meta, ok := metadataMap[v.ID]; ok {
			imageInfo.SourceType = meta.SourceType
			imageInfo.RegistryHost = meta.RegistryHost
			if meta.LastCheckAt != nil {
				imageInfo.LastCheckAt = meta.LastCheckAt.Format("2006-01-02 15:04:05")
			}
			imageInfo.LastCheckError = meta.LastCheckError
		} else {
			// 默认为 remote
			imageInfo.SourceType = model.SourceTypeRemote
		}
		imageInfoList = append(imageInfoList, imageInfo)
	}
	resp.Data = imageInfoList
	return resp, nil
}

// getRemoteImages 从远程环境获取镜像列表
func (l *ImagesListLogic) getRemoteImages() (resp *types.Resp, err error) {
	resp = &types.Resp{}

	// 获取或创建远程客户端
	client := l.svcCtx.RemoteClient
	if client == nil {
		env := l.svcCtx.CurrentEnvironment
		client = module.NewRemoteClientWithToken(env.URL, env.SecretKey, env.JWTToken)
	}

	// 从远程获取镜像列表
	data, err := client.ProxyRequest("GET", "/api/images", nil)
	if err != nil {
		resp.Code = 500
		resp.Msg = "获取远程镜像列表失败: " + err.Error()
		resp.Data = []interface{}{}
		return resp, nil
	}

	// 解析远程返回的数据
	var images []Info
	if err := json.Unmarshal(data, &images); err != nil {
		resp.Code = 500
		resp.Msg = "解析远程镜像数据失败: " + err.Error()
		resp.Data = []interface{}{}
		return resp, nil
	}

	resp.Code = 200
	resp.Msg = "success"
	resp.Data = images
	return resp, nil
}
