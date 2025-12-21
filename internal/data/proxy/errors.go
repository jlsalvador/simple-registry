package proxy

import "errors"

var (
	ErrDataStorageNotInitialized = errors.New("data storage not initialized, use NewProxyDataStorage()")
	ErrUpstreamError             = errors.New("upstream error")
)
