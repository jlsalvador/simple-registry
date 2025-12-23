package registry

import "regexp"

const ExprName = "^[a-z0-9]+(?:(?:.|_|__|-+)[a-z0-9]+)*(?:/[a-z0-9]+(?:(?:.|_|__|-+)[a-z0-9]+)*)*$"
const ExprTag = "^[a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}$"
const ExprDigest = "^[a-z0-9]+(?:[+._-][a-z0-9])?:[a-zA-Z0-9=_-]+$"
const ExprUUID = "^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$"

var RegExprName = regexp.MustCompile(ExprName)
var RegExprTag = regexp.MustCompile(ExprTag)
var RegExprDigest = regexp.MustCompile(ExprDigest)
var RegExprUUID = regexp.MustCompile(ExprUUID)
