package rbac

import "errors"

var ErrAuthHeaderNotFound = errors.New("authorization http header not found")
var ErrAuthCredentialsInvalid = errors.New("invalid authorization credentials")
var ErrHttpRequestInvalid = errors.New("invalid http request")
var ErrBasicAuthInvalid = errors.New("invalid basic auth credentials")
var ErrInvalidVerb = errors.New("invalid verb")
