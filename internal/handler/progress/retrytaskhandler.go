package progress

import (
	"net/http"

	"github.com/xcz1997/dockerCopilot/internal/logic/progress"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func RetryTaskHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskId := r.URL.Query().Get("taskId")
		if taskId == "" {
			httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{
				"code": 400,
				"msg":  "taskId 不能为空",
				"data": map[string]interface{}{},
			})
			return
		}

		l := progress.NewRetryTaskLogic(r.Context(), serverCtx)
		resp, err := l.RetryTask(taskId)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
