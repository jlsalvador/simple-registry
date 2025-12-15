package yaml

import "errors"

var (
	ErrWhileParsing    = errors.New("error while parsing")
	ErrUnsupportedKind = errors.New("kind is not supported")
	ErrWhileUnmarshal  = errors.New("error while unmarshal")
)
