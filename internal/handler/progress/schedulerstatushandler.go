package progress

import (
	"net/http"

	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func SchedulerStatusHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := &types.Resp{
			Code: 200,
			Msg:  "success",
		}

		if serverCtx.GroupScheduler == nil {
			resp.Code = 500
			resp.Msg = "调度器未初始化"
			resp.Data = map[string]interface{}{}
			httpx.OkJsonCtx(r.Context(), w, resp)
			return
		}

		status := serverCtx.GroupScheduler.GetSchedulerStatus()
		resp.Data = status
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
