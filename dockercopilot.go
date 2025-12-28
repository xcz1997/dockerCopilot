package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/xcz1997/dockerCopilot/internal/config"
	"github.com/xcz1997/dockerCopilot/internal/handler"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/utiles"
	"github.com/robfig/cron/v3"
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

	// 启动群组调度器
	if ctx.GroupScheduler != nil {
		// 设置进度更新器，让群组更新操作能在任务列表中显示
		ctx.GroupScheduler.SetProgressUpdater(svc.NewProgressAdapter(ctx))
		if err := ctx.GroupScheduler.Start(); err != nil {
			logx.Errorf("启动群组调度器失败: %v", err)
		}
		defer ctx.GroupScheduler.Stop()
	}

	list, err := utiles.GetImagesList(ctx)
	if err != nil {
		logx.Errorf("panic获取镜像列表出错: %v", err)
		panic(err)
	}
	go ctx.HubImageInfo.CheckUpdate(list)
	corndanmu := cron.New(cron.WithParser(cron.NewParser(
		cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
	)))
	_, err = corndanmu.AddFunc("30 * * * *", func() {
		list, err := utiles.GetImagesList(ctx)
		if err != nil {
			logx.Errorf("panic获取镜像列表出错: %v", err)
			panic(err)
		}
		ctx.HubImageInfo.CheckUpdate(list)
	})
	if err != nil {
		logx.Errorf("panic添加定时任务出错: %v", err)
		panic(err)
	}
	corndanmu.Start()
	defer corndanmu.Stop()
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
	spaRoutes := []string{"/manager", "/manager/containers", "/manager/images", "/manager/groups", "/manager/backups", "/manager/settings", "/manager/login", "/manager/tasks", "/manager/projects"}
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
	logx.AddWriter(logx.NewWriter(os.Stdout))
	return nil
}
