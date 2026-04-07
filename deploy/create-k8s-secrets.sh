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
apply_secret auth-db-secret          .env.auth_db
apply_secret users-db-secret         .env.users_db
apply_secret policy-db-secret        .env.policy_db

# Services
apply_secret auth-service-secret     .env.auth_service.secret
apply_secret user-service-secret     .env.user_service.secret
apply_secret policy-service-secret   .env.policy_service.secret
apply_secret bids-service-secret     .env.bids_service.secret
apply_secret traefik-secret          .env.traefik

echo "Creating configmaps"

# Shared
apply_configmap shared-config            .env.shared.config

# Services
apply_configmap auth-service-config      .env.auth_service.config
apply_configmap user-service-config      .env.user_service.config
apply_configmap policy-service-config    .env.policy_service.config
apply_configmap bids-service-config      .env.bids_service.config
apply_configmap bidder-frontend-config   .env.bidder_frontend

echo "Done applying config"
