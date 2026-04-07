# Deployment Notes

## Prerequisites

All via brew:

- `docker`
- `kubectl`
- `minikube` (local K8S cluster, start with `minikube start`)
- `kompose` (Compose to K8S conversion)
- `awscli` (configure with `aws configure`)


## Pushing images to ECR

- Ensure ECR repos exist (e.g. `bidder/auth-service`)
- Run from each service's directory:

```bash
./push-to-ecr.sh <AWS_ACCOUNT_ID> <IMAGE_NAME> <VERSION>
```

- Args: AWS account ID, image name, semver tag
- Produces: `<ACCOUNT_ID>.dkr.ecr.eu-west-2.amazonaws.com/bidder/<IMAGE_NAME>:<VERSION>`

Example:

```bash
cd auth-service && ../push-to-ecr.sh 123456789012 auth-service 1.0.0
```

Repeat for: `auth-service`, `user-service`, `policy-service`, `bids-service`, `bidder-frontend` (which is just called `frontend` since it is `bidder/frontend`).


## Image Tags Problems (`latest`)

- K8S with a non-`latest` tag defaults to `imagePullPolicy: IfNotPresent` (won't re-pull on change)
- Different nodes can cache different versions of `latest`
- Always use explicit version tags
- Manifests set `imagePullPolicy: Always` as a safety net


## Parameterising images

Versions and account ID live in [`k8s/values.yaml`](k8s/values.yaml). Manifests use placeholders:

```yaml
image: ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/bidder/auth-service:${AUTH_SERVICE_VERSION}
```

Apply with `envsubst`:

```bash
export AWS_ACCOUNT_ID=123456789012
export AWS_REGION=eu-west-2
export ECR_PULL_SECRET=ecr-pull-secret
export AUTH_SERVICE_VERSION=1.0.0
export USER_SERVICE_VERSION=1.0.0
export POLICY_SERVICE_VERSION=1.0.0
export BIDS_SERVICE_VERSION=1.0.0
export FRONTEND_VERSION=1.0.0

envsubst < k8s/services.yaml | kubectl apply -f -
```

Vendor images (Postgres, Mongo, Redis, Traefik) are pinned directly in manifests.


## Secrets and config

`.env` files map to K8S **Secrets** (sensitive) and **ConfigMaps** (non-sensitive) via `--from-env-file`:

```bash
# secrets
kubectl create secret generic auth-db-secret       --from-env-file=../.env.auth_db       -n bidder
kubectl create secret generic auth-service-secret   --from-env-file=../.env.auth_service   -n bidder
kubectl create secret generic users-db-secret       --from-env-file=../.env.users_db       -n bidder
kubectl create secret generic user-service-secret   --from-env-file=../.env.user_service   -n bidder
kubectl create secret generic policy-db-secret      --from-env-file=../.env.policy_db      -n bidder
kubectl create secret generic policy-service-secret --from-env-file=../.env.policy_service  -n bidder
kubectl create secret generic bids-service-secret   --from-env-file=../.env.bids_service   -n bidder
kubectl create secret generic traefik-secret        --from-env-file=../.env.traefik        -n bidder

# config
kubectl create configmap shared-config          --from-env-file=../.env.shared          -n bidder
kubectl create configmap bidder-frontend-config --from-env-file=../.env.bidder_frontend  -n bidder
```

Manifests reference them with:

```yaml
envFrom:
  - secretRef:
      name: auth-service-secret
  - configMapRef:
      name: shared-config
```

To update a secret (no in-place update in K8S):

```bash
kubectl delete secret auth-service-secret -n bidder
kubectl create secret generic auth-service-secret --from-env-file=../.env.auth_service -n bidder
kubectl rollout restart deployment/auth-service -n bidder
```


## Ports

Reference: [`k8s/ports.yaml`](k8s/ports.yaml)

| Compose         | K8S          | Scope                    |
| --------------- | ------------ | ------------------------ |
| `ports: "X:Y"`  | ClusterIP    | Internal only            |
| Exposed to host | NodePort     | Port 30000-32767 on node |
| Public          | LoadBalancer | External IP via cloud    |

- Databases, caches, bids-service: ClusterIP
- Traefik, frontend: NodePort (swap to LoadBalancer for cloud)
- Edit `nodePort` values in `k8s/traefik.yaml` and `k8s/services.yaml`


## Storage paths

Compose volumes (`./postgres/auth_db/data:/var/lib/...`) become PV + PVC pairs in K8S.

- **PersistentVolume** = host-side path
- **PersistentVolumeClaim** = pod's storage request
- Host paths parameterised in [`k8s/volumes.yaml`](k8s/volumes.yaml), defaults in [`k8s/values.yaml`](k8s/values.yaml)

```bash
export AUTH_DB_HOST_PATH=/data/bidder/postgres/auth_db
export USERS_DB_HOST_PATH=/data/bidder/postgres/users_db
export POLICY_DB_HOST_PATH=/data/bidder/policy_db
export POLICY_CACHE_HOST_PATH=/data/bidder/redis/policy_cache
export USERS_CACHE_HOST_PATH=/data/bidder/redis/users_cache

envsubst < k8s/volumes.yaml | kubectl apply -f -
```

- Minikube: paths resolve inside the VM, not on host. Mount with `minikube mount /host/path:/data/bidder`
- Container-side paths (`/var/lib/postgresql/data`, `/data/db`, etc.) are fixed by the applications


## Networks

Compose networks map to K8S **NetworkPolicies** ([`k8s/network-policies.yaml`](k8s/network-policies.yaml)):

- `bids-backend` label: all backend services, databases, caches, traefik
- `bids-frontend` label: frontend only
- Pods can only talk to others with the same network label
- Requires a CNI that enforces policies (Calico, Cilium). Minikube's default may not.


## Traefik in production

No Docker socket in K8S, so Traefik switches to file-based routing via ConfigMap ([`k8s/traefik-routes.yaml`](k8s/traefik-routes.yaml)).

| What         | Dev                        | Prod                                                |
| ------------ | -------------------------- | --------------------------------------------------- |
| Provider     | Docker socket              | File (ConfigMap)                                    |
| Dashboard    | On                         | Off                                                 |
| Log level    | `DEBUG`                    | `WARN`                                              |
| Host rules   | `Host(\`localhost\`)`      | `Host(\`${DOMAIN}\`)`                               |
| Service URLs | `http://auth_service:8080` | `http://auth-service.bidder.svc.cluster.local:8080` |

Set production domain:

```bash
export DOMAIN=bidder.yourdomain.com
envsubst < k8s/traefik-routes.yaml | kubectl apply -f -
```


## Nginx

Not included in production. The dev `nginx/` layer just simulates TLS termination with mkcert. In production, TLS is handled in front of the cluster (server Nginx or cloud LB).


## Full deployment

```bash
cd deploy

# 1. start cluster
minikube start

# 2. namespace
kubectl apply -f k8s/namespace.yaml

# 3. secrets and config
kubectl create secret generic auth-db-secret       --from-env-file=../.env.auth_db       -n bidder
kubectl create secret generic auth-service-secret   --from-env-file=../.env.auth_service   -n bidder
kubectl create secret generic users-db-secret       --from-env-file=../.env.users_db       -n bidder
kubectl create secret generic user-service-secret   --from-env-file=../.env.user_service   -n bidder
kubectl create secret generic policy-db-secret      --from-env-file=../.env.policy_db      -n bidder
kubectl create secret generic policy-service-secret --from-env-file=../.env.policy_service  -n bidder
kubectl create secret generic bids-service-secret   --from-env-file=../.env.bids_service   -n bidder
kubectl create secret generic traefik-secret        --from-env-file=../.env.traefik        -n bidder
kubectl create configmap shared-config              --from-env-file=../.env.shared          -n bidder
kubectl create configmap bidder-frontend-config     --from-env-file=../.env.bidder_frontend -n bidder

# 4. network policies
kubectl apply -f k8s/network-policies.yaml

# 5. persistent volumes
export AUTH_DB_HOST_PATH=/data/bidder/postgres/auth_db
export USERS_DB_HOST_PATH=/data/bidder/postgres/users_db
export POLICY_DB_HOST_PATH=/data/bidder/policy_db
export POLICY_CACHE_HOST_PATH=/data/bidder/redis/policy_cache
export USERS_CACHE_HOST_PATH=/data/bidder/redis/users_cache
envsubst < k8s/volumes.yaml | kubectl apply -f -

# 6. databases and caches
kubectl apply -f k8s/databases.yaml

# 7. traefik
kubectl apply -f k8s/traefik.yaml
kubectl apply -f k8s/traefik-routes.yaml

# 8. application services
export AWS_ACCOUNT_ID=123456789012
export AWS_REGION=eu-west-2
export ECR_PULL_SECRET=ecr-pull-secret
export AUTH_SERVICE_VERSION=1.0.0
export USER_SERVICE_VERSION=1.0.0
export POLICY_SERVICE_VERSION=1.0.0
export BIDS_SERVICE_VERSION=1.0.0
export FRONTEND_VERSION=1.0.0
export DOMAIN=localhost
envsubst < k8s/services.yaml | kubectl apply -f -

# 9. verify
kubectl get pods -n bidder
kubectl get services -n bidder
```

Access on minikube:

```bash
minikube service traefik -n bidder
minikube service bidder-frontend -n bidder
```


## Kompose

Initial manifests were scaffolded with:

```bash
kompose convert -f compose.prod.yml -o k8s/
```

Output needs manual fixes:

- Images: swap to ECR URIs
- `env_file`: convert to Secrets/ConfigMaps
- Networks: add NetworkPolicies (not generated)
- Traefik: switch to file provider
- `imagePullPolicy`: set correctly

Manifests in `k8s/` are already adjusted.


## Deploying to EC2 (AL2)

### Install dependencies

```bash
# Docker
sudo yum install -y docker
sudo systemctl enable docker && sudo systemctl start docker
sudo usermod -aG docker ec2-user
# log out and back in for group change

# kubectl
curl -LO "https://dl.k8s.io/release/$(curl -Ls https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl && sudo mv kubectl /usr/local/bin/

# minikube
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
chmod +x minikube-linux-amd64 && sudo mv minikube-linux-amd64 /usr/local/bin/minikube

# envsubst
sudo yum install -y gettext
```

### ECR access for minikube

Minikube runs its own Docker daemon. Log into ECR from within it:

```bash
minikube start --driver=docker
eval $(minikube docker-env)
aws ecr get-login-password --region eu-west-2 | docker login --username AWS --password-stdin <ACCOUNT_ID>.dkr.ecr.eu-west-2.amazonaws.com
```

- ECR tokens expire after 12 hours
- For long-running clusters, refresh via cron or use an image pull secret:

```bash
TOKEN=$(aws ecr get-login-password --region eu-west-2)
kubectl create secret docker-registry ecr-pull-secret \
  --docker-server=<ACCOUNT_ID>.dkr.ecr.eu-west-2.amazonaws.com \
  --docker-username=AWS \
  --docker-password="$TOKEN" \
  -n bidder
```

Then add `imagePullSecrets: [{ name: ecr-pull-secret }]` to each Deployment's pod spec.

### Copy files

```bash
scp -r deploy/ ec2-user@<EC2_IP>:~/deploy/
scp .env.* ec2-user@<EC2_IP>:~/
```

### Deploy

```bash
minikube start --driver=docker

kubectl apply -f deploy/k8s/namespace.yaml

# secrets
kubectl create secret generic auth-db-secret       --from-env-file=.env.auth_db       -n bidder
kubectl create secret generic auth-service-secret   --from-env-file=.env.auth_service   -n bidder
kubectl create secret generic users-db-secret       --from-env-file=.env.users_db       -n bidder
kubectl create secret generic user-service-secret   --from-env-file=.env.user_service   -n bidder
kubectl create secret generic policy-db-secret      --from-env-file=.env.policy_db      -n bidder
kubectl create secret generic policy-service-secret --from-env-file=.env.policy_service  -n bidder
kubectl create secret generic bids-service-secret   --from-env-file=.env.bids_service   -n bidder
kubectl create secret generic traefik-secret        --from-env-file=.env.traefik        -n bidder
kubectl create configmap shared-config              --from-env-file=.env.shared          -n bidder
kubectl create configmap bidder-frontend-config     --from-env-file=.env.bidder_frontend -n bidder

# network policies
kubectl apply -f deploy/k8s/network-policies.yaml

# volumes
export AUTH_DB_HOST_PATH=/data/bidder/postgres/auth_db
export USERS_DB_HOST_PATH=/data/bidder/postgres/users_db
export POLICY_DB_HOST_PATH=/data/bidder/policy_db
export POLICY_CACHE_HOST_PATH=/data/bidder/redis/policy_cache
export USERS_CACHE_HOST_PATH=/data/bidder/redis/users_cache
envsubst < deploy/k8s/volumes.yaml | kubectl apply -f -

# databases and caches
kubectl apply -f deploy/k8s/databases.yaml

# traefik
export DOMAIN=bidder.yourdomain.com
envsubst < deploy/k8s/traefik-routes.yaml | kubectl apply -f -
kubectl apply -f deploy/k8s/traefik.yaml

# services
export AWS_ACCOUNT_ID=123456789012
export AWS_REGION=eu-west-2
export ECR_PULL_SECRET=ecr-pull-secret
export AUTH_SERVICE_VERSION=1.0.0
export USER_SERVICE_VERSION=1.0.0
export POLICY_SERVICE_VERSION=1.0.0
export BIDS_SERVICE_VERSION=1.0.0
export FRONTEND_VERSION=1.0.0
envsubst < deploy/k8s/services.yaml | kubectl apply -f -

# verify
kubectl get pods -n bidder
kubectl get services -n bidder
```

### Exposing externally

Minikube runs inside Docker's network namespace. Forward ports to make services reachable:

```bash
kubectl port-forward --address 0.0.0.0 svc/traefik 80:81 -n bidder &
```

- Ensure the EC2 security group allows inbound on exposed ports (80, 443)
- Alternative: `minikube tunnel` (then switch Service types to `LoadBalancer`)

### TLS

Handled by the server's Nginx in front of the cluster:

```nginx
server {
    listen 443 ssl;
    server_name bidder.yourdomain.com;
    ssl_certificate     /etc/ssl/certs/bidder.crt;
    ssl_certificate_key /etc/ssl/private/bidder.key;
    location / {
        proxy_pass http://127.0.0.1:80;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Updating a service

```bash
# locally
../push-to-ecr.sh 123456789012 auth-service 1.1.0

# on EC2 (re-export all vars, envsubst replaces them all)
export AUTH_SERVICE_VERSION=1.1.0
envsubst < deploy/k8s/services.yaml | kubectl apply -f -
```

If tag unchanged but image changed: `kubectl rollout restart deployment/auth-service -n bidder`

### Useful commands

```bash
kubectl logs deployment/auth-service -n bidder            # logs
kubectl describe pod <pod-name> -n bidder                 # debug
kubectl get events -n bidder --sort-by=.lastTimestamp      # events
kubectl top pods -n bidder                                # resource usage
kubectl rollout status deployment/auth-service -n bidder  # rollout progress
```
