# 📦 Simple Registry

A lightweight OCI-compatible container registry with RBAC support and pull-through caching.

---

## 🚀 Features

- __🎖️ OCI Native:__ Implements the [OCI Distribution Specification v1.1.1][oci-spec].
- __🪶 Lightweight:__ Low memory footprint and minimal dependencies.
- __🛂 Granular RBAC:__ Fine-grained control per repository, action, and role.
- __📦 Pull-through Caching:__ On-demand caching from external registries.
- __🔒 Flexible Authentication:__ Anonymous access, Basic Auth, and time-bound Bearer tokens.
- __🌀 Stateless & Scalable:__ High availability supported when backed by shared storage.

---

## 🏗️ Status

The core functionality is stable, and OCI conformance is actively being validated.

- ✅ Completed:
  - Core push/pull
  - RBAC, and Auth models.
  - Pull-through cache.
- 📆 Upcoming:
  - Garbage collection of unused blobs.
  - The internal YAML configuration schema is still evolving and may change in
    backward-incompatible ways before v1.0.

Pull requests are welcome.

---

## 🏁 Quick Start

### 1. Secure Your Credentials

Generate a secure password hash for your YAML configuration:

```sh
simple-registry genhash
```

### 2. Launch the registry

Run the registry with a data directory and administrative credentials:

```sh
simple-registry serve \
  -datadir ./data \
  -adminname admin \
  -adminpwdfile ./admin-password.txt \
  -addr 0.0.0.0:5000
```

> __ℹ️ Note:__ For production, use the `-cert` and `-key` flags to listen on `https://`.

```sh
# Generate a self-signed TLS certificate for testing purposes.
openssl req -new -x509 \
    -keyout tls.key \
    -out tls.crt \
    -days 36500 \
    -nodes -subj "/C=SE/ST=ES/L=Sevilla/O=ACME/CN=localhost"

simple-registry serve \
  -datadir /var/lib/registry \
  -adminname admin \
  -adminpwdfile ./admin-password.txt \
  -cert tls.crt \
  -key tls.key
```

### 3. Usage Example

```sh
# Login to your new registry
docker login localhost:5000

# Tag and push an image
docker tag busybox localhost:5000/library/busybox
docker push localhost:5000/library/busybox
```

---

## ⚙️ Configuration & RBAC

The registry is configured via YAML manifests.
You can split your configuration into multiple files and directories using the `-cfgdir` flag.

| Component | Description                                              |
| --------- | -------------------------------------------------------- |
| Storage   | Defines where the blobs and manifests are stored.        |
| Identity  | Definition of Users and Groups.                          |
| RBAC      | Rules linking roles and role-bindings to control access. |
| Cache     | Configuration for pull-through cache targets.            |

Example:

```sh
simple-registry serve \
  -cfgdir ./config \
  -cfgdir ./rbac \
  -cfgdir ./proxies
```

---

## 🔒 Authentication Model

You can define, using regular expressions, which users and groups have access to specific repositories.

Simple Registry evaluates requests in three tiers:

1. __Anonymous:__ Mix public/private repositories.
2. __Basic Auth:__ Defines users and groups for basic authentication. Hash password by bcrypt.
3. __Bearer Token:__ Supports issued tokens with built-in expiration validation.

> __ℹ️ Note:__ There are some manifests examples in [docs/yaml/examples](docs/yaml/examples)

---

## 📄 License

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
