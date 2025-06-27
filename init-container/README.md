# Webhook Init Container: Self-Signed Certificate Generator & Webhook Configurator

This Go-based init container generates a self-signed TLS certificate and key for use by a webhook server, and automatically creates or updates a Kubernetes MutatingWebhookConfiguration with the correct CA bundle.

## What the Init Container Does
- Ensures the directory `/etc/webhook/certs/` exists.
- Generates a self-signed CA and a server certificate (with SANs for webhook-service DNS names).
- Writes the certificate and key to `/etc/webhook/certs/`.
- Uses the Kubernetes Go client to create or update a `MutatingWebhookConfiguration` in the cluster, with the CA bundle set to the generated CA cert.

## How to Build the Container

1. Make sure you are in the `init-container` directory with the `Dockerfile` and Go source files present.
2. Build the Docker image:

   ```bash
   docker build -t webhook-init-container .
   ```

## How to Use
- Add this image as an init container in your deployment.
- Mount `/etc/webhook/certs/` as a shared volume between the init container and your webhook server container.
- The webhook server should use `/etc/webhook/certs/tls.crt` and `/etc/webhook/certs/tls.key` for TLS.
- The init container will automatically create or update the `MutatingWebhookConfiguration` resource in the cluster with the correct CA bundle.

---
**Note:** The certificate is self-signed and intended for development or internal use. For production, use a trusted CA and manage webhook configuration securely.
