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

	cfg, err := buildConfig(&flags)
	if err != nil {
		return err
	}

	return runServer(cfg)
}

func buildConfig(flags *Flags) (*config.Config, error) {
	opts := buildOptions(flags)

	cfg, err := config.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create configuration: %w", err)
	}
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	return cfg, nil
}

func buildOptions(flags *Flags) []config.Option {
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

	if flags.Addr != "" {
		opts = append(opts, config.WithHttpAddr(flags.Addr))
	}

	if flags.UI {
		opts = append(opts, config.WithHttpUI(flags.UI))
	}

	if flags.CertFile != "" {
		opts = append(opts, config.WithHttpCertFile(flags.CertFile))
	}

	if flags.KeyFile != "" {
		opts = append(opts, config.WithHttpKeyFile(flags.KeyFile))
	}

	return opts
}

func runServer(cfg *config.Config) error {
	h := handler.NewHandler(*cfg)

	isTLS := cfg.Http.CertFile != "" && cfg.Http.KeyFile != ""

	scheme := "HTTP"
	if isTLS {
		scheme = "HTTPS"
	}

	log.Info(
		"service.name", version.AppName,
		"service.version", version.AppVersion,
		"event.dataset", "cmd.serve",
		"addr", cfg.Http.Addr,
		"scheme", scheme,
		"message", "listening for requests",
	).Print()

	if isTLS {
		return http.ListenAndServeTLS(
			cfg.Http.Addr,
			cfg.Http.CertFile,
			cfg.Http.KeyFile,
			h,
		)
	}

	return http.ListenAndServe(cfg.Http.Addr, h)
}
