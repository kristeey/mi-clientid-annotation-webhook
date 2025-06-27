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

## Deploying to kind cluster

1. Build images as specified above
2. Create kind cluster: `kind create cluster`
3. Load images onto kind nodes:
   - `kind load docker-image mi-client-mutating-webhook-server:latest`
   - `kind load docker-image webhook-init-container:latest`
4. Edit `/deploy/k8s/deployment.yaml`:
   - Set `image:` for `webhook-server` to your built image.
   - Set `image:` for `cert-init` (the init container) to your built init container image.
   - Set the following
     - `AZURE_TENANT_ID`: The Azure tenant.
     - `AZURE_SUBSCRIPTION_ID`: The Azure subscription to look for MI client ID.
     - `AZURE_CLIENT_ID`: The Azure identity client ID to authenticate as against Entra ID.
     - `AZURE_CLIENT_SECRET`: Client secret for that Azure identity. Do not use in production (either use secrets or workload identity which do not need this).
3. Apply the server and webhook manifests:
   ```bash
   kubectl apply -f deploy/k8s/deployment.yaml
   kubectl apply -f deploy/k8s/webhook.yaml
   ```
4. Edit `deploy/k8s/test-sa.yaml` to include `mi.clientid.webhook/azure-mi-client-name` annotation.
5. Apply the service account:
   ```bash
   kubectl apply -f deploy/k8s/test-sa.yaml
   ```
6. Verify that the service account has been mutated to include the `azure.workload.identity/client-id` annotation: `kubectl -n webhook-demo get sa test-sa -o yaml`

---

