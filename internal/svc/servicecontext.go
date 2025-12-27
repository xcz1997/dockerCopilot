package svc

import (
	"database/sql"
	"os"
	"sync"

	"github.com/docker/docker/client"
	"github.com/xcz1997/dockerCopilot/internal/config"
	"github.com/xcz1997/dockerCopilot/internal/model"
	"github.com/xcz1997/dockerCopilot/internal/module"
	"github.com/xcz1997/dockerCopilot/internal/scheduler"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

type ServiceContext struct {
	Config                     config.Config
	CookieCheckMiddleware      rest.Middleware
	Jwtuuid                    string
	BearerTokenCheckMiddleware rest.Middleware
	JwtSecret                  string
	PortainerJwt               string
	HubImageInfo               *module.ImageUpdateData
	IndexCheckMiddleware       rest.Middleware
	ProgressStore              ProgressStoreType
	DockerClient               *client.Client
	DB                         *sql.DB
	GroupScheduler             *scheduler.GroupScheduler
	mu                         sync.Mutex
}

type TaskProgress struct {
	TaskID     string
	Percentage int
	Message    string
	Name       string
	DetailMsg  string
	IsDone     bool
}

type ProgressStoreType map[string]TaskProgress

func NewServiceContext(c config.Config) *ServiceContext {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logx.Errorf("Unable to create docker client: %s", err)
	}

	// 初始化数据库，优先从环境变量读取数据目录，默认 /data
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/data"
	}
	db, err := model.InitDB(dataDir)
	if err != nil {
		logx.Errorf("Unable to initialize database: %s", err)
	}

	// 创建镜像更新检查器
	hubImageInfo := module.NewImageCheck()

	// 创建群组调度器
	groupScheduler := scheduler.NewGroupScheduler(cli, hubImageInfo)

	return &ServiceContext{
		Config:         c,
		HubImageInfo:   hubImageInfo,
		ProgressStore:  make(ProgressStoreType),
		DockerClient:   cli,
		DB:             db,
		GroupScheduler: groupScheduler,
	}
}

func (ctx *ServiceContext) UpdateProgress(taskID string, progress TaskProgress) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.ProgressStore[taskID] = progress
}

func (ctx *ServiceContext) GetProgress(taskID string) (TaskProgress, bool) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	progress, ok := ctx.ProgressStore[taskID]
	return progress, ok
}
