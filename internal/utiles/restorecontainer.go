package utiles

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	dockerBackend "github.com/docker/docker/api/types/backend"
	"github.com/xcz1997/dockerCopilot/internal/module"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

func RestoreContainer(ctx *svc.ServiceContext, filename string, taskID string) error {
	var backupList []string
	basePath := os.Getenv("BACKUP_DIR") // 从环境变量中获取备份目录
	if basePath == "" {
		basePath = "/data/backups" // 如果环境变量未设置，使用默认值
	}
	fullPath := filepath.Join(basePath, filename)
	oldProgress := svc.TaskProgress{
		TaskID:     taskID,
		Percentage: 0,
		Message:    "",
		Name:       "",
		DetailMsg:  "",
		IsDone:     false,
	}
	oldProgress.Name = "恢复容器"
	content, err := os.ReadFile(fullPath)
	if err != nil {
		logx.Error("Failed to read file: %s", err)
		oldProgress.Percentage = 0
		oldProgress.Message = "读取文件失败或者未找到文件。请确认文件名仅由大小写字母、数字和短横线组成"
		oldProgress.DetailMsg = err.Error()
		oldProgress.IsDone = true
		ctx.UpdateProgress(taskID, oldProgress)
	}
	var configList []dockerBackend.ContainerCreateConfig
	err = json.Unmarshal(content, &configList)
	if err != nil {
		logx.Error("Failed to parse json: %s", err)
		oldProgress.Percentage = 0
		oldProgress.Message = "解析文件失败"
		oldProgress.DetailMsg = err.Error()
		oldProgress.IsDone = true
		ctx.UpdateProgress(taskID, oldProgress)
	}
	for i, containerInfo := range configList {
		info := "正在恢复第" + strconv.Itoa(i+1) + "个容器"
		oldProgress.Percentage = int(float64(i) / float64(len(configList)) * 100)
		oldProgress.Message = info
		oldProgress.DetailMsg = info
		ctx.UpdateProgress(taskID, oldProgress)
		ctx.DockerClient.NegotiateAPIVersion(context.TODO())

		// 使用统一的镜像拉取方法（支持加速器和私有 Registry 认证）
		imageName := containerInfo.Config.Image
		pullOpts := module.PullImageOptions{
			ImageName:      imageName,
			UseAccelerator: true,
			RegistryAuth:   module.GetPullAuthForImage(imageName),
		}
		reader, pullResult, err := module.PullImage(context.TODO(), ctx.DockerClient, pullOpts)
		if err != nil {
			backupList = append(backupList, imageName+"拉取镜像出现错误"+err.Error())
			logx.Errorf("Failed to pull image: %s", err)
			continue
		}
		err = decodePullResp(reader, ctx, taskID)
		if err != nil {
			backupList = append(backupList, imageName+"拉取镜像出现错误"+err.Error())
			logx.Errorf("Failed to pull image: %s", err)
			continue
		}

		// 如果使用了加速器，需要重新打标签
		if pullResult.ActualImageName != imageName {
			logx.Infof("重新打标签: %s -> %s", pullResult.ActualImageName, imageName)
			if err := module.TagImage(context.TODO(), ctx.DockerClient, pullResult.ActualImageName, imageName); err != nil {
				logx.Errorf("重新打标签失败: %v", err)
			}
		}
		_, err = ctx.DockerClient.ContainerCreate(context.TODO(), containerInfo.Config, containerInfo.HostConfig, containerInfo.NetworkingConfig, nil, containerInfo.Name)
		if err != nil {
			logx.Error("Failed to create container: %s", err)
			info = "正在恢复第" + strconv.Itoa(i+1) + "个容器"
			backupList = append(backupList, containerInfo.Name+"恢复失败"+err.Error())
			continue
		} else {
			backupList = append(backupList, containerInfo.Name+"恢复成功")
		}
	}
	oldProgress.Percentage = 100
	oldProgress.DetailMsg = strings.Join(backupList, ",\n")
	oldProgress.Message = "恢复完成"
	oldProgress.IsDone = true
	ctx.UpdateProgress(taskID, oldProgress)
	return nil
}
