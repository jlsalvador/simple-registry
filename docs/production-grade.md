# Production Grade

You can follow these steps to ensure your simple-registry is production-grade.

## Password Hash

It's recommended to use password hashes for your users.

You can generate a secure password hash running `simple-registry genhash`:

```sh
# Generate a secure password hash for the -adminpwdfile flag:
simple-registry genhash > ./admin-password.txt

# Launch your instance with the generated password hash:
simple-registry serve \
    -datadir ./data \
    -adminpwdfile ./admin-password.txt
```

## HTTPS and certificates

Clients requires HTTPS and a valid TLS certificate to connect.
For production, use the `-cert` and `-key` flags to enable HTTPS.

> **ℹ️ Note:**
> Docker client will accept HTTP for localhost connections.

You can use your own TLS certificates or generate a self-signed one for
testing purposes *(requires client configuration to skip tls-verification).

```sh
# Generate a self-signed TLS certificate for testing purposes:
openssl req -new -x509 \
    -keyout tls.key \
    -out tls.crt \
    -days 36500 \
    -nodes -subj "/C=SE/ST=ES/L=Sevilla/O=ACME/CN=localhost"
```

```sh
# Launch your simple-registry instance listening at https://0.0.0.0:5000
simple-registry serve \
  -datadir ./data \
  -adminpwdfile ./admin-password.txt \
  -cert tls.crt \
  -key tls.key
```

> **ℹ️ Note:**
> Podman client can skip TLS verification with the flag `--tls-verify=false`.

```sh
podman login --tls-verify=false localhost:5000
podman push --tls-verify=false localhost:5000/library/busybox:latest
```

## YAML Manifests

Instead of multiples flags, we recommend using YAML manifests to configure your
simple-registry instance.

You can split/merge your YAML manifests into multiple files and directories.

Here are some examples:

```yaml
# rbac/users.yaml
---
apiVersion: simple-registry.jlsalvador.online/v1beta1
kind: User
metadata:
  name: admin
spec:
  # Generate a password hash using `simple-registry -genhash`
  passwordHash: $2a$10$GsxTxNCV6Tv9lm9em287xOdRzE7VlbhI0EVRSvZFOq/cCSU6eJuWK # simple-registry
  groups: [admins]
---
apiVersion: simple-registry.jlsalvador.online/v1beta1
kind: User
metadata:
  name: user1
  passwordHash: $2a$10$imG59lWpbGj/MaDeheh/CuSYSOFdaVw.aw1GuaUqPHVf7sQc14rbi # password
spec:
  groups: [dev]
---
apiVersion: simple-registry.jlsalvador.online/v1beta1
kind: User
metadata:
  name: anonymous
spec:
  groups: [public]
```

```yaml
# rbac/roles.yaml
---
apiVersion: simple-registry.jlsalvador.online/v1beta1
kind: Role
metadata:
  name: admins
spec:
  resources:
  - "*"
  verbs:
  - "*"
---
apiVersion: simple-registry.jlsalvador.online/v1beta1
kind: Role
metadata:
  name: readwrite
spec:
  resources:
  - catalog
  - blobs
  - manifests
  verbs:
  - HEAD
  - GET
  - POST
  - PUT
  - PATCH
---
apiVersion: simple-registry.jlsalvador.online/v1beta1
kind: Role
metadata:
  name: readonly
spec:
  resources:
  - catalog
  - blobs
  - manifests
  verbs:
  - HEAD
  - GET
```

```yaml
# rbac/rolebindins.yaml
---
apiVersion: simple-registry.jlsalvador.online/v1beta1
kind: RoleBinding
metadata:
  name: admins
spec:
  subjects:
    - kind: Group
      name: admins
  roleRef:
    name: admins
  scopes: ["^.*$"]
---
apiVersion: simple-registry.jlsalvador.online/v1beta1
kind: RoleBinding
metadata:
  name: devs
spec:
  subjects:
    - kind: Group
      name: dev
  roleRef:
    name: readwrite
  scopes: ["^dev/.+$"]
---
apiVersion: simple-registry.jlsalvador.online/v1beta1
kind: RoleBinding
metadata:
  name: public-catalog
spec:
  subjects:
    - kind: Group
      name: public
  roleRef:
    name: readonly
  scopes: ["^$"]
---
apiVersion: simple-registry.jlsalvador.online/v1beta1
kind: RoleBinding
metadata:
  name: public-library
spec:
  subjects:
    - kind: Group
      name: public
  roleRef:
    name: readonly
  scopes: ["^library/.*$"]
```

```yaml
# ./proxies/docker-io.yaml
---
apiVersion: simple-registry.jlsalvador.online/v1beta1
kind: PullThroughCache
metadata:
  name: docker-io
spec:
  upstream:
    url: https://registry-1.docker.io
    timeout: 60s
    ttl: 30d
  scopes:
   - "^library/.+$"
```

```sh
simple-registry serve \
    -datadir ./data \
    -cfgdir ./rbac \
    -cfgdir ./proxies
```
