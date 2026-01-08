# Simple Gateway API Example

This example demonstrates a basic Gateway API setup with:

- **GatewayClass** - Defines the gateway controller
- **Gateway** - Configures HTTP (port 80) and HTTPS (port 443) listeners
- **HTTPRoute** - Routes traffic based on path prefixes

## Resources

| Resource | Name | Description |
|----------|------|-------------|
| GatewayClass | `example-gateway-class` | References `example.com/gateway-controller` |
| Gateway | `example-gateway` | Listens on ports 80 (HTTP) and 443 (HTTPS) |
| HTTPRoute | `example-route` | Routes `/api` to `api-service:8080`, `/` to `frontend-service:80` |

## Generate YAML

```bash
pkl eval gateway.pkl -f yaml > gateway.yaml
```

Or from the project root:

```bash
make examples
```

## Output

The generated YAML contains all three resources as a stream (separated by `---`), ready for `kubectl apply -f`.
