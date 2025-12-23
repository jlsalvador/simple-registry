# Pull-Through Cache

simple-registry supports pull-through caching from other registries.

This means that you can pull images from your simple-registry instance, and your
request will be forwarded to one or more upstream registries. The image will be
cached into your simple-registry.

> **ℹ️ Note:**
> simple-registry always pull the image from the upstream to ensure it's
> up-to-date. If upstream is down, simple-registry will return its latest
> cached copy if it exist.

## Configuration

Create your YAML manifests as the following example:

```yaml
# ./config/proxies.yaml
---
apiVersion: simple-registry.jlsalvador.online/v1beta1
kind: PullThroughCache
metadata:
  name: docker-io
spec:
  upstream:
    url: https://registry-1.docker.io
    timeout: 60s
    # # CAUTION!
    # # Private resources that this user has access to upstream is made available
    # # on your mirror.
    # username: your-docker-username
    # password: your-plain-docker-password
    # # Or you can store your plain password in a file
    # passwordFile: /run/secrets/dockerhub-password
    ttl: 30d
  scopes:
   - "^library/.+$"
   - "^miniflux/miniflux(:.+)?$"
---
apiVersion: simple-registry.jlsalvador.online/v1beta1
kind: PullThroughCache
metadata:
  name: ghcr-io
spec:
  upstream:
    url: https://ghcr.io/
    timeout: 60s
    username: your-github-user
    password: your-plain-github-password
    ttl: 30d
  scopes:
   - "^my-github-user/.+$"
```

> ⚠️ **Security note:**
> When pull-through caching is enabled with upstream credentials, all repositories
> accessible by those credentials may become available through this registry.
> Use dedicated upstream accounts with minimal permissions.

Later, launch simple-registry with the flag `-cfgdir` pointing to the directory
containing your YAML files:

```sh
simple-registry serve \
  -datadir ./data \
  -adminpwd secret \
  -cfgdir ./config
```

## Client example

```sh
docker pull localhost:5000/library/busybox
docker pull localhost:5000/my-github-user/my-repo
docker pull localhost:5000/my-github-user/my-another-project
```
