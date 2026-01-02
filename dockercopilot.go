package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/robfig/cron/v3"
	"github.com/xcz1997/dockerCopilot/internal/config"
	"github.com/xcz1997/dockerCopilot/internal/handler"
	"github.com/xcz1997/dockerCopilot/internal/middleware"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/utiles"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/x/errors"
	xhttp "github.com/zeromicro/x/http"
	"go/types"
)

//go:embed front/*
var embeddedFront embed.FS

var configFile = flag.String("f", "etc/dockerCopilot.yaml", "the config file")

type UnauthorizedResponse struct {
	Code int                    `json:"code"`
	Msg  string                 `json:"msg"`
	Data map[string]interface{} `json:"data"`
}

func main() {
	logDir := "./logs"
	ErrSetupLog := SetupLog(logDir)
	if ErrSetupLog != nil {
		logx.Errorf("failed to setup log: %v", ErrSetupLog)
		os.Exit(1)
	}
	logx.SetLevel(logx.InfoLevel)
	logx.DisableStat() // 禁用请求统计日志

	flag.Parse()
	var c config.Config
	err := conf.Load(*configFile, &c, conf.UseEnv())
	if err != nil {
		logx.Errorf("无法加载配置文件出错: %v", err)
		logx.Errorf("请确认secretKey设置正确，要求非纯数字且大于八位")
		os.Exit(1)
	}
	server := rest.MustNewServer(c.RestConf, rest.WithCors("*"), rest.WithUnauthorizedCallback(
		func(w http.ResponseWriter, r *http.Request, err error) {
			response := UnauthorizedResponse{
				Code: http.StatusUnauthorized, // 401
				Msg:  "未授权",
				Data: map[string]interface{}{},
			}
			httpx.WriteJson(w, http.StatusUnauthorized, response)
		}))
	defer server.Stop()
	ctx := svc.NewServiceContext(c)

	// 应用性能配置（从数据库加载，环境变量优先级最高）
	perfConfig := ctx.PerformanceConfig
	maxConcurrent := perfConfig.GetEffectiveMaxConcurrent()
	checkInterval := perfConfig.GetEffectiveCheckInterval()

	if perfConfig.LowPowerMode {
		logx.Info("=== 低性能模式已启用 ===")
		logx.Infof("  - 镜像检查并发数: %d (单线程)", maxConcurrent)
		logx.Infof("  - 镜像检查间隔: %d 分钟", checkInterval)
		logx.Infof("  - 启动时检查: %v", !perfConfig.DisableAutoCheck)
	} else {
		logx.Infof("性能配置: 并发数=%d, 检查间隔=%d分钟", maxConcurrent, checkInterval)
	}

	// 设置镜像检查并发数
	ctx.HubImageInfo.SetMaxConcurrent(maxConcurrent)

	// 启动群组调度器
	if ctx.GroupScheduler != nil {
		// 设置进度更新器，让群组更新操作能在任务列表中显示
		ctx.GroupScheduler.SetProgressUpdater(svc.NewProgressAdapter(ctx))
		if err := ctx.GroupScheduler.Start(); err != nil {
			logx.Errorf("启动群组调度器失败: %v", err)
		}
		defer ctx.GroupScheduler.Stop()
	}

	// 启动容器事件监听器
	if ctx.EventWatcher != nil {
		ctx.EventWatcher.Start()
		defer ctx.EventWatcher.Stop()
	}

	// 启动时检查镜像更新（可配置禁用）
	if !perfConfig.DisableAutoCheck {
		list, err := utiles.GetImagesList(ctx)
		if err != nil {
			logx.Errorf("获取镜像列表出错: %v", err)
		} else {
			if perfConfig.LowPowerMode {
				// 低性能模式：同步执行，避免启动时资源竞争
				logx.Info("低性能模式：同步检查镜像更新...")
				ctx.HubImageInfo.CheckUpdate(list)
			} else {
				// 正常模式：异步执行
				go ctx.HubImageInfo.CheckUpdate(list)
			}
		}
	} else {
		logx.Info("启动时镜像更新检查已禁用")
	}

	// 定时镜像检查任务（可配置间隔或禁用）
	var corndanmu *cron.Cron
	if checkInterval > 0 {
		corndanmu = cron.New(cron.WithParser(cron.NewParser(
			cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
		)))
		// 构建 cron 表达式：每 N 分钟执行
		cronExpr := fmt.Sprintf("*/%d * * * *", checkInterval)
		if checkInterval >= 60 {
			// 如果间隔大于等于60分钟，改为每 N 小时执行
			hours := checkInterval / 60
			cronExpr = fmt.Sprintf("0 */%d * * *", hours)
		}
		_, err = corndanmu.AddFunc(cronExpr, func() {
			list, err := utiles.GetImagesList(ctx)
			if err != nil {
				logx.Errorf("定时任务获取镜像列表出错: %v", err)
				return
			}
			ctx.HubImageInfo.CheckUpdate(list)
		})
		if err != nil {
			logx.Errorf("添加镜像检查定时任务出错: %v", err)
		} else {
			corndanmu.Start()
			logx.Infof("镜像检查定时任务已启动: %s", cronExpr)
		}
		defer func() {
			if corndanmu != nil {
				corndanmu.Stop()
			}
		}()
	} else {
		logx.Info("镜像自动检查定时任务已禁用 (CheckIntervalMinutes=0)")
	}
	httpx.SetErrorHandler(func(err error) (int, any) {
		switch e := err.(type) {
		case *errors.CodeMsg:
			return http.StatusOK, xhttp.BaseResponse[types.Nil]{
				Code: e.Code,
				Msg:  e.Msg,
			}
		default:
			return http.StatusOK, xhttp.BaseResponse[types.Nil]{
				Code: 50000,
				Msg:  err.Error(),
			}
		}
	})

	// 添加远程代理中间件（在远程环境时自动代理请求）
	remoteProxyMiddleware := middleware.NewRemoteProxyMiddleware(ctx)
	server.Use(remoteProxyMiddleware.Handle)

	handler.RegisterHandlers(server, ctx)
	RegisterHandlers(server)
	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	logx.Info("程序版本" + config.Version)
	server.Start()
}
func RegisterHandlers(engine *rest.Server) {
	// 创建静态资源处理器
	frontFS, err := fs.Sub(embeddedFront, "front")
	if err != nil {
		log.Fatal(err)
	}
	fileServer := http.FileServer(http.FS(frontFS))
	indexHTML, _ := fs.ReadFile(frontFS, "index.html")

	// SPA 处理器
	spaHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	}

	// 静态资源处理器
	assetsHandler := func(w http.ResponseWriter, r *http.Request) {
		fileServer.ServeHTTP(w, r)
	}

	// favicon 处理器
	faviconHandler := func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/favicon.png"
		fileServer.ServeHTTP(w, r)
	}

	// 注册 SPA 路由
	spaRoutes := []string{"/manager", "/manager/containers", "/manager/images", "/manager/groups", "/manager/backups", "/manager/settings", "/manager/login", "/manager/tasks", "/manager/projects", "/manager/environments"}
	for _, path := range spaRoutes {
		engine.AddRoute(rest.Route{Method: http.MethodGet, Path: path, Handler: spaHandler})
	}

	// 注册静态资源路由
	engine.AddRoute(rest.Route{Method: http.MethodGet, Path: "/assets/:file", Handler: assetsHandler})
	engine.AddRoute(rest.Route{Method: http.MethodGet, Path: "/favicon.png", Handler: faviconHandler})
	engine.AddRoute(rest.Route{Method: http.MethodGet, Path: "/manager/favicon.png", Handler: faviconHandler})
}

// 检查并创建日志目录
func ensureLogDirectory(logDir string) error {
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		return os.MkdirAll(logDir, 0755) // 创建目录并设置权限
	}
	return nil
}

// SetupLog 初始化日志设置
func SetupLog(logDir string) error {
	// 检查日志目录是否存在
	if err := ensureLogDirectory(logDir); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	logConf := logx.LogConf{
		Path:     logDir,
		Level:    "info",
		KeepDays: 7,
		Compress: true,
		Mode:     "file",
	}
	logx.MustSetup(logConf)
	// 使用带过滤功能的 Writer，过滤高频 API 请求日志
	filteredWriter := middleware.NewFilteredWriter(os.Stdout)
	logx.AddWriter(logx.NewWriter(filteredWriter))
	return nil
}
