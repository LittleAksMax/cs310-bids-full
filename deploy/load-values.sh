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
export AUTH_SERVICE_VERSION=$(yq '.images["auth-service"].version' "$VALUES")
export USER_SERVICE_VERSION=$(yq '.images["user-service"].version' "$VALUES")
export POLICY_SERVICE_VERSION=$(yq '.images["policy-service"].version' "$VALUES")
export BIDS_SERVICE_VERSION=$(yq '.images["bids-service"].version' "$VALUES")
export FRONTEND_VERSION=$(yq '.images.frontend.version' "$VALUES")

# Domain
export BACKEND_DOMAIN=$(yq '.domain.backend' "$VALUES")
export FRONTEND_DOMAIN=$(yq '.domain.frontend' "$VALUES")

# Storage host paths
export AUTH_DB_HOST_PATH=$(yq '.storage["auth-db"]' "$VALUES")
export USERS_DB_HOST_PATH=$(yq '.storage["users-db"]' "$VALUES")
export POLICY_DB_HOST_PATH=$(yq '.storage["policy-db"]' "$VALUES")
export POLICY_CACHE_HOST_PATH=$(yq '.storage["policy-cache"]' "$VALUES")
export USERS_CACHE_HOST_PATH=$(yq '.storage["users-cache"]' "$VALUES")

echo "Loaded values from $VALUES"