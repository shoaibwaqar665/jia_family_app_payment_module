# API Gateway Architecture

## Overview

The Payment Service uses an **API Gateway** pattern to secure and manage all external traffic. The gRPC endpoint is **NOT exposed directly** to external clients.

## Architecture Diagram

```
┌──────────────────────────────────────────────────────────────────┐
│                        External Clients                           │
│  (Web Apps, Mobile Apps, Third-party Services)                   │
└───────────────────────────┬──────────────────────────────────────┘
                            │
                            │ HTTPS/TLS
                            ▼
┌──────────────────────────────────────────────────────────────────┐
│                      API Gateway (Envoy/Kong)                     │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  Authentication & Authorization                             │  │
│  │  - JWT Validation                                           │  │
│  │  - API Key Validation                                       │  │
│  │  - Rate Limiting                                            │  │
│  │  - Request Logging                                          │  │
│  └────────────────────────────────────────────────────────────┘  │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  Routing & Load Balancing                                   │  │
│  │  - Service Discovery                                        │  │
│  │  - Health Checks                                            │  │
│  │  - Circuit Breaker                                          │  │
│  │  - Retry Logic                                              │  │
│  └────────────────────────────────────────────────────────────┘  │
└───────────────────────────┬──────────────────────────────────────┘
                            │
                            │ mTLS + Spiffe ID
                            ▼
┌──────────────────────────────────────────────────────────────────┐
│                  Payment Service (Internal)                       │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  gRPC Server (Port 8081) - NOT EXPOSED                     │  │
│  │  - Auth Interceptor (Spiffe + JWT)                         │  │
│  │  - Service Discovery Client                                 │  │
│  └────────────────────────────────────────────────────────────┘  │
│                          │                                        │
│                          ▼                                        │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  Service Manager                                            │  │
│  │  - Contact Service Client                                   │  │
│  │  - Family Service Client                                    │  │
│  │  - Document Service Client                                  │  │
│  └────────────────────────────────────────────────────────────┘  │
│                          │                                        │
│                          ▼                                        │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  gRPC Client Pool (mTLS + Spiffe)                          │  │
│  └────────────────────────────────────────────────────────────┘  │
└───────────────────────────┬──────────────────────────────────────┘
                            │
                            │ mTLS + Spiffe ID
                            ▼
┌──────────────────────────────────────────────────────────────────┐
│            Other Services (Contact, Family, etc.)                 │
└──────────────────────────────────────────────────────────────────┘
```

## Key Principles

### 1. **No Direct gRPC Exposure**
- ❌ **DO NOT** expose gRPC endpoint directly to external clients
- ✅ **DO** route all traffic through API Gateway
- ✅ **DO** use mTLS for internal service-to-service communication

### 2. **API Gateway Responsibilities**
- **Authentication**: Validate JWT tokens and API keys
- **Authorization**: Check user permissions
- **Rate Limiting**: Prevent abuse
- **Request Routing**: Route to appropriate services
- **Load Balancing**: Distribute traffic
- **Monitoring**: Log all requests
- **Circuit Breaking**: Prevent cascading failures

### 3. **Internal Service Mesh**
- Services communicate via **mTLS + Spiffe**
- **Envoy proxy** handles service discovery
- **No external exposure** of internal services

## Configuration

### API Gateway Settings

```yaml
# API Gateway Configuration
api_gateway:
  # Gateway address (all external traffic goes through gateway)
  address: "api-gateway:8080"
  
  # Enable TLS for gateway communication
  enable_tls: true
  
  # API Key for gateway authentication
  api_key: "${API_GATEWAY_KEY}"
  
  # Gateway TLS certificates (optional, for mutual TLS)
  cert_file: "${GATEWAY_CERT_FILE}"
  key_file: "${GATEWAY_KEY_FILE}"
  ca_file: "${GATEWAY_CA_FILE}"
  
  # Timeouts
  dial_timeout_seconds: 30
  request_timeout_seconds: 30
```

### gRPC Server Settings

```yaml
grpc:
  # Internal gRPC address (NOT exposed externally)
  address: ":8081"
  
  # Enable gRPC reflection (disable in production)
  enable_reflection: false
```

### Environment Variables

```bash
# API Gateway
export API_GATEWAY_ADDRESS=api-gateway:8080
export API_GATEWAY_KEY=your-api-key-here
export API_GATEWAY_ENABLE_TLS=true

# Gateway TLS (optional)
export GATEWAY_CERT_FILE=/etc/gateway/certs/client.crt
export GATEWAY_KEY_FILE=/etc/gateway/certs/client.key
export GATEWAY_CA_FILE=/etc/gateway/certs/ca.crt
```

## API Gateway Implementation Options

### Option 1: Envoy Proxy (Recommended)

```yaml
# envoy-gateway.yaml
static_resources:
  listeners:
  - name: listener_0
    address:
      socket_address:
        address: 0.0.0.0
        port_value: 8080
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: ingress_http
          codec_type: AUTO
          
          # JWT Authentication
          http_filters:
          - name: envoy.filters.http.jwt_authn
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication
              providers:
                auth0:
                  issuer: https://your-auth-domain.com
                  audiences:
                  - your-api-audience
                  remote_jwks:
                    http_uri:
                      uri: https://your-auth-domain.com/.well-known/jwks.json
                      cluster: auth_service
                      timeout: 5s
                  from_headers:
                  - name: Authorization
                    value_prefix: "Bearer "
          
          # Rate Limiting
          - name: envoy.filters.http.ratelimit
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.ratelimit.v3.RateLimit
              domain: payments
              rate_limit_service:
                grpc_service:
                  envoy_grpc:
                    cluster_name: rate_limit_cluster
          
          # Router
          - name: envoy.filters.http.router
          
          route_config:
            name: local_route
            virtual_hosts:
            - name: payment_service
              domains: ["*"]
              routes:
              # Payment Service Routes
              - match:
                  prefix: "/payment.v1.PaymentService/"
                route:
                  cluster: payment_service
                  timeout: 30s
                metadata:
                  filter_metadata:
                    envoy.filters.http.jwt_authn:
                      payload:
                        sub: "{.sub}"
                        email: "{.email}"
      
      clusters:
      # Payment Service Cluster
      - name: payment_service
        connect_timeout: 0.25s
        type: LOGICAL_DNS
        lb_policy: ROUND_ROBIN
        load_assignment:
          cluster_name: payment_service
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

### Option 2: Kong API Gateway

```yaml
# kong-config.yaml
_format_version: "3.0"

services:
  - name: payment-service
    url: https://payment-service:8081
    protocol: grpc
    tls_verify: true
    tls_verify_depth: 1
    ca_certificates:
      - /etc/kong/certs/ca.crt
    routes:
      - name: payment-routes
        paths:
          - /payment.v1.PaymentService/
        strip_path: false
        preserve_host: true
    plugins:
      # JWT Authentication
      - name: jwt
        config:
          secret_is_base64: false
          uri_param_names:
            - token
          claims_to_verify:
            - exp
            - iat
      # Rate Limiting
      - name: rate-limiting
        config:
          minute: 100
          hour: 1000
          policy: local
      # Request Logging
      - name: file-log
        config:
          path: /var/log/kong/payment-requests.log
          reopen: true
```

### Option 3: Traefik

```yaml
# traefik-config.yaml
api:
  dashboard: true
  insecure: false

entryPoints:
  web:
    address: ":8080"
    http:
      tls:
        options: default

providers:
  kubernetes:
    namespaces:
      - default

tls:
  options:
    default:
      minVersion: "VersionTLS12"
      sslProtocols:
        - "TLSv1.2"
        - "TLSv1.3"

# Middleware for JWT
http:
  middlewares:
    jwt-auth:
      plugin:
        jwt:
          jwksUrl: "https://your-auth-domain.com/.well-known/jwks.json"
          issuer: "https://your-auth-domain.com"
          audiences:
            - "your-api-audience"
  
  routers:
    payment-service:
      rule: "PathPrefix(`/payment.v1.PaymentService/`)"
      service: payment-service
      middlewares:
        - jwt-auth
      tls:
        options: default
  
  services:
    payment-service:
      loadBalancer:
        servers:
          - url: "grpcs://payment-service:8081"
```

## Security Considerations

### 1. **Network Segmentation**

```yaml
# Kubernetes Network Policies
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: payment-service-policy
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: payment-service
  policyTypes:
  - Ingress
  - Egress
  ingress:
  # Only allow traffic from API Gateway
  - from:
    - podSelector:
        matchLabels:
          app: api-gateway
    ports:
    - protocol: TCP
      port: 8081
  egress:
  # Allow egress to other services
  - to:
    - podSelector:
        matchLabels:
          app: contact-service
    ports:
    - protocol: TCP
      port: 50051
```

### 2. **TLS Configuration**

```go
// API Gateway TLS config
tlsConfig := &tls.Config{
    MinVersion: tls.VersionTLS12,
    CipherSuites: []uint16{
        tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
        tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
    },
    PreferServerCipherSuites: true,
    CurvePreferences: []tls.CurveID{
        tls.CurveP256,
        tls.CurveP384,
        tls.CurveP521,
    },
}
```

### 3. **Authentication Flow**

```
1. Client → API Gateway
   - Request with JWT token in Authorization header
   
2. API Gateway
   - Validates JWT signature
   - Checks token expiration
   - Verifies audience and issuer
   - Extracts user claims
   
3. API Gateway → Payment Service
   - Forwards request with user context
   - Uses mTLS + Spiffe ID for authentication
   
4. Payment Service
   - Validates Spiffe ID from API Gateway
   - Processes request with user context
   
5. Payment Service → Other Services (if needed)
   - Uses mTLS + Spiffe ID
   - Forwards user context
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
    spec:
      serviceAccountName: payment-service
      containers:
      - name: payment-service
        image: jia/payment-service:latest
        ports:
        - containerPort: 8081
          name: grpc
        env:
        - name: GRPC_ADDRESS
          value: ":8081"
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
  type: ClusterIP  # NOT LoadBalancer or NodePort
  selector:
    app: payment-service
  ports:
  - port: 8081
    targetPort: 8081
    protocol: TCP
    name: grpc
```

### API Gateway Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-gateway
  namespace: default
spec:
  replicas: 2
  selector:
    matchLabels:
      app: api-gateway
  template:
    metadata:
      labels:
        app: api-gateway
    spec:
      containers:
      - name: envoy
        image: envoyproxy/envoy:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9901
          name: admin
        volumeMounts:
        - name: envoy-config
          mountPath: /etc/envoy
      volumes:
      - name: envoy-config
        configMap:
          name: envoy-gateway-config
---
apiVersion: v1
kind: Service
metadata:
  name: api-gateway
  namespace: default
spec:
  type: LoadBalancer  # Exposed externally
  selector:
    app: api-gateway
  ports:
  - port: 8080
    targetPort: 8080
    protocol: TCP
    name: http
  - port: 9901
    targetPort: 9901
    protocol: TCP
    name: admin
```

## Monitoring

### Metrics

```go
// API Gateway metrics
gateway_requests_total{service="payment-service", status="200"}
gateway_request_duration_seconds{service="payment-service", quantile="0.95"}
gateway_requests_failed_total{service="payment-service", reason="auth_failed"}
gateway_rate_limit_hits_total{service="payment-service"}

// Payment Service metrics
grpc_server_requests_total{method="CreatePayment"}
grpc_server_request_duration_seconds{method="CreatePayment"}
grpc_server_requests_failed_total{method="CreatePayment", reason="unauthorized"}
```

### Logging

```go
// API Gateway logs
{
  "timestamp": "2024-01-15T10:30:00Z",
  "method": "POST",
  "path": "/payment.v1.PaymentService/CreatePayment",
  "status": 200,
  "duration_ms": 45,
  "user_id": "user_123",
  "ip": "192.168.1.100",
  "user_agent": "Mozilla/5.0..."
}

// Payment Service logs
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "message": "Payment created",
  "payment_id": "pay_abc123",
  "user_id": "user_123",
  "amount": 2999,
  "currency": "usd",
  "spiffe_id": "spiffe://jia.app/api-gateway"
}
```

## Testing

### Test API Gateway Integration

```bash
# Test through API Gateway
curl -X POST https://api-gateway:8080/payment.v1.PaymentService/CreatePayment \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 2999,
    "currency": "usd",
    "payment_method": "credit_card",
    "customer_id": "user_123"
  }'

# Test direct connection (should fail)
grpcurl -plaintext payment-service:8081 list
# Error: connection refused (expected)
```

## Best Practices

1. **Always use API Gateway** for external traffic
2. **Never expose gRPC endpoints** directly
3. **Use mTLS** for internal communication
4. **Implement rate limiting** at the gateway
5. **Log all requests** for auditing
6. **Use network policies** to restrict access
7. **Monitor gateway metrics** for anomalies
8. **Rotate API keys** regularly
9. **Use circuit breakers** to prevent cascading failures
10. **Implement health checks** for all services

## Troubleshooting

### Gateway Connection Issues

```bash
# Check gateway health
curl http://api-gateway:9901/stats

# Check gateway logs
kubectl logs -f deployment/api-gateway

# Test gateway connectivity
grpcurl -plaintext api-gateway:8080 list
```

### Authentication Issues

```bash
# Verify JWT token
jwt decode YOUR_JWT_TOKEN

# Check gateway JWT validation
kubectl logs -f deployment/api-gateway | grep jwt

# Test with valid token
curl -H "Authorization: Bearer VALID_TOKEN" \
  https://api-gateway:8080/payment.v1.PaymentService/ListPricingZones
```

## Summary

✅ **API Gateway Pattern**: All external traffic routes through gateway  
✅ **No Direct Exposure**: gRPC endpoint not exposed externally  
✅ **Security**: JWT + mTLS + Spiffe authentication  
✅ **Monitoring**: Comprehensive logging and metrics  
✅ **Resilience**: Circuit breakers and health checks  
✅ **Scalability**: Load balancing and service discovery  

The Payment Service is now **production-ready** with proper API Gateway integration! 🎉

