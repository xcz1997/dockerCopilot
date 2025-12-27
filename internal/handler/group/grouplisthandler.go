package group

import (
	"net/http"

	"github.com/xcz1997/dockerCopilot/internal/logic/group"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GroupListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := group.NewGroupListLogic(r.Context(), svcCtx)
		resp, err := l.GroupList()
		if err != nil {
			httpx.WriteJson(w, resp.Code, resp)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
