package environment

import (
	"net/http"

	"github.com/xcz1997/dockerCopilot/internal/logic/environment"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func EnvironmentListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := environment.NewEnvironmentListLogic(r.Context(), svcCtx)
		resp, err := l.EnvironmentList()
		if err != nil {
			httpx.WriteJson(w, resp.Code, resp)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
