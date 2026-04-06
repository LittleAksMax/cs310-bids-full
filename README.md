# Automated Bidding System for Amazon Ads

This is a microservices-based platform for automating bid management across Amazon Ads seller profiles. Users can define bidding policies using a custom DSL (bidscript), attach them to seller profiles, and schedule automated bid adjustments based on real-time campaign performance data.

## Project Structure

### Services

- **`auth-service/`** - Authentication and JWT token management.
- **`user-service/`** - User data, Amazon OAuth, campaign management, and schedule coordination.
- **`policy-service/`** - Bidding policy storage and bidscript compilation.
- **`bids-service/`** - Bid execution engine that evaluates policies against campaign data.
- **`bidder-frontend/`** - React/TypeScript UI for managing policies, schedules, and accounts.
- **`bid-consumer/`** - Legacy. Was originally planned as a message consumer but turned out to be redundant, so it was removed.

### Libraries

- **`amazon-ads-api-go-sdk/`** - Go SDK wrapping the Amazon Ads API (profiles, campaigns, ad groups, targets, reporting).
- **`bidscript/`** - A small DSL for expressing rule-based bid adjustments. Includes a lexer, parser, and evaluator.
- **`bids-util/`** - Shared Go utility packages used across backend services.

### Middleware

- **`traefik/`** - Traefik reverse proxy configuration. Acts as the API gateway.
- **`traefik-plugin-authheader/`** - Custom Traefik plugin that validates JWTs and transforms them into signed internal headers.

### Infrastructure & Architecture

- **`compose.yml`** - Docker Compose file that orchestrates the full stack. Main entry point for running locally.
- **`nginx/`** - Nginx reverse proxy for SSL/TLS termination.
- **`certs/`** - Locally-trusted TLS certificates for the LwA flow. See [certs README](./certs/README.md).
- **`k8s/`** - Kubernetes deployment configurations.
- **`postgres/`** - Persistent data directories for Postgres databases (`auth_db`, `users_db`).
- **`redis/`** - Persistent data directories for Redis caches (`policy_cache`, `users_cache`).
- **`policy_db/`** - MongoDB data directory and initialisation scripts for the policy database.

---

## Environment Files

Each service requires its own environment file(s) to run -- referenced in `compose.yml`. Example configurations can be found in the `.env.example` file within each service's respective project folder.

The following `.env` files are expected in the project root:

- `.env.auth_db`
- `.env.auth_service`
- `.env.bidder_frontend`
- `.env.bids_db`
- `.env.bids_service`
- `.env.policy_db`
- `.env.policy_service`
- `.env.rabbitmq`
- `.env.shared`
- `.env.traefik`
- `.env.user_service`
- `.env.users_db`

---

## Submodules

Most of the services and libraries in this project live in their own Git repositories and are pulled in as submodules. After cloning the main repository, you'll need to initialise and pull them:

```bash
git submodule update --init --recursive
```

If you've already cloned but the submodule folders are empty, the above command will sort that out. To pull the latest changes across all submodules later on, add the `--remote` flag:

```bash
git submodule update --init --remote --recursive
```

---

## Running Locally

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- [mkcert](https://github.com/FiloSottile/mkcert) (for generating local TLS certificates)

### Steps

1. Clone the repository and pull submodules:

```bash
git clone <repo-url>
cd cs310
git submodule update --init --recursive
```

2. Set up your `.env` files based on the `.env.example` templates in each service folder (see [Environment Files](#environment-files)).

3. Generate local TLS certificates (if not already present). You may want to check the Nginx configurations to make sure the file is being pointed to correctly.

```bash
cd certs
mkcert localhost 127.0.0.1 ::1
```

4. Spin everything up:

```bash
docker compose up --build
```

That should bring up all the services, databases, caches, and the reverse proxy stack.

---

## Kubernetes

Kubernetes configurations can be generated from the `compose.yml` using [Kompose](https://kompose.io/):

```bash
brew install kompose
kompose convert -f compose.yml -o k8s/
```

This section is a work in progress -- K8s deployment hasn't been fully set up yet.
