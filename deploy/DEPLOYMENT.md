# Deployment Notes

## Prerequisites

All via brew:

- `docker`
- `kubectl` -- [Install](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/)
- `minikube` (local K8S cluster, start with `minikube start`)
- `k9s` (CLI kubernetes manager) -- [Install](https://k9scli.io/topics/install/)
- `kompose` (Compose to K8S conversion)
- `awscli` (configure with `aws configure`)
- `yq` (read from yaml files from CLI) -- [Github](https://github.com/mikefarah/yq)

## Cluster setup

Start minikube and create the `bidder` namespace:

```bash
minikube start
kubectl apply -f k8s/namespace.yaml
```

Verify the cluster is running and the namespace exists:

```bash
kubectl cluster-info
kubectl get namespace bidder
```

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
source ./load-values.sh

envsubst < k8s/services.yaml | kubectl apply -f -
```

Vendor images (Postgres, Mongo, Redis, Traefik) are pinned directly in manifests.

## Secrets and config

Each `.env` file is split into a `.secret` (sensitive) and `.config` (non-sensitive) variant. Database env files (`.env.auth-db`, `.env.users-db`, `.env.policy-db`) are kept as pure secrets. `.env.bidder-frontend` is kept as a pure configmap.

The script also creates file-based ConfigMaps that are volume-mounted into pods:

- `policy-cache-config` — Redis config from `redis/policy_cache/redis.conf`
- `users-cache-config` — Redis config from `redis/users_cache/redis.conf`
- `policy-db-init-scripts` — MongoDB init script from `policy_db/init-scripts/`
- `traefik-authheader-plugin` — Traefik plugin source from `traefik-plugin-authheader/`

The [`create-k8s-secrets.sh`](create-k8s-secrets.sh) script creates all Secrets and ConfigMaps. It is safe to re-run:

```bash
./create-k8s-secrets.sh ..
```

Manifests reference them with:

```yaml
envFrom:
  - secretRef:
      name: auth-service-secret
  - configMapRef:
      name: auth-service-config
  - secretRef:
      name: shared-secret
  - configMapRef:
      name: shared-config
```

To update a secret, edit the `.secret` file and re-run the script:

```bash
./create-k8s-secrets.sh ..
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
source ./load-values.sh k8s/values.yaml
envsubst < k8s/volumes.yaml | kubectl apply -f -
```

- `minikube`: paths resolve inside the VM/container, not on host. With the Docker driver, `minikube mount` does not work (9p unsupported). Instead, pass the mount at startup

```bash
minikube start --driver=docker --mount --mount-string="/host/path:/data/bidder"
```

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
| Host rules   | `Host(\`localhost\`)`      | `Host(\`${BACKEND_DOMAIN}\`)`                       |
| Service URLs | `http://auth_service:8080` | `http://auth-service.bidder.svc.cluster.local:8080` |

Set production domains in [`k8s/values.yaml`](k8s/values.yaml), then:

```bash
source ./load-values.sh k8s/values.yaml
envsubst < k8s/traefik-routes.yaml | kubectl apply -f -
```

## Nginx and minikube networking

In production, TLS is handled in front of the cluster (server Nginx or cloud LB). For local dev with minikube, an nginx container provides TLS termination and proxies to the K8S NodePorts.

### How it works

Minikube runs as a Docker container with its own IP. K8S Services of type `NodePort` are reachable at `<minikube-ip>:<nodePort>`. The nginx container joins minikube's Docker network so it can reach those ports, then exposes HTTPS on `localhost`.

```
Browser --HTTPS--> nginx (localhost:443/444) --HTTP--> minikube:NodePort --> K8S Service --> Pod
```

### Getting the minikube IP

```bash
minikube ip
```

This is typically `192.168.49.2` but can change. If it changes, update the `upstream` blocks in [`nginx/k8s.conf`](../nginx/k8s.conf).

### Getting the NodePorts

After deploying, check which ports K8S assigned:

```bash
kubectl get services -n bidder
```

Look for the `NodePort` column on `traefik` and `bidder-frontend`:

| Service           | Port mapping   | Meaning                       |
| ----------------- | -------------- | ----------------------------- |
| `traefik`         | `81:30080/TCP` | API gateway on NodePort 30080 |
| `bidder-frontend` | `81:30081/TCP` | Frontend on NodePort 30081    |

These NodePort values are set in [`k8s/traefik.yaml`](k8s/traefik.yaml) and [`k8s/services.yaml`](k8s/services.yaml). If K8S assigns different ports, update the `upstream` blocks in `nginx/k8s.conf` to match.

### nginx/k8s.conf

The config defines two server pairs (HTTP redirect + HTTPS):

- **Ports 80/443** → traefik NodePort (API gateway)
- **Ports 81/444** → bidder-frontend NodePort

Each uses mkcert TLS certificates from `certs/`. The backend uses `localhost+2.pem`, the frontend uses `frontend-localhost+2.pem`.

### Running nginx alongside minikube

The nginx container must be on minikube's Docker network to reach `192.168.49.2`. This is configured in [`compose.prod.yml`](compose.prod.yml):

```yaml
services:
  nginx:
    networks:
      - minikube
networks:
  minikube:
    external: true # created by minikube start
```

Start it:

```bash
cd deploy
docker compose -f compose.prod.yml up -d
```

Then access:

- **API gateway**: `https://localhost` (port 443)
- **Frontend**: `https://localhost:444`

### Gotchas

1. **minikube must be running first.** The `minikube` Docker network is created by `minikube start`. If nginx starts before minikube, Compose will fail because the external network doesn't exist yet.

2. **IP can change.** If you delete and recreate the minikube cluster, the IP may change. Always verify with `minikube ip` and update `nginx/k8s.conf` if needed.

3. **NodePorts are fixed in manifests, not dynamic.** The manifests pin `nodePort: 30080` and `nodePort: 30081`. If you change them in the manifests, update `nginx/k8s.conf` to match.

4. **nginx container can't use `localhost`.** Inside the nginx container, `localhost` is itself. You must use minikube's IP (`192.168.49.2`) in the upstream blocks.

5. **ECR tokens expire after 12 hours.** If pods enter `ImagePullBackOff` after a long uptime, refresh the ECR pull secret (step 5 in [Full deployment](#full-deployment)).

6. **Restart nginx after config changes.** nginx reads `k8s.conf` at startup. After editing it:
   ```bash
   docker compose -f compose.prod.yml restart
   ```

## Full deployment

```bash
cd deploy

# 1. start cluster (add --mount flags if using Docker driver)
minikube start

# 2. namespace
kubectl apply -f k8s/namespace.yaml

# 3. load values from values.yaml
source ./load-values.sh k8s/values.yaml

# 4. secrets and config
./create-k8s-secrets.sh ..

# 5. ECR pull secret
TOKEN=$(aws ecr get-login-password --region eu-west-2)
kubectl create secret docker-registry ecr-pull-secret \
  --docker-server=${AWS_ACCOUNT_ID}.dkr.ecr.eu-west-2.amazonaws.com \
  --docker-username=AWS \
  --docker-password="$TOKEN" \
  --dry-run=client -o yaml | kubectl apply -n bidder -f -

# 6. network policies
kubectl apply -f k8s/network-policies.yaml

# 7. persistent volumes
envsubst < k8s/volumes.yaml | kubectl apply -f -

# 8. databases and caches
kubectl apply -f k8s/databases.yaml

# 9. traefik
kubectl apply -f k8s/traefik.yaml
envsubst < k8s/traefik-routes.yaml | kubectl apply -f -

# 10. application services
envsubst < k8s/services.yaml | kubectl apply -f -

# 11. verify
kubectl get pods -n bidder
kubectl get services -n bidder
```

Access on minikube (via nginx TLS proxy):

```bash
# start the nginx proxy (requires minikube to be running)
docker compose -f compose.prod.yml up -d
```

- API gateway: `https://localhost`
- Frontend: `https://localhost:444`

See [Nginx and minikube networking](#nginx-and-minikube-networking) for setup details and troubleshooting.

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
# # Docker
# sudo yum install -y docker
# sudo systemctl enable docker && sudo systemctl start docker
# sudo usermod -aG docker ec2-user
# # log out and back in for group change

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
minikube start --driver=docker --mount --mount-string="/host/path:/data/bidder"

kubectl apply -f deploy/k8s/namespace.yaml

# load values
source deploy/load-values.sh deploy/k8s/values.yaml

# secrets and config
deploy/create-k8s-secrets.sh .

# ECR pull secret
TOKEN=$(aws ecr get-login-password --region eu-west-2)
kubectl create secret docker-registry ecr-pull-secret \
  --docker-server=${AWS_ACCOUNT_ID}.dkr.ecr.eu-west-2.amazonaws.com \
  --docker-username=AWS \
  --docker-password="$TOKEN" \
  --dry-run=client -o yaml | kubectl apply -n bidder -f -

# network policies
kubectl apply -f deploy/k8s/network-policies.yaml

# volumes
envsubst < deploy/k8s/volumes.yaml | kubectl apply -f -

# databases and caches
kubectl apply -f deploy/k8s/databases.yaml

# traefik
kubectl apply -f deploy/k8s/traefik.yaml
envsubst < deploy/k8s/traefik-routes.yaml | kubectl apply -f -

# services
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

### Restarting deployments

To restart all deployments (e.g. after pushing new images with the same tag):

```bash
kubectl rollout restart deployment -n bidder
```

To restart a single service:

```bash
kubectl rollout restart deployment/auth-service -n bidder
```

### Nuking the namespace

To tear down everything and start fresh, delete the namespace:

```bash
kubectl delete namespace bidder
```

This removes all resources within it (Deployments, Services, Pods, Secrets, ConfigMaps, PVCs). PersistentVolumes are cluster-scoped and must be deleted separately:

```bash
kubectl delete pv -l app -A
```

Then re-run the full deployment steps from the [Full deployment](#full-deployment) section.

### Stopping and starting the cluster

To stop minikube (frees port bindings and CPU/memory, but preserves all state):

```bash
minikube stop
```

To start it back up:

```bash
minikube start
```

All deployments, services, secrets, and volumes are retained across stop/start. Pods will resume automatically. This is the equivalent of Docker stop and Docker start.
