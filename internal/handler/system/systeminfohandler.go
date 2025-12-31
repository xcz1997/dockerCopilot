package system

import (
	"net/http"

	"github.com/xcz1997/dockerCopilot/internal/logic/system"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func SystemInfoHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := system.NewSystemInfoLogic(r.Context(), svcCtx)
		resp, err := l.SystemInfo()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
