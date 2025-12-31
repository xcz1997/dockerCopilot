package settings

import (
	"net/http"

	"github.com/xcz1997/dockerCopilot/internal/logic/settings"
	"github.com/xcz1997/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func PrivateRegistriesGetHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := settings.NewPrivateRegistriesGetLogic(r.Context(), svcCtx)
		resp, err := l.PrivateRegistriesGet()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
