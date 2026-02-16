```markdown
# 🛡️ Fraud Engine: Full Kubernetes Deployment & GitOps Guide

This repository contains a Go-based fraud detection microservice. This guide serves as a complete reference for deploying the stack (Go, Postgres, Redis, Kafka) to **Azure Kubernetes Service (AKS)** using **Argo CD** and **Docker Hub**.

---

## 🏗️ 1. Infrastructure Setup (One-Time)

### **A. Container Registry (Docker Hub)**
1. Create a repository on [Docker Hub](https://hub.docker.com/) (e.g., `yourdocker/fraud-engine`).
2. Login locally:
   ```bash
   docker login

```

### **B. Create the Azure AKS Cluster**

```bash
# 1. Create Resource Group
az group create --name fraud-engine-rg --location eastus

# 2. Create the Cluster (2 nodes for balance of cost/performance)
az aks create --resource-group fraud-engine-rg --name fraud-cluster --node-count 2 --generate-ssh-keys

# 3. Connect kubectl to the cluster
az aks get-credentials --resource-group fraud-engine-rg --name fraud-cluster

```

### **C. Install Argo CD (The GitOps Brain)**

```bash
# 1. Install Argo CD in its own namespace
kubectl create namespace argocd
kubectl apply -n argocd -f [https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml](https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml)

# 2. Expose Argo UI via Public IP
kubectl patch svc argocd-server -n argocd -p '{"spec": {"type": "LoadBalancer"}}'

# 3. Get the Public IP for Argo UI
kubectl get svc -n argocd argocd-server

# 4. Get Initial Admin Password
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d

```

---

## 🛠️ 2. Development & Deployment Workflow

### **Step 1: Generate Swagger Documentation**

Every time you update your API comments in Go, regenerate the docs:

```bash
swag init -g cmd/api/main.go --parseDependency --parseInternal

```

### **Step 2: Build & Push to Docker Hub**

```bash
# Build for Linux
docker build -t yourdocker/fraud-engine:latest .

# Push to your registry
docker push yourdocker/fraud-engine:latest

```

### **Step 3: GitOps Sync**

Update your `k8s/app.yaml` with the new image tag, then push to GitHub. Argo CD will automatically sync the changes to Azure.

```bash
git add .
git commit -m "feat: deploy latest fraud logic"
git push origin main

```

---

## 🔍 3. Troubleshooting & Access

### **Accessing the API / Swagger**

1. **Find Public IP:** `kubectl get svc -n fraud-engine fraud-engine-service`
2. **Swagger URL:** `http://<EXTERNAL-IP>/swagger/index.html`

### **Common Debugging Commands**

* **Check Logs:** `kubectl logs -f deployment/fraud-engine-deployment -n fraud-engine`
* **Check Database:** `kubectl exec -it postgres-0 -n fraud-engine -- psql -U admin -d fraud_db`
* **Restart Deployment:** `kubectl rollout restart deployment/fraud-engine-deployment -n fraud-engine`

---

## ⚠️ 4. Key Deployment Fixes (Lessons Learned)

| Issue | Resolution |
| --- | --- |
| **Postgres Start Failure** | Use `PGDATA` env var to point to a subfolder (e.g., `/var/lib/postgresql/data/pgdata`) to avoid `lost+found` errors. |
| **Kafka Image Failure** | Avoid `:latest` tags for Bitnami images; use specific versions like `bitnami/kafka:3.7`. |
| **DB Connection Error** | Ensure `main.go` reads `DB_DSN` environment variable to match Kubernetes manifest keys. |
| **Argo CD Sync Error** | Enable **Server-Side Apply** in Argo CD sync options if the manifest is too large. |

---

## 💰 5. Cost Management (Daily Cleanup)

To stop being charged by Azure while not working, follow these steps in order:

### **1. Detach Public IPs (Saves Networking Costs)**

Change services back to `ClusterIP` to release the paid Azure Load Balancer IPs.

```bash
kubectl patch svc fraud-engine-service -n fraud-engine -p '{"spec": {"type": "ClusterIP"}}'
kubectl patch svc argocd-server -n argocd -p '{"spec": {"type": "ClusterIP"}}'

```

### **2. Stop the Cluster (Saves Compute Costs)**

This shuts down the VMs but keeps your cluster configuration and disks intact.

```bash
az aks stop --name fraud-cluster --resource-group fraud-engine-rg

```

### **3. Resume Work**

```bash
az aks start --name fraud-cluster --resource-group fraud-engine-rg

```

### **4. Final Deletion (Zero Future Costs)**

Only run this if you want to delete everything forever.

```bash
az group delete --name fraud-engine-rg --yes --no-wait

```

```

Would you like me to help you create a **Makefile** next to make these commands even shorter (e.g., just typing `make stop` instead of the full Azure CLI command)?

```