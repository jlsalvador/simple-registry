package handler

import (
	"encoding/json"
	netHttp "net/http"
	"strings"
)

func (m *ServeMux) Token(w netHttp.ResponseWriter, r *netHttp.Request) {
	q := r.URL.Query()
	scopes := q["scope"]

	rUsr, rPwd, ok := r.BasicAuth()

	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="registry-token"`)
		w.WriteHeader(netHttp.StatusUnauthorized)
		return
	}

	// Check if the user exists and password is valid.
	if !m.cfg.Rbac.HasUser(rUsr, rPwd) {
		w.WriteHeader(netHttp.StatusForbidden)
		return
	}

	fullScope := strings.Join(scopes, " ")
	token, err := m.cfg.Rbac.GenerateToken(rUsr, fullScope)
	if err != nil {
		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	payload := map[string]string{
		"token": token,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}
