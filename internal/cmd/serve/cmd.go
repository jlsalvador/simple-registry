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

	opts := []config.Option{}

	if flags.DataDir != "" {
		opts = append(opts, config.WithDataDir(flags.DataDir))
	}

	if flags.AdminName != "" {
		opts = append(opts, config.WithAdminName(flags.AdminName))
	}

	if flags.AdminPwdFile != "" {
		opts = append(opts, config.WithAdminPwdFile(flags.AdminPwdFile))
	} else if flags.AdminPwd != "" {
		opts = append(opts, config.WithAdminPwd([]byte(flags.AdminPwd)))
	}

	if flags.TokenSecretFile != "" {
		opts = append(opts, config.WithTokenSecretFile(flags.TokenSecretFile))
	} else if flags.TokenSecret != "" {
		opts = append(opts, config.WithTokenSecret([]byte(flags.TokenSecret)))
	}

	if flags.TokenTimeout != 0 {
		opts = append(opts, config.WithTokenTimeout(flags.TokenTimeout))
	}

	if len(flags.CfgDir) > 0 {
		opts = append(opts, config.WithCfgDirs(flags.CfgDir))
	}

	cfg, err := config.New(opts...)
	if err != nil {
		return fmt.Errorf("failed to create configuration: %w", err)
	}
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	h := handler.NewHandler(*cfg, flags.UI)

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
