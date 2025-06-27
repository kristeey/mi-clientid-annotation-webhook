# Mutating Admission Webhook: ServiceAccount Label Mutator

This repository contains:

- **Webhook Server**: Go code for a Kubernetes mutating admission webhook that adds a label to ServiceAccounts with the label `hei: hallo`.
- **Init Container**: Bash project (in `/init-container`) that generates a self-signed certificate for the webhook server at startup.
- **Kubernetes Deployment Files**: Manifests in `/deploy/k8s` to deploy the webhook server and init container, including all necessary resources.

---

## Building the Webhook Server

1. Build the Docker image:
   ```bash
   docker build -t mi-client-mutating-webhook-server .
   ```

## Building the Init Container

1. Change to the init container directory and build the image:
   ```bash
   cd init-container
   docker build -t webhook-init-container .
   ```

---

## Deploying to Kubernetes

1. Push both images to your container registry.
2. Edit `/deploy/k8s/deployment.yaml`:
   - Set `image:` for `webhook-server` to your built image.
   - Set `image:` for `cert-init` (the init container) to your built init container image.
3. Apply the manifests:
   ```bash
   kubectl apply -f deploy/k8s/deployment.yaml
   kubectl apply -f deploy/k8s/webhook.yaml
   ```

The init container will generate a self-signed certificate and place it in `/etc/webhook/certs`, which is then used by the webhook server.

## Deploying to kind cluster

1. Build images as specified above
2. Create kind cluster: `kind create cluster`
3. Load images onto kind nodes:
   - `kind load docker-image mi-client-mutating-webhook-server:latest`
   - `kind load docker-image webhook-init-container:latest`
4. Edit `/deploy/k8s/deployment.yaml`:
   - Set `image:` for `webhook-server` to your built image.
   - Set `image:` for `cert-init` (the init container) to your built init container image.
3. Apply the manifests:
   ```bash
   kubectl apply -f deploy/k8s/deployment.yaml
   kubectl apply -f deploy/k8s/webhook.yaml
   ```

---

**Note:** For production, use a proper CA and certificate management process. The provided setup is for development and demonstration purposes.
