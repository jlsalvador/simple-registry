package cmd

import (
	"net/http"

	"github.com/jlsalvador/simple-registry/internal/config"
	"github.com/jlsalvador/simple-registry/internal/data/filesystem"
	"github.com/jlsalvador/simple-registry/internal/http/handler"
	"github.com/jlsalvador/simple-registry/internal/rbac"
)

func Serve(
	addr,
	dataDir,
	adminName,
	adminPwd,
	certFile,
	keyFile,
	rbacDir string,
) error {
	var rbacEngine rbac.Engine
	if rbacDir != "" {
		rbacEngine = config.LoadRBACFromYamlDir(rbacDir)
	} else {
		rbacEngine = config.GetRBACEngineStatic(adminName, adminPwd)
	}

	config := config.Config{
		Rbac: rbacEngine,
		Data: filesystem.NewFilesystemDataStorage(dataDir),
	}

	h := handler.NewHandler(config)

	if certFile != "" && keyFile != "" {
		return http.ListenAndServeTLS(addr, certFile, keyFile, h)
	}
	return http.ListenAndServe(addr, h)
}
