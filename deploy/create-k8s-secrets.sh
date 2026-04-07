NAMESPACE="bidder"

if [ $# -ne 1 ]; then
  echo "Usage: $0 <env-dir>" >&2
  exit 1
fi

ENV_DIR="$1"

if [ ! -d "$ENV_DIR" ]; then
  echo "Error: directory '$ENV_DIR' not found" >&2
  exit 1
fi

apply_secret() {
  local name="$1" file="$2"
  kubectl create secret generic "$name" \
    --from-env-file="$ENV_DIR/$file" \
    --dry-run=client -o yaml | kubectl apply -n "$NAMESPACE" -f -
}

apply_configmap() {
  local name="$1" file="$2"
  kubectl create configmap "$name" \
    --from-env-file="$ENV_DIR/$file" \
    --dry-run=client -o yaml | kubectl apply -n "$NAMESPACE" -f -
}

echo "Creating secrets"

# Shared
apply_secret shared-secret           .env.shared.secret

# Databases
apply_secret auth-db-secret          .env.auth-db
apply_secret users-db-secret         .env.users-db
apply_secret policy-db-secret        .env.policy-db

# Services
apply_secret auth-service-secret     .env.auth-service.secret
apply_secret user-service-secret     .env.user-service.secret
apply_secret policy-service-secret   .env.policy-service.secret
apply_secret bids-service-secret     .env.bids-service.secret
apply_secret traefik-secret          .env.traefik

echo "Creating configmaps"

# Shared
apply_configmap shared-config            .env.shared.config

# Services
apply_configmap auth-service-config      .env.auth-service.config
apply_configmap user-service-config      .env.user-service.config
apply_configmap policy-service-config    .env.policy-service.config
apply_configmap bids-service-config      .env.bids-service.config
apply_configmap bidder-frontend-config   .env.bidder-frontend

echo "Creating file-based configmaps"

# Redis configs (mounted as volumes)
kubectl create configmap policy-cache-config \
  --from-file=redis.conf="$ENV_DIR/redis/policy_cache/redis.conf" \
  --dry-run=client -o yaml | kubectl apply -n "$NAMESPACE" -f -

kubectl create configmap users-cache-config \
  --from-file=redis.conf="$ENV_DIR/redis/users_cache/redis.conf" \
  --dry-run=client -o yaml | kubectl apply -n "$NAMESPACE" -f -

# MongoDB init scripts
kubectl create configmap policy-db-init-scripts \
  --from-file=01-create-policyuser.js="$ENV_DIR/policy_db/init-scripts/01-create-policyuser.js" \
  --dry-run=client -o yaml | kubectl apply -n "$NAMESPACE" -f -

# Traefik auth header plugin
kubectl create configmap traefik-authheader-plugin \
  --from-file=authheader.go="$ENV_DIR/traefik-plugin-authheader/authheader.go" \
  --from-file=go.mod="$ENV_DIR/traefik-plugin-authheader/go.mod" \
  --from-file=.traefik.yml="$ENV_DIR/traefik-plugin-authheader/.traefik.yml" \
  --dry-run=client -o yaml | kubectl apply -n "$NAMESPACE" -f -

echo "Done applying config"
