package cmd

import (
	"fmt"
	"net/http"

	"github.com/jlsalvador/simple-registry/internal/config"
	"github.com/jlsalvador/simple-registry/internal/http/handler"
	"github.com/jlsalvador/simple-registry/internal/version"
	"github.com/jlsalvador/simple-registry/pkg/log"
)

func Serve(
	addr,
	dataDir,
	adminName,
	adminPwd,
	adminPwdFile,
	certFile,
	keyFile,
	cfgDir string,
) error {
	var cfg *config.Config
	var err error

	if cfgDir != "" {
		cfg, err = config.NewFromYamlDir(
			cfgDir,
			dataDir,
		)
	} else {
		cfg, err = config.New(
			adminName,
			adminPwd,
			adminPwdFile,
			dataDir,
		)
	}
	if err != nil {
		return err
	}
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	h := handler.NewHandler(*cfg)

	isTLS := certFile != "" && keyFile != ""
	scheme := "HTTP"
	if isTLS {
		scheme = "HTTPS"
	}

	log.Info(
		"service.name", version.AppName,
		"service.version", version.AppVersion,
		"event.dataset", "cmd.serve",
		"addr", addr,
		"scheme", scheme,
		"msg", "listening for requests",
	).Print()

	if isTLS {
		return http.ListenAndServeTLS(addr, certFile, keyFile, h)
	}
	return http.ListenAndServe(addr, h)
}
