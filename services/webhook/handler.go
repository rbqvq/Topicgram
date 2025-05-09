package webhook

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func Handler() http.Handler {
	return h2c.NewHandler(&handler{gin: NewEngine()}, &http2.Server{})
}

type handler struct {
	gin *gin.Engine
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.RemoteAddr, ":") {
		r.RemoteAddr = "127.0.0.1:0"
	}

	h.gin.ServeHTTP(w, r)
}
