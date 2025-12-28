# 📦 Simple Registry

A lightweight OCI-compatible container registry with RBAC support
and pull-through caching.

---

## 🚀 Features

- **🎖️ OCI Native:** Implements the [OCI Distribution Specification v1.1.1][oci-spec].
- **🪶 Lightweight:** Low memory footprint and minimal dependencies.
- **🛂 Role-based Access Control (RBAC):** Per repository, action, and role.
- **📦 Pull-through Caching:** Configurable on-demand caching from external registries.
- **🔒 Flexible Authentication:** Anonymous, Basic Auth, and tokens.
- **🌀 Stateless & Scalable:** Horizontal scaling backed by shared storage.

---

## 🏗️ Status

The core functionality is stable, and OCI conformance is actively being validated.

- ✅ Completed:
  - Core push/pull
  - RBAC and Auth models.
  - Pull-through cache.
- 📆 Upcoming:
  - Garbage collection of unused blobs.
  - The internal YAML configuration schema is still evolving and may change in
    backward-incompatible ways prior to v1.0.

Pull requests are welcome.

---

## 🏁 Quick Start

### 1. Launch the registry

You can run a registry listening on HTTP with a data directory and
administrative credentials for testing purposes:

```sh
simple-registry serve \
  -datadir ./data \
  -adminpwd secret
```

### 2. Usage Example

```sh
# Login to your new registry
# Username: admin
# Password: secret
docker login localhost:5000

# Tag and push an image
docker tag busybox localhost:5000/library/busybox
docker push localhost:5000/library/busybox
```

---

## 🐳 Launch as a Container

To launch the registry as a container, you can use the following command:

```sh
# Generate an admin hashed password.
#
# Note the space before `echo`. This is necessary to prevent saving the
# password in your shell history.
 echo -n "secret" | docker run \
  -i --rm \
  docker.io/jlsalvador/simple-registry:latest \
    genhash > admin-password.txt

# Create a volume with the proper permissions for nonroot.
#
# The user & group IDs were obtained from:
# https://github.com/GoogleContainerTools/distroless/blob/main/common/variables.bzl#L17
docker run --rm \
  -v simple-registry:/data \
  --user root \
  docker.io/library/busybox \
    chown -R 65532:65534 /data

docker run \
  -d \
  --restart on-failure \
  --name simple-registry \
  -p 5000:5000 \
  -v simple-registry:/data \
  -v $(pwd)/admin-password.txt:/pwd.txt:ro \
  docker.io/jlsalvador/simple-registry:latest \
    serve \
    -adminpwdfile /pwd.txt \
    -datadir /data
```

---

## ⚙️ Configuration & RBAC

The registry can be configured via YAML manifests.
You can split your configuration into multiple files and directories using the
`-cfgdir` flag.

Here are a few components that can be configured:

| Component | Description                                   |
| --------- | --------------------------------------------- |
| Storage   | Defines where the blobs are stored.           |
| Identity  | Definition of Users and Groups.               |
| RBAC      | Rules linking roles to control access.        |
| Cache     | Configuration for pull-through cache targets. |

Example:

```sh
simple-registry serve \
  -cfgdir ./config \
  -cfgdir ./rbac \
  -cfgdir ./proxies
```

Please, read the [docs](docs) to learn more about the configuration files and
their syntax.

> **ℹ️ Note:**
> There are some manifests examples in [docs/examples](docs/examples)

---

## 🔒 Authentication Model

You can define, using regular expressions, which users and groups
have access to specific repositories.

Simple Registry evaluates requests in three tiers:

1. **Anonymous:** Mixes public and private repositories.
2. **Basic Auth:** Defines users and groups for basic authentication.
   Passwords are hashed using bcrypt.
3. **Bearer Token:** Supports issued tokens with built-in expiration validation.

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
