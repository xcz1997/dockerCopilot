package progress

import (
	"net/http"

	"github.com/xcz1997/dockerCopilot/internal/logic/progress"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListTasksHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取查询参数
		status := r.URL.Query().Get("status") // all, current, history

		l := progress.NewListTasksLogic(r.Context(), serverCtx)
		resp, err := l.ListTasks(status)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
