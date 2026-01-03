# Role-Based Access Control (RBAC)

This document describes the **Role-Based Access Control (RBAC)** system used by
**Simple Registry**, how it is configured using YAML files, and how the
registry evaluates permissions for incoming requests.

---

## Overview

RBAC in **Simple Registry** is built around four core concepts:

1. **Users:** Authenticated identities.
2. **Groups:** Collections of users.
3. **Roles:** Sets of permissions (verbs + resources).
4. **RoleBindings:** Associations between users or groups, roles, and scopes.

For every HTTP request, the registry:

1. Authenticates the user.
2. Resolves the user’s groups.
3. Collects all applicable RoleBindings.
4. Checks whether at least one bound role authorizes the requested action on
   the target resource and repository.

If any role matches, the request is allowed.

---

## Users

Users are defined using the `User` resource.

### User Example

```yaml
apiVersion: simple-registry.jlsalvador.online/v1beta1
kind: User
metadata:
  name: admin
spec:
  passwordHash: $2a$10$GsxTxNCV6Tv9lm9em287xOdRzE7VlbhI0EVRSvZFOq/cCSU6eJuWK
  groups:
  - admins
```

> **ℹ️ Note:**
> [See more User manifests examples here](./examples/users.yaml).

### User Fields

- **metadata.name**
  The username. This is the identity used during HTTP authentication.

- **spec.passwordHash**
  A bcrypt password hash.
  It can be generated using:

  ```sh
  simple-registry -genhash
  ```

- **spec.groups**
  List of groups the user belongs to.

---

### Anonymous user

```yaml
kind: User
metadata:
  name: anonymous
spec:
  groups:
  - public
```

The `anonymous` user represents unauthenticated access and allows you to define
which parts of the registry are publicly accessible.

---

## Roles

A **Role** defines *what actions are allowed*, but not *who* can perform them
or *where*.

### Roles Example

```yaml
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

> **ℹ️ Note:**
> [See more Role manifests examples here](./examples/roles.yaml).

### Roles Fields

- **metadata.name**
  The role identifier.

- **spec.resources**
  The resources the role applies to.

  Common resources:

  - `catalog`
  - `blobs`
  - `manifests`

  The wildcard `"*"` matches all resources.

- **spec.verbs**
  Allowed HTTP methods:

  - `GET`, `HEAD` -> read access
  - `POST`, `PUT`, `PATCH` -> write access
  - `"*"` -> all verbs

---

### Roles defined in the examples

#### `admins`

```yaml
resources: ["*"]
verbs: ["*"]
```

Full administrative access.

---

#### `readwrite`

```yaml
resources: [catalog, blobs, manifests]
verbs: [HEAD, GET, POST, PUT, PATCH]
```

Allows:

- Pulling images
- Pushing images
- Uploading blobs and manifests

This example does **not** allow deletion.

---

#### `readonly`

```yaml
resources: [catalog, blobs, manifests]
verbs: [HEAD, GET]
```

Read-only access (pull only).

---

## RoleBindings

A **RoleBinding** connects:

- **who** (subjects)
- **what** (role)
- **where** (scope)

### RoleBindings Example

```yaml
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

> **ℹ️ Note:**
> [See more RoleBinding manifests examples here](./examples/rolebindings.yaml).

---

### RoleBindings Fields

#### `spec.subjects`

Defines who the role applies to.

```yaml
subjects:
  - kind: Group
    name: public
```

- **kind**: `User` or `Group`
- **name**: name of the user or group

---

#### `spec.roleRef`

The referenced role.

```yaml
roleRef:
  name: readonly
```

---

#### `spec.scopes`

A list of **Go regular expressions** that limit the repositories to which the
role applies.

```yaml
scopes:
  - "^library/.*$"
```

The repository name must match the regexp for the role to be considered.

---

## Using `scopes` with regular expressions

Scopes are Go regular expressions evaluated against the repository name.

### Common examples

| Scope                     | Meaning                                   |
| ------------------------- | ----------------------------------------- |
| `^$`                      | List of repositories (catalog)            |
| `^.+$`                    | All repositories                          |
| `^.*$`                    | Catalog and all repositories              |
| `^library/.*$`            | Public Docker Hub repositories            |
| `^myorg/.+$`              | All prefix `myorg/`                       |
| `^myorg/(app\|infra)-.+$` | All prefix `myorg/app-` or `myorg/infra-` |
| `^team1/project(:.+)?$`   | Project with any optionally tag           |
| `^team1/[^/]+$`           | Exactly one level under `team1/`          |

---

### Special case: catalog access

```yaml
scopes: ["^$"]
```

This scope is used for requests to `/v2/_catalog`, which are not associated
with a specific repository.

This is useful for allowing anonymous users to list the catalog without
requiring authentication.

Every listed repository will be checked against the RBAC rules. So only the
allowed repositories for the request's user will be returned.

---

## Explained examples

### Public access to the catalog only

```yaml
metadata:
  name: public-catalog
spec:
  subjects:
    - kind: Group
      name: public
  roleRef:
    name: readonly
  scopes: ["^$"]
```

- Applies to anonymous users
- Allows listing the catalog
- Does not allow access to individual repositories

---

### Public access to official images

```yaml
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

Allows:

```sh
docker pull example.com/library/nginx
```

But not:

```sh
docker push example.com/library/nginx
```

---

### Global administrators

```yaml
metadata:
  name: admins
spec:
  subjects:
    - kind: Group
      name: admins
  roleRef:
    name: admins
  scopes: ["^.*$"]
```

Full access to all repositories and resources.

---

### ⚠️ Dangerous example

```yaml
# This rolebinding allows any user to do anything.
apiVersion: simple-registry.jlsalvador.online/v1beta1
kind: RoleBinding
metadata:
  name: dmz
spec:
  subjects:
    - kind: Group
      name: public
  roleRef:
    name: admins
  scopes: ["^.*$"]
```

Binding the `admins` role to the `public` group with a global scope effectively
makes the registry completely open.

---

## How a request is evaluated

For every incoming HTTP request:

1. The user is authenticated (`admin`, `anonymous`, etc.).
2. The user’s groups are resolved.
3. All matching RoleBindings are collected:
   - subject matches
   - scope regexp matches the repository
4. The registry checks whether any bound role allows:
   - the target resource (`blobs`, `manifests`, etc.)
   - the HTTP verb (`GET`, `POST`, etc.)

If **at least one role authorizes the request**, access is granted.

---

## Best practices

- Use `readonly` roles for public access.
- Keep scopes as narrow as possible.
- Avoid `"*"` except for administrators.
- Separate read-only and read-write roles.
- Always test behavior using the `anonymous` user.
- You can merge multiple manifests in a single YAML file concatenating them
  with `---`. See [one example here](./examples/users.yaml).
- You can use multiples directories to store multiple manifests. Use the flag
  `-cfgdir` multiple times for each directory.

  ```sh
  simple-registry server -cfgdir ./users -cfgdir ./roles
  ```

---

## Summary

RBAC in **Simple Registry** provides:

- Fine-grained control per **user**, **group**, **repository**, and **action**
- Powerful scoping via regular expressions
- Declarative YAML-based configuration
- Secure defaults with explicit opt-in exposure

This model supports anything from fully private registries to public mirrors
with precise access control.
