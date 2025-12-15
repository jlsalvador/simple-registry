package config

import (
	"github.com/jlsalvador/simple-registry/internal/data"
	"github.com/jlsalvador/simple-registry/internal/rbac"
)

type Config struct {
	Rbac rbac.Engine
	Data data.DataStorage
}
