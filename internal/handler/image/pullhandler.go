package image

import (
	"net/http"

	"github.com/xcz1997/dockerCopilot/internal/logic/image"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/xcz1997/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func PullHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetNewImageReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := image.NewPullLogic(r.Context(), svcCtx)
		resp, err := l.Pull(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
