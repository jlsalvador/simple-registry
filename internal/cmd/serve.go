package cmd

import (
	"fmt"
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
	adminPwdFile,
	certFile,
	keyFile,
	rbacDir string,
) error {
	var rbacEngine *rbac.Engine
	var err error
	if rbacDir != "" {
		rbacEngine, err = config.LoadRBACFromYamlDir(rbacDir)
	} else {
		rbacEngine, err = config.GetRBACEngineStatic(adminName, adminPwd, adminPwdFile)
	}
	if err != nil {
		return err
	}

	config := config.Config{
		Rbac: *rbacEngine,
		Data: filesystem.NewFilesystemDataStorage(dataDir),
	}

	h := handler.NewHandler(config)

	isTLS := certFile != "" && keyFile != ""

	fmt.Printf("Listening on %s (%s)\n", addr, func() string {
		if isTLS {
			return "HTTPS"
		}
		return "HTTP"
	}())

	if isTLS {
		return http.ListenAndServeTLS(addr, certFile, keyFile, h)
	}
	return http.ListenAndServe(addr, h)
}
