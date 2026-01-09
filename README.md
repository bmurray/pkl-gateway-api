# pkl-gateway-api

Pkl templates for [Kubernetes Gateway API](https://gateway-api.sigs.k8s.io/) resources.

## Overview

Type-safe Pkl modules for Gateway API resources like HTTPRoute, Gateway, GatewayClass, and more. Generated from the official Gateway API CRDs.

## Installation

### Requirements

- [Pkl](https://pkl-lang.org/) 0.25.0+

### Add to Your PklProject

Add both `k8s` and `gateway-api` to your `PklProject` file:

```pkl
amends "pkl:Project"

dependencies {
  ["k8s"] {
    uri = "package://pkg.pkl-lang.org/pkl-k8s/k8s@1.3.0"
  }
  ["gateway-api"] {
    uri = "package://pkg.pkl-lang.org/github.com/bmurray/pkl-gateway-api/gateway-api@1.4.1"
  }
}
```

Then resolve dependencies:

```bash
pkl project resolve
```

## Usage

### Create Resources Using Shortname Imports

Example HTTPRoute configuration using the `@gateway-api` shortname:

```pkl
amends "@gateway-api/gateway.networking.k8s.io/v1/HTTPRoute.pkl"

metadata {
  name = "my-route"
  namespace = "default"
}

spec {
  parentRefs {
    new {
      name = "my-gateway"
    }
  }
  hostnames {
    "example.com"
  }
  rules {
    new {
      matches {
        new {
          path {
            type = "PathPrefix"
            value = "/api"
          }
        }
      }
      backendRefs {
        new {
          name = "my-service"
          port = 8080
        }
      }
    }
  }
}
```

### Render to YAML

```bash
pkl eval my-route.pkl -f yaml
```

Output:

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: my-route
  namespace: default
spec:
  parentRefs:
    - name: my-gateway
  hostnames:
    - example.com
  rules:
    - matches:
        - path:
            type: PathPrefix
            value: /api
      backendRefs:
        - name: my-service
          port: 8080
```

## Supported Resources

### Standard (GA/Beta)

| Resource | API Versions |
|----------|--------------|
| Gateway | v1, v1beta1 |
| GatewayClass | v1, v1beta1 |
| HTTPRoute | v1, v1beta1 |
| GRPCRoute | v1 |
| ReferenceGrant | v1beta1 |
| BackendTLSPolicy | v1 |

### Experimental

| Resource | API Versions |
|----------|--------------|
| TCPRoute | v1alpha2 |
| TLSRoute | v1alpha2 |
| UDPRoute | v1alpha2 |

## Versioning

The package URI follows this format:
```
package://pkg.pkl-lang.org/github.com/bmurray/pkl-gateway-api/gateway-api@{version}
```

The version corresponds to the Gateway API version the templates are generated from (e.g., `1.4.1`).

## Development

### Requirements

- Go 1.22+
- [Pkl](https://pkl-lang.org/) 0.25.0+

### Generating Templates

```bash
# Generate from Gateway API v1.4.1 (default)
make generate

# Generate from a specific Gateway API version
make generate VERSION=v1.3.0

# Include experimental resources (TCPRoute, TLSRoute, UDPRoute)
make generate VERSION=v1.4.1 EXPERIMENTAL=true
```

### Project Structure

```
pkl-gateway-api/
├── cmd/generate/       # Code generator CLI
├── internal/
│   ├── crd/            # CRD YAML parsing
│   ├── schema/         # OpenAPI to internal model conversion
│   └── generator/      # Pkl module generation
├── templates/          # Base Pkl templates
├── generated-package/  # Generated output (gitignored)
└── Makefile
```

## License

Apache License 2.0
