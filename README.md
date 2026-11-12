# node-min-lifetime

`node-min-lifetime` is a Kubernetes controller that prevents nodes from being scaled down before they reach a configured minimum lifetime.

It manages the Cluster Autoscaler / Karpenter annotations:

```text
cluster-autoscaler.kubernetes.io/scale-down-disabled
karpenter.sh/do-not-disrupt
```

When a node is younger than its configured minimum lifetime, the controller sets this annotation to `"true"`. Once the node reaches the minimum lifetime, the annotation is removed so the Cluster Autoscaler can scale the node down normally.

This is useful for cloud providers that charge per hour of usage instead of minute, making it more efficient to keep nodes alive for some minimum amount of time.

## Features

- Watches Kubernetes `Node` resources
- Supports a default minimum node lifetime
- Supports per-node lifetime overrides
- Supports filtering nodes with a Kubernetes label selector
- Supports explicit per-node scale-down enable/disable overrides

## How it works

For each matching node, the controller:

1. Reads the node creation timestamp.
2. Determines the minimum lifetime for the node.
3. Compares the node age with the configured minimum lifetime.
4. Adds or removes the Cluster Autoscaler scale-down annotation.

If the node is too young, the controller applies:

```yaml
cluster-autoscaler.kubernetes.io/scale-down-disabled: "true"
karpenter.sh/do-not-disrupt: "true"
```

When the node is old enough, the annotation is removed.

## Configuration

Configuration can be provided with command-line flags or environment variables.

### Command-line flags

| Flag | Default                         | Description |
| --- |---------------------------------| --- |
| `--metrics-bind-address` | `:8080`                         | Address used for the metrics endpoint. |
| `--health-probe-bind-address` | `:8081`                         | Address used for health and readiness probes. |
| `--node-selector` | empty                           | Optional Kubernetes label selector limiting which nodes are managed. |
| `--default-min-lifetime` | `45m`                           | Default minimum lifetime for nodes. |
| `--leader-elect` | `false`                         | Enables leader election. |
| `--leader-elect-id` | `node-min-lifetime.eplight.org` | Leader election ID. |

### Environment variables

Environment variables use the same names as the command-line flags, converted to uppercase with underscores.

Examples:

```bash
DEFAULT_MIN_LIFETIME=2h
NODE_SELECTOR=nodepool=default
LEADER_ELECT=true
METRICS_BIND_ADDRESS=:8080
HEALTH_PROBE_BIND_ADDRESS=:8081
```

## Node annotations and labels

### Minimum lifetime override

You can override the default minimum lifetime for a specific node with an annotation or label.

Annotation:

```yaml
node-min-lifetime.eplight.org/min-lifetime: "2h"
```

Label:

```yaml
node-min-lifetime.eplight.org/min-lifetime: "2h"
```

Duration values use Go duration syntax, for example:

- `30m`
- `1h`
- `2h30m`

### Explicit scale-down override

You can explicitly control scale-down protection for a specific node.

Annotation:

```yaml
node-min-lifetime.eplight.org/scale-down-disabled: "true"
```

Label:

```yaml
node-min-lifetime.eplight.org/scale-down-disabled: "true"
```

Supported values:

| Value | Behavior |
| --- | --- |
| `"true"` | Force scale-down protection. |
| `"false"` | Do not protect the node from scale-down. |
| unset | Use minimum lifetime logic. |

## Running locally

### Prerequisites

- Go 1.26+
- Access to a Kubernetes cluster
- A valid kubeconfig with permissions to read and update nodes

### Run

```bash
go run ./cmd/node-min-lifetime
```

Run with custom options:

```bash
go run ./cmd/node-min-lifetime \
  --default-min-lifetime=2h \
  --node-selector=nodepool=default
```

## Building

Build the binary locally:

```bash
go build -o node-min-lifetime ./cmd/node-min-lifetime
```

## Docker

Build the container image:

```bash
docker build -t node-min-lifetime:local .
```

## Kubernetes deployment

Apply the default manifest:

```bash
kubectl apply -f manifests/default.yaml
```

The default deployment uses the image:

```text
ghcr.io/eplightning/node-min-lifetime:v0.2.0
```

To regenerate the manifest from the kustomize configuration:

```bash
make k8smanifests
```

## Health checks

| Endpoint | Default address | Description |
| --- | --- | --- |
| `/healthz` | `:8081` | Liveness check. |
| `/readyz` | `:8081` | Readiness check. |

## Metrics

Metrics are exposed on the configured metrics bind address.

Default:

```text
:8080
```

## RBAC

The controller needs permissions to watch and update Kubernetes nodes.

Required node permissions include:

- `get`
- `list`
- `watch`
- `patch`
- `update`

It also requires standard leader-election permissions when leader election is enabled.

## Examples

Protect all managed nodes from scale-down for at least 2 hours:

```bash
node-min-lifetime --default-min-lifetime=2h
```

Protect only nodes matching a selector:

```bash
node-min-lifetime \
  --default-min-lifetime=2h \
  --node-selector=nodepool=worker
```

Override the minimum lifetime on a specific node:

```bash
kubectl annotate node my-node \
  node-min-lifetime.eplight.org/min-lifetime=4h
```

Force scale-down protection on a node:

```bash
kubectl annotate node my-node \
  node-min-lifetime.eplight.org/scale-down-disabled=true
```

Disable scale-down protection for a node:

```bash
kubectl annotate node my-node \
  node-min-lifetime.eplight.org/scale-down-disabled=false
```

## Development

Download dependencies:

```bash
go mod download
```

Run the application:

```bash
go run ./cmd/node-min-lifetime
```

Build:

```bash
go build ./cmd/node-min-lifetime
```

Regenerate Kubernetes manifests:

```bash
make k8smanifests
```
