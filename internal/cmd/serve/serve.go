package serve

import (
	"fmt"
	"net/http"

	"github.com/jlsalvador/simple-registry/internal/config"
	"github.com/jlsalvador/simple-registry/internal/http/handler"
	"github.com/jlsalvador/simple-registry/internal/version"
	"github.com/jlsalvador/simple-registry/pkg/log"
)

const CmdName = "serve"
const CmdHelp = "Start the registry server"

func CmdFn() error {
	flags, err := parseFlags()
	if err != nil {
		return err
	}

	var cfg *config.Config
	if len(flags.CfgDir) > 0 {
		cfg, err = config.NewFromYamlDir(
			flags.CfgDir,
			flags.DataDir,
		)
	} else {
		cfg, err = config.New(
			flags.AdminName,
			flags.AdminPwd,
			flags.AdminPwdFile,
			flags.DataDir,
		)
	}
	if err != nil {
		return err
	}
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	cfg.IsWebUIEnabled = flags.WebUI

	h := handler.NewHandler(*cfg)

	isTLS := flags.CertFile != "" && flags.KeyFile != ""
	scheme := "HTTP"
	if isTLS {
		scheme = "HTTPS"
	}

	log.Info(
		"service.name", version.AppName,
		"service.version", version.AppVersion,
		"event.dataset", "cmd.serve",
		"addr", flags.Addr,
		"scheme", scheme,
		"message", "listening for requests",
	).Print()

	if isTLS {
		return http.ListenAndServeTLS(flags.Addr, flags.CertFile, flags.KeyFile, h)
	}
	return http.ListenAndServe(flags.Addr, h)
}
