package svc

import (
	"database/sql"
	"os"
	"sync"
	"time"

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

// SubTask 子任务
type SubTask struct {
	Name       string     `json:"name"`
	Status     string     `json:"status"` // pending, in_progress, completed, failed
	Message    string     `json:"message"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

type TaskProgress struct {
	TaskID     string
	Percentage int
	Message    string
	Name       string
	DetailMsg  string
	IsDone     bool
	StartedAt  time.Time
	FinishedAt *time.Time
	SubTasks   []SubTask
	// 用于重试的元数据
	TaskType   string // container_update, group_check, group_update
	TargetID   string // 容器ID或群组ID
	TargetName string // 容器名或群组名
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

func (ctx *ServiceContext) GetAllTasks() []TaskProgress {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	tasks := make([]TaskProgress, 0, len(ctx.ProgressStore))
	for _, task := range ctx.ProgressStore {
		tasks = append(tasks, task)
	}
	return tasks
}

// ProgressAdapter 适配器，实现 scheduler.TaskProgressUpdater 接口
type ProgressAdapter struct {
	svcCtx *ServiceContext
}

// NewProgressAdapter 创建进度适配器
func NewProgressAdapter(ctx *ServiceContext) *ProgressAdapter {
	return &ProgressAdapter{svcCtx: ctx}
}

// UpdateProgress 实现 scheduler.TaskProgressUpdater 接口
func (a *ProgressAdapter) UpdateProgress(taskID string, percentage int, message string, name string, detailMsg string, isDone bool) {
	a.svcCtx.mu.Lock()
	defer a.svcCtx.mu.Unlock()

	now := time.Now()
	existing, exists := a.svcCtx.ProgressStore[taskID]

	if !exists {
		// 新任务
		existing = TaskProgress{
			TaskID:    taskID,
			StartedAt: now,
		}
	}

	existing.Percentage = percentage
	existing.Message = message
	existing.Name = name
	existing.DetailMsg = detailMsg
	existing.IsDone = isDone

	if isDone && existing.FinishedAt == nil {
		existing.FinishedAt = &now
	}

	a.svcCtx.ProgressStore[taskID] = existing
}

// UpdateProgressWithMeta 更新进度并设置元数据（用于重试）
func (a *ProgressAdapter) UpdateProgressWithMeta(taskID string, percentage int, message string, name string, detailMsg string, isDone bool, taskType string, targetID string, targetName string) {
	a.svcCtx.mu.Lock()
	defer a.svcCtx.mu.Unlock()

	now := time.Now()
	existing, exists := a.svcCtx.ProgressStore[taskID]

	if !exists {
		existing = TaskProgress{
			TaskID:     taskID,
			StartedAt:  now,
			TaskType:   taskType,
			TargetID:   targetID,
			TargetName: targetName,
		}
	}

	existing.Percentage = percentage
	existing.Message = message
	existing.Name = name
	existing.DetailMsg = detailMsg
	existing.IsDone = isDone

	if isDone && existing.FinishedAt == nil {
		existing.FinishedAt = &now
	}

	a.svcCtx.ProgressStore[taskID] = existing
}

// UpdateSubTask 更新子任务
func (a *ProgressAdapter) UpdateSubTask(taskID string, subTaskName string, status string, message string) {
	a.svcCtx.mu.Lock()
	defer a.svcCtx.mu.Unlock()

	existing, exists := a.svcCtx.ProgressStore[taskID]
	if !exists {
		return
	}

	now := time.Now()
	found := false
	for i, st := range existing.SubTasks {
		if st.Name == subTaskName {
			existing.SubTasks[i].Status = status
			existing.SubTasks[i].Message = message
			if status == "in_progress" && existing.SubTasks[i].StartedAt == nil {
				existing.SubTasks[i].StartedAt = &now
			}
			if (status == "completed" || status == "failed") && existing.SubTasks[i].FinishedAt == nil {
				existing.SubTasks[i].FinishedAt = &now
			}
			found = true
			break
		}
	}

	if !found {
		subTask := SubTask{
			Name:    subTaskName,
			Status:  status,
			Message: message,
		}
		if status == "in_progress" {
			subTask.StartedAt = &now
		}
		if status == "completed" || status == "failed" {
			subTask.FinishedAt = &now
		}
		existing.SubTasks = append(existing.SubTasks, subTask)
	}

	a.svcCtx.ProgressStore[taskID] = existing
}
