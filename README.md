# 📦 Simple Registry

A lightweight OCI-compatible container registry with RBAC support
and pull-through caching.

![Simple Registry Web App User Interface](./docs/screenshots/landing.png)

---

## ✨ Features

- **🎖️ OCI Native:** Implements the [OCI Distribution Specification v1.1.1][oci-spec].
- **🪶 Lightweight:** Low memory footprint and minimal dependencies.
- **🛂 Role-Based Access Control (RBAC):** Per repository, action, and role.
- **📦 Pull-through Caching:** Configurable on-demand caching from external registries.
- **🌐 Web User Interface:** Optional built-in browser-only, for now.
- **🔒 Flexible Authentication:** Anonymous, Basic Auth, and tokens.
- **♻️ Garbage Collection:** On-demand cleanup of unused layers.
- **🌀 Stateless & Scalable:** Horizontal scaling backed by shared storage.

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

> **ℹ️ Note:**
> You can see more options by running `simple-registry serve -h`.

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

## 🐳 Quick Deployment with Docker Compose

The easiest way to run Simple Registry with persistent storage:

1. **Create a [`compose.yaml`](./compose.yaml) file**
2. **Launch it:**

```sh
docker compose up -d
```

---

## ⚙️ Configuration

The registry can be configured via YAML manifests.
You can split your configuration into multiple files and directories using the
`-cfgdir` flag.

Here are a few components that can be configured:

| Component       | Description                                   |
| --------------- | --------------------------------------------- |
| Configuration   | Defines Simple Registry's behavior.           |
| Identity & RBAC | Rules linking roles to users and groups.      |
| Cache           | Configuration for pull-through cache targets. |

Example:

```sh
simple-registry serve \
  -cfgdir ./config \
  -cfgdir ./rbac \
  -cfgdir ./proxies
```

Please, read the [docs](docs) to learn more about each configuration and their
syntax:

- [Role-Based Access Control](docs/role-based-access-control.md)
- [Production-grade guide](docs/production-grade.md)
- [Pull-Through Cache](docs/pull-through-cache.md)

> **ℹ️ Note:**
> There are some manifests examples in [docs/examples](docs/examples)

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
