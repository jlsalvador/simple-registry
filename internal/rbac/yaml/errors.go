package yaml

import "errors"

var (
	ErrWhileParsing   = errors.New("error while parsing")
	ErrWhileUnmarshal = errors.New("error while unmarshal")
)
