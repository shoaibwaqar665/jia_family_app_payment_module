# Service Mesh Integration Guide (Envoy + Spiffe)

This document describes how the Payment Service integrates with the service mesh using Envoy and Spiffe for secure service-to-service communication.

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Configuration](#configuration)
4. [Service Discovery](#service-discovery)
5. [mTLS Authentication](#mtls-authentication)
6. [Service-to-Service Communication](#service-to-service-communication)
7. [Deployment](#deployment)
8. [Testing](#testing)
9. [Troubleshooting](#troubleshooting)

## Overview

The Payment Service uses:
- **Envoy Proxy** for service discovery and load balancing
- **Spiffe/Spire** for workload identity and authentication
- **mTLS** for encrypted service-to-service communication

### Key Features

- ✅ Automatic service discovery via Envoy XDS
- ✅ mTLS encryption for all inter-service communication
- ✅ Spiffe ID-based authentication
- ✅ Circuit breaker for resilience
- ✅ Health checks and monitoring
- ✅ Graceful degradation

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Payment Service                           │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  gRPC Server (Port 8081)                              │  │
│  │  - Auth Interceptor (Spiffe + JWT)                    │  │
│  │  - Service Discovery Client                           │  │
│  └───────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  Service Manager                                       │  │
│  │  - Contact Service Client                             │  │
│  │  - Family Service Client                              │  │
│  │  - Document Service Client                            │  │
│  └───────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  gRPC Client Pool (mTLS + Spiffe)                     │  │
│  └───────────────────────────────────────────────────────┘  │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                    Envoy Proxy                               │
│  - XDS Configuration                                        │
│  - Service Discovery                                        │
│  - Load Balancing                                           │
│  - mTLS Termination                                         │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│              Other Services (Contact, Family, etc.)          │
└─────────────────────────────────────────────────────────────┘
```

## Configuration

### Environment Variables

Set these environment variables before starting the service:

```bash
# Service Mesh Configuration
export SERVICE_MESH_ENABLED=true
export SERVICE_MESH_SPIFFE_ID=spiffe://jia.app/payment-service

# Service Discovery
export SERVICE_MESH_DISCOVERY_ENVOY_ADDRESS=localhost:8500
export SERVICE_MESH_DISCOVERY_NAMESPACE=default
export SERVICE_MESH_DISCOVERY_SCHEME=xds

# mTLS Configuration
export MTLS_ENABLED=true
export MTLS_CERT_FILE=/etc/spiffe/certs/payment-service.crt
export MTLS_KEY_FILE=/etc/spiffe/certs/payment-service.key
export MTLS_CA_FILE=/etc/spiffe/certs/ca.crt

# External Services
export EXTERNAL_SERVICES_CONTACT_SERVICE_NAME=contact-service
export EXTERNAL_SERVICES_CONTACT_SERVICE_ADDRESS=contact-service:50051
export EXTERNAL_SERVICES_CONTACT_SERVICE_SPIFFE_ID=spiffe://jia.app/contact-service
```

### config.yaml

```yaml
# Service Mesh Configuration
service_mesh:
  enabled: true
  spiffe_id: "spiffe://jia.app/payment-service"
  
  discovery:
    envoy_address: "localhost:8500"
    namespace: "default"
    scheme: "xds"
    timeout_seconds: 30

# mTLS Configuration
mtls:
  enabled: true
  cert_file: "${MTLS_CERT_FILE}"
  key_file: "${MTLS_KEY_FILE}"
  ca_file: "${MTLS_CA_FILE}"
  min_version: "1.2"

# External Services
external_services:
  contact_service:
    name: "contact-service"
    address: "contact-service:50051"
    spiffe_id: "spiffe://jia.app/contact-service"
    timeout_seconds: 10

# Circuit Breaker
circuit_breaker:
  enabled: true
  failure_threshold: 5
  success_threshold: 2
  timeout_seconds: 60
  half_open_max_calls: 3
```

## Service Discovery

### Envoy XDS Integration

The service uses Envoy's XDS (xDS API) for service discovery:

```go
// Initialize service discovery
discoveryConfig := discovery.DefaultServiceConfig()
discoveryConfig.EnvoyAddress = "localhost:8500"
discoveryConfig.Scheme = "xds"
discoveryConfig.Namespace = "default"

sd, err := discovery.NewServiceDiscovery(discoveryConfig)
if err != nil {
    log.Fatal("Failed to initialize service discovery", err)
}

// Resolve service address
address, err := sd.ResolveService("contact-service")
// Returns: "contact-service:50051" or resolved IP
```

### DNS Fallback

If Envoy is not available, the service falls back to DNS resolution:

```go
// DNS SRV lookup
_, addrs, _ := net.LookupSRV("", "", "contact-service.default.svc.cluster.local")

// DNS A record lookup
addrs, _ := net.LookupHost("contact-service")
```

## mTLS Authentication

### Certificate Setup

1. **Generate Spiffe certificates** using Spire:

```bash
# Start Spire server
spire-server run -config /opt/spire/conf/server/server.conf

# Register payment service workload
spire-server entry create \
  -spiffeID spiffe://jia.app/payment-service \
  -parentID spiffe://jia.app/agent \
  -selector k8s:ns:default \
  -selector k8s:sa:payment-service

# Get SVID (Spiffe Verifiable Identity Document)
spire-agent api fetch jwt \
  -audience contact-service \
  -spiffeID spiffe://jia.app/payment-service
```

2. **Store certificates**:

```bash
# Payment service certificates
/etc/spiffe/certs/payment-service.crt
/etc/spiffe/certs/payment-service.key
/etc/spiffe/certs/ca.crt
```

### Client Configuration

```go
// Create gRPC client with mTLS
clientConfig := grpc.DefaultClientConfig()
clientConfig.Target = "contact-service:50051"
clientConfig.EnableMTLS = true
clientConfig.SpiffeID = "spiffe://jia.app/payment-service"
clientConfig.CertFile = "/etc/spiffe/certs/payment-service.crt"
clientConfig.KeyFile = "/etc/spiffe/certs/payment-service.key"
clientConfig.CAFile = "/etc/spiffe/certs/ca.crt"

client, err := grpc.NewClient(clientConfig)
```

### Server Configuration

The server validates incoming spiffe IDs:

```go
// Enable spiffe validation in auth interceptor
authInterceptor := interceptors.NewAuthInterceptorWithSpiffe([]string{
    "spiffe://jia.app/contact-service",
    "spiffe://jia.app/family-service",
    "spiffe://jia.app/document-service",
})

// Server will accept requests with valid spiffe IDs
```

## Service-to-Service Communication

### Making Calls to Other Services

```go
// Initialize service manager
serviceManager, err := services.NewServiceManager(config, logger)
if err != nil {
    log.Fatal("Failed to create service manager", err)
}

// Initialize all service clients
err = serviceManager.Initialize(ctx)
if err != nil {
    log.Fatal("Failed to initialize services", err)
}

// Get contact service client
contactClient, err := serviceManager.GetContactService()
if err != nil {
    log.Fatal("Failed to get contact service", err)
}

// Make a call
userInfo, err := contactClient.GetUserInfo(ctx, "user_123")
if err != nil {
    log.Error("Failed to get user info", zap.Error(err))
    return
}

log.Info("User info retrieved",
    zap.String("user_id", userInfo.ID),
    zap.String("name", userInfo.Name))
```

### Example: Getting Family Members

```go
// Get family members for a family
members, err := contactClient.GetFamilyMembers(ctx, "family_456")
if err != nil {
    log.Error("Failed to get family members", zap.Error(err))
    return
}

log.Info("Family members retrieved",
    zap.String("family_id", "family_456"),
    zap.Int("count", len(members)))

// Grant entitlements to all family members
for _, member := range members {
    err := entitlementService.GrantEntitlement(ctx, member.ID, "premium_features")
    if err != nil {
        log.Error("Failed to grant entitlement",
            zap.String("user_id", member.ID),
            zap.Error(err))
    }
}
```

## Deployment

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: payment-service
  namespace: default
spec:
  replicas: 3
  selector:
    matchLabels:
      app: payment-service
  template:
    metadata:
      labels:
        app: payment-service
        version: v1
    spec:
      serviceAccountName: payment-service
      containers:
      - name: payment-service
        image: jia/payment-service:latest
        ports:
        - containerPort: 8081
          name: grpc
        env:
        - name: SERVICE_MESH_ENABLED
          value: "true"
        - name: SERVICE_MESH_SPIFFE_ID
          value: "spiffe://jia.app/payment-service"
        - name: MTLS_ENABLED
          value: "true"
        - name: MTLS_CERT_FILE
          value: "/etc/spiffe/certs/server.crt"
        - name: MTLS_KEY_FILE
          value: "/etc/spiffe/certs/server.key"
        - name: MTLS_CA_FILE
          value: "/etc/spiffe/certs/ca.crt"
        volumeMounts:
        - name: spiffe-certs
          mountPath: /etc/spiffe/certs
          readOnly: true
      volumes:
      - name: spiffe-certs
        csi:
          driver: spire.csi.k8s.io
          volumeAttributes:
            spiffeID: spiffe://jia.app/payment-service
---
apiVersion: v1
kind: Service
metadata:
  name: payment-service
  namespace: default
spec:
  selector:
    app: payment-service
  ports:
  - port: 8081
    targetPort: 8081
    protocol: TCP
    name: grpc
```

### Envoy Sidecar Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: envoy-config
data:
  envoy.yaml: |
    static_resources:
      listeners:
      - name: listener_0
        address:
          socket_address:
            address: 0.0.0.0
            port_value: 9901
        filter_chains:
        - filters:
          - name: envoy.filters.network.http_connection_manager
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
              stat_prefix: ingress_http
              codec_type: AUTO
              route_config:
                name: local_route
                virtual_hosts:
                - name: local_service
                  domains: ["*"]
                  routes:
                  - match:
                      prefix: "/"
                    route:
                      cluster: service_payment
              http_filters:
              - name: envoy.filters.http.router
      
      clusters:
      - name: service_payment
        connect_timeout: 0.25s
        type: LOGICAL_DNS
        lb_policy: ROUND_ROBIN
        load_assignment:
          cluster_name: service_payment
          endpoints:
          - lb_endpoints:
            - endpoint:
                address:
                  socket_address:
                    address: payment-service
                    port_value: 8081
        transport_socket:
          name: envoy.transport_sockets.tls
          typed_config:
            "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext
            common_tls_context:
              tls_certificates:
              - certificate_chain:
                  filename: /etc/spiffe/certs/client.crt
                private_key:
                  filename: /etc/spiffe/certs/client.key
              validation_context:
                trusted_ca:
                  filename: /etc/spiffe/certs/ca.crt
```

## Testing

### Test with Mock Services

```go
// Create mock service discovery
mockDiscovery := discovery.NewMockServiceDiscovery(logger)
mockDiscovery.RegisterService("contact-service", "localhost", 50051)

// Create service manager with mock
serviceManager, _ := services.NewServiceManager(config, logger)
serviceManager.discovery = mockDiscovery

// Test service call
contactClient, _ := serviceManager.GetContactService()
userInfo, err := contactClient.GetUserInfo(ctx, "test_user")
```

### Integration Tests

```bash
# Start test services
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
go test -v ./internal/shared/services/... -tags=integration

# Cleanup
docker-compose -f docker-compose.test.yml down
```

## Troubleshooting

### Service Discovery Issues

**Problem**: Cannot resolve service address

```bash
# Check Envoy connectivity
curl http://localhost:8500/config_dump

# Check DNS resolution
nslookup contact-service.default.svc.cluster.local

# Check service discovery logs
kubectl logs -f deployment/payment-service -c envoy
```

**Solution**: Ensure Envoy is running and XDS is configured correctly

### mTLS Certificate Issues

**Problem**: Certificate validation failed

```bash
# Verify certificate
openssl x509 -in /etc/spiffe/certs/payment-service.crt -text -noout

# Check certificate expiry
openssl x509 -in /etc/spiffe/certs/payment-service.crt -noout -dates

# Verify CA certificate
openssl verify -CAfile /etc/spiffe/certs/ca.crt /etc/spiffe/certs/payment-service.crt
```

**Solution**: Regenerate certificates using Spire

### Spiffe ID Validation Issues

**Problem**: Spiffe ID not allowed

```
Error: spiffe ID not allowed: spiffe://jia.app/unknown-service
```

**Solution**: Add the spiffe ID to the allowed list:

```go
authInterceptor.AddAllowedSpiffeID("spiffe://jia.app/unknown-service")
```

### Connection Timeout

**Problem**: gRPC call timeout

```bash
# Check service health
kubectl get pods -l app=contact-service

# Check network connectivity
kubectl exec -it payment-service-pod -- nc -zv contact-service 50051

# Check circuit breaker status
kubectl logs -f deployment/payment-service | grep circuit
```

**Solution**: Increase timeout or check service availability

## Monitoring

### Metrics

The service exposes these metrics:

- `grpc_client_requests_total` - Total gRPC client requests
- `grpc_client_request_duration_seconds` - Request duration
- `grpc_client_requests_failed_total` - Failed requests
- `circuit_breaker_state` - Circuit breaker state (0=closed, 1=open, 2=half-open)

### Health Checks

```bash
# Check service health
curl http://localhost:8081/health

# Check service manager health
curl http://localhost:8081/health/services
```

## Best Practices

1. **Always use service discovery** instead of hardcoded addresses
2. **Enable mTLS** in production environments
3. **Use circuit breakers** for external service calls
4. **Monitor spiffe certificate expiry** and rotate before expiration
5. **Implement retry logic** with exponential backoff
6. **Use health checks** to detect unhealthy services
7. **Log all service-to-service calls** for debugging
8. **Use timeouts** for all external calls

## References

- [Envoy Documentation](https://www.envoyproxy.io/docs/)
- [Spiffe/Spire Documentation](https://spiffe.io/docs/latest/spire-about/)
- [gRPC Service Mesh](https://grpc.io/docs/languages/go/quickstart/)
- [Kubernetes Service Mesh](https://kubernetes.io/docs/concepts/services-networking/service-mesh/)

