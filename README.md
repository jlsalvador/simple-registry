# Simple Registry

A lightweight, self-hosted OCI-compatible container registry focused on
simplicity, security and fine-grained access control.

## Table of Contents

1. [Features](#features)
2. [Status](#status)
3. [Quick Star](#quick-start)
4. [Authentication Model](#authentication-model)
5. [Authorization (RBAC)](#authorization-rbac)
6. [OCI Compatibility Notes](#oci-compatibility-notes)
7. [License](#license)

---

## Features

- 🎖️ Implements the [OCI Distribution Specification v1.1.1][oci-spec].
- 🪶 Extremely lightweight, minimal dependencies.
- 🔑 HTTP Basic authentication (username & password).
- ⏰ Bearer token authentication with expiration support.
- 🛂 Fine-grained RBAC (per-repository, per-action, per-role).
- 🕵️ Anonymous access (optional, controlled via RBAC).

---

## Status

🚧 **Active development**

- Core functionality is stable
- OCI conformance tests are actively being validated
- Internal APIs may still evolve

Pull requests are welcome.

---

## Quick Start

### 1. Generate a password hash

```sh
simple-registry -genhash
```

You will be prompted for a password and a secure hash will be printed.
Store this hash in your RBAC configuration.

### 2. Start the registry

```sh
simple-registry \
  -adminname admin \
  -adminpwd-file ./admin-password.txt \
  -datadir ./data \
  -addr 0.0.0.0:5000
```

With TLS:

```sh
openssl req -new -x509 \
    -keyout tls.key \
    -out tls.crt \
    -days 36500 \
    -nodes -subj "/C=SE/ST=ES/L=Seville/O=ACME/CN=localhost"

simple-registry \
  -adminname admin \
  -adminpwd-file ./admin-password.txt \
  -datadir /var/lib/registry \
  -cert tls.crt \
  -key tls.key
```

### 3. Login with Docker or Podman

```sh
docker login localhost:5000
```

or

```sh
podman login --tls-verify=false localhost:5000
```

> ⚠️ When TLS is enabled with a self-signed certificate,
> you may need `--tls-verify=false` or to trust the CA explicitly.

### 4. Push & pull images

```sh
docker tag busybox localhost:5000/library/busybox
docker push localhost:5000/library/busybox
docker pull localhost:5000/library/busybox
```

---

## Authentication Model

Simple Registry currently supports **three authentication modes**:

### 🔓 Anonymous access

If an `anonymous` user exists in RBAC configuration:

- Requests without `Authorization` headers are treated as `anonymous`
- Permissions are evaluated normally via RBAC rules
- No authentication challenge is sent unless required

This allows:

- Public pull-only registries
- Mixed public/private repositories

### 🔑 HTTP Basic authentication

When a request is **not allowed for anonymous** access:

- The registry responds with:

```text
WWW-Authenticate: Basic realm="simple-registry"
```

- Docker/Podman clients will retry automatically with credentials
- Credentials are validated against RBAC users

This matches Docker Registry client expectations.

### ⏰ Bearer tokens

Bearer tokens can be issued externally and validated by the registry:

- Tokens are bound to a user
- Tokens have an expiration time
- Expired tokens are rejected automatically

---

## Authorization (RBAC)

> You can find RBAC manifests examples in [docs/yaml/examples](docs/yaml/examples)

Authorization is enforced **per request**:

- User identity is resolved (anonymous, basic auth, or bearer token)
- RBAC rules are evaluated
- The request is either:

  - ✅ Allowed
  - ❌ Rejected with `401 Unauthorized` (anonymous)
  - ❌ Rejected with `403 Forbidden` (authenticated but unauthorized)

RBAC is applied uniformly across all endpoints.

---

## OCI Compatibility Notes

- Manifests and blobs are stored as raw OCI blobs.
- Manifest lists (indexes) and single-platform manifests are supported.
- API is implemented according to [OCI Distribution Specification v1.1.1][oci-spec].

Docker and Podman clients are tested for:

- Pull
- Push
- Multi-arch images
- Referrers (attestations, artifacts)

---

## License

Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

<http://www.apache.org/licenses/LICENSE-2.0>

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

[oci-spec]: https://github.com/opencontainers/distribution-spec/blob/v1.1.1/spec.md
