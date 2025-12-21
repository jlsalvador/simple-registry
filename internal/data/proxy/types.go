package proxy

import (
	"time"

	"github.com/jlsalvador/simple-registry/internal/data"
)

type Proxy struct {
	Url      string
	Timeout  time.Duration
	Username string
	Password string
	Scopes   []string
}

type ProxyDataStorage struct {
	ds      data.DataStorage
	proxies []Proxy
}

func NewProxyDataStorage(ds data.DataStorage, proxies []Proxy) *ProxyDataStorage {
	return &ProxyDataStorage{ds, proxies}
}
