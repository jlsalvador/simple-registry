package registry

const RegExpName = "[a-z0-9]+(?:(?:.|_|__|-+)[a-z0-9]+)*(?:/[a-z0-9]+(?:(?:.|_|__|-+)[a-z0-9]+)*)*"
const RegExpTag = "[a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}"
const RegExpDigest = "[a-z0-9]+(?:[+._-][a-z0-9])?:[a-zA-Z0-9=_-]+"
const RegExpUUID = "[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}"
