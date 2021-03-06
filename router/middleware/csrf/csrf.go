package csrf

import (
	"net/http"

	"github.com/octavore/naga/service"

	"github.com/octavore/nagax/logger"
	"github.com/octavore/nagax/router"
	csrf2 "github.com/octavore/nagax/users/csrf"
	"github.com/octavore/nagax/users/session"
	"github.com/octavore/nagax/util/errors"
)

var csrfWhitelist = map[string]bool{
	"GET":     true,
	"HEAD":    true,
	"OPTIONS": true,
	"TRACE":   true,
}

type Module struct {
	Router  *router.Module
	Session *session.Module
	CSRF    *csrf2.Module
	Logger  *logger.Module
}

func (m *Module) Init(c *service.Config) {
}

func (m *Module) New(ignorePaths ...string) func(rw http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	ignoreMap := map[string]bool{}
	for _, url := range ignorePaths {
		ignoreMap[url] = true
	}

	return func(rw http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
		if !csrfWhitelist[req.Method] {
			session, err := m.Session.Verify(req)
			if err != nil {
				m.Router.HandleError(rw, req, errors.Wrap(err))
				return
			}
			// only check csrf if user is logged in
			// don't check csrf for logged out users right now
			if session != "" && !ignoreMap[req.URL.Path] {
				csrfToken := req.Header.Get("x-csrf-token")
				var ok bool
				if csrfToken == "" && req.Method == "POST" {
					csrfToken = req.PostFormValue("csrfToken")
				}
				if csrfToken == "" {
					ok = false
				} else {
					ok, err = m.CSRF.Verify(session, csrfToken)
					if err != nil {
						ok = false
						m.Logger.Error(errors.Wrap(err))
					}
				}
				if !ok {
					err := router.NewRequestError(req, http.StatusBadRequest, "invalid csrf token")
					m.Router.HandleError(rw, req, errors.Wrap(err))
					return
				}
			}
		}
		next(rw, req)
	}
}
