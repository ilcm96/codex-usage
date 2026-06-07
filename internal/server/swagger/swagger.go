package swagger

import (
	"net/http"

	_ "github.com/ilcm96/codex-usage/docs"
	httpSwagger "github.com/swaggo/http-swagger"
)

func Register(mux *http.ServeMux) {
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)
}
