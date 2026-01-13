# Garbage Collection

The `garbage-collect` command is a maintenance utility for `simple-registry`
designed to reclaim storage space by removing **blobs** and **manifests** that
are no longer referenced.

---

## Usage Examples

### Run a simulation (Dry Run)

Check the resources that would be deleted without actually deleting files:

```sh
simple-registry garbage-collect --datadir /path/to/data --dryrun
```

### Deep cleanup (Delete untagged images)

Remove images that no longer have a tag pointing to them:

```sh
simple-registry garbage-collect --datadir /path/to/data --delete-untagged
```

---

## Configuration & Flags

The command accepts both command-line flags and environment variables
(using the `SIMPLE_REGISTRY_` prefix).

| Flag                | Description                                         |
| ------------------- | --------------------------------------------------- |
| `--datadir`         | Required. Path to the data directory.               |
| `--cfgdir`          | Optional. Directory containing YAML files.          |
| `--dryrun`          | If enabled, simulates removing files.               |
| `--delete-untagged` | If enabled, manifests without tags will be deleted. |
| `--last-access`     | Optional. Minimum last access time to keep objects. |

---

## Logging and Output

The command provides detailed logs at the `DEBUG` level for each item removed
and an `INFO` summary at the end:

* **Manifests marked/deleted**: Number of manifest files processed.
* **Blobs marked/deleted**: Number of layer files processed.

---

## Technical Workflow

### 1. Root Collection

The process starts by determining which manifests must be preserved:

* **If `--delete-untagged` is `false`**:
  All manifests found in the repositories are considered roots.
* **If `--delete-untagged` is `true`**:
  Only manifests that are currently pointed to by at least one tag are
  considered roots.

### 2. Mark Phase (Traversal)

Starting from the roots, the collector inspects the content of each manifest to
mark its dependencies. It supports the following media types:

* **OCI Image Manifest** & **Docker V2**:
  Marks the image configuration and all filesystem layers.
* **OCI Index** & **Docker Manifest List**:
  Recursively marks all manifests included in the list.
* **Docker V1**:
  Marks the `FSLayers`.
* **Referrers**:
  Artifacts referencing the analyzed manifests.

> **ℹ️ Note:**
> If a **Pull-Through Cache** is configured, the garbage collector operates
> only on local data. It will not trigger a "mirror" (download) from the
> upstream registry during the marking phase.

### 3. Sweep Phase (Cleanup)

The collector compares the "marked" set against the actual files on disk.
An object is deleted only if:

1. It is **not marked** as "in-use."
2. Its **last access time** is older than the duration specified in
   `--last-access`.
