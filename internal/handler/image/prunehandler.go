package image

import (
	"net/http"

	"github.com/xcz1997/dockerCopilot/internal/logic/image"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// PruneHandler 清除所有未使用的镜像
func PruneHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := image.NewPruneLogic(r.Context(), svcCtx)
		resp, err := l.Prune()
		if err != nil {
			httpx.WriteJson(w, resp.Code, resp)
		} else {
			httpx.WriteJson(w, resp.Code, resp)
		}
	}
}
