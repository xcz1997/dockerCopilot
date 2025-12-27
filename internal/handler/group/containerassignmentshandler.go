package group

import (
	"net/http"

	"github.com/xcz1997/dockerCopilot/internal/logic/group"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ContainerAssignmentsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := group.NewContainerAssignmentsLogic(r.Context(), svcCtx)
		resp, err := l.ContainerAssignments()
		if err != nil {
			httpx.WriteJson(w, resp.Code, resp)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
