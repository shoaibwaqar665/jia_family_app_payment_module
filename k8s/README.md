# Payment Service Kubernetes Manifests

This directory contains Kubernetes manifests for deploying the payment service.

## Prerequisites

- Kubernetes cluster (1.24+)
- kubectl configured to access the cluster
- kustomize (optional, for deployment)

## Deployment

### Using kubectl

Deploy all resources:

```bash
kubectl apply -f k8s/
```

### Using kustomize

```bash
kubectl apply -k k8s/
```

## Components

- **namespace.yaml**: Creates the `payment-service` namespace
- **service-account.yaml**: Creates service account and RBAC permissions
- **configmap.yaml**: Application configuration
- **secrets.yaml**: Sensitive data (passwords, API keys, etc.)
- **deployment.yaml**: Main application deployment with 3 replicas
- **service.yaml**: ClusterIP service for internal access
- **hpa.yaml**: Horizontal Pod Autoscaler for automatic scaling
- **network-policy.yaml**: Network policies for security
- **kustomization.yaml**: Kustomize configuration

## Configuration

### Environment Variables

The service expects the following environment variables (set in secrets.yaml):

- `DB_DSN`: PostgreSQL connection string
- `REDIS_HOST`: Redis hostname
- `REDIS_PORT`: Redis port
- `REDIS_DB`: Redis database number
- `REDIS_PASSWORD`: Redis password
- `JWT_PUBLIC_KEY`: JWT public key for authentication
- `STRIPE_SECRET_KEY`: Stripe secret key
- `STRIPE_PUBLISHABLE_KEY`: Stripe publishable key
- `STRIPE_WEBHOOK_SECRET`: Stripe webhook secret

### TLS Configuration

The service expects TLS certificates mounted at `/etc/tls/`:
- `tls.crt`: Server certificate
- `tls.key`: Server private key

Create a Kubernetes secret with your TLS certificates:

```bash
kubectl create secret tls payment-service-tls \
  --cert=/path/to/tls.crt \
  --key=/path/to/tls.key \
  -n payment-service
```

## Scaling

The HPA automatically scales the deployment based on CPU and memory usage:
- Minimum replicas: 3
- Maximum replicas: 10
- Target CPU utilization: 70%
- Target memory utilization: 80%

## Monitoring

The service exposes Prometheus metrics on port 9090.

## Security

- Network policies restrict ingress/egress traffic
- Service runs as non-root user (UID 65534)
- TLS enabled for gRPC communication
- Secrets stored in Kubernetes secrets (not in config files)

## Troubleshooting

### Check pod status

```bash
kubectl get pods -n payment-service
```

### View logs

```bash
kubectl logs -f deployment/payment-service -n payment-service
```

### Check service endpoints

```bash
kubectl get endpoints -n payment-service
```

### Test gRPC health check

```bash
kubectl exec -it deployment/payment-service -n payment-service -- \
  grpc_health_probe -addr=:50051
```

## Production Considerations

1. **Secrets Management**: Use a proper secrets management solution (e.g., HashiCorp Vault, AWS Secrets Manager)
2. **TLS Certificates**: Use cert-manager for automatic certificate management
3. **Monitoring**: Integrate with Prometheus and Grafana
4. **Logging**: Use centralized logging (e.g., ELK stack, Loki)
5. **Service Mesh**: Consider using Istio or Linkerd for advanced traffic management
6. **Backup**: Ensure database backups are configured
7. **Disaster Recovery**: Have a disaster recovery plan in place

