if [ $# -ne 1 ]; then
  echo "Usage: source load-values.sh <values.yaml>" >&2
  return 1 2>/dev/null || exit 1
fi

VALUES="$1"

if [ ! -f "$VALUES" ]; then
  echo "Error: $VALUES not found" >&2
  return 1 2>/dev/null || exit 1
fi

# AWS / ECR
export AWS_ACCOUNT_ID=$(yq '.aws.accountId' "$VALUES")
export AWS_REGION=$(yq '.aws.region' "$VALUES")
export ECR_PULL_SECRET=$(yq '.aws.pullSecret' "$VALUES")

# Image versions
export AUTH_SERVICE_VERSION=$(yq '.images.auth_service.version' "$VALUES")
export USER_SERVICE_VERSION=$(yq '.images.user_service.version' "$VALUES")
export POLICY_SERVICE_VERSION=$(yq '.images.policy_service.version' "$VALUES")
export BIDS_SERVICE_VERSION=$(yq '.images.bids_service.version' "$VALUES")
export FRONTEND_VERSION=$(yq '.images.bidder_frontend.version' "$VALUES")

# Domain
export BACKEND_DOMAIN=$(yq '.domain.backend' "$VALUES")
export FRONTEND_DOMAIN=$(yq '.domain.frontend' "$VALUES")

# Storage host paths
export AUTH_DB_HOST_PATH=$(yq '.storage.auth_db' "$VALUES")
export USERS_DB_HOST_PATH=$(yq '.storage.users_db' "$VALUES")
export POLICY_DB_HOST_PATH=$(yq '.storage.policy_db' "$VALUES")
export POLICY_CACHE_HOST_PATH=$(yq '.storage.policy_cache' "$VALUES")
export USERS_CACHE_HOST_PATH=$(yq '.storage.users_cache' "$VALUES")

echo "Loaded values from $VALUES"