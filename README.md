# Augno API

Augno is a Go-based microservices platform using an API Gateway and domain-focused services coordinated via gRPC and RabbitMQ.

## Architecture

- **API Gateway**: Entry point for all public HTTP traffic.
- **Services**: Coordinate business logic and manage transaction boundaries.
- **Mediators**: Encapsulate reusable business logic steps.
- **Repositories**: Handle data persistence.
- **Transport**: Thin, stateless handlers for HTTP, gRPC, and RabbitMQ.

## Development

### Prerequisites

- Go 1.25+
- Docker & Tilt
- [sqlc](https://sqlc.dev/)
- [mockgen](https://github.com/uber-go/mock)
- [minikube](https://minikube.sigs.k8s.io/)

### Setup

```bash
make install-tools  # Install dev dependencies (goose, sqlc, mockgen, etc.)
make local-db       # Start MySQL & PostgreSQL in Docker, apply migrations, configure .env
minikube start      # Start a local Kubernetes cluster
make dev            # Spin up the environment with Tilt
```

### Local Databases

`make local-db` spins up both databases in Docker containers, applies all migrations, and writes connection strings to `.env` automatically:

- **MySQL 8** on port `3306` — core-service (`augno` database)
- **PostgreSQL 16** on port `5432` — agent-service (`augno_agents` database)

Data is persisted in named Docker volumes so it survives container restarts. `make local-db` also seeds the core database with sample data (accounts, users, items, orders, etc.) via the SQL files in `shared/db/seed/`.

```bash
# Re-seed core data (idempotent, safe to run multiple times)
make seed-core

# Seed with a specific plan (default: enterprise)
make seed-core ARGS="--plan starter"

# Seed agent definitions and tools
make seed-agents <account_id>

# Connect directly
mysql -u root -p'Testing123!' -h 127.0.0.1 --protocol=tcp augno
psql postgres://augno@localhost:5432/augno_agents

# Tear down containers (data preserved)
make local-db-down

# Tear down containers and delete all data (volumes) for a fresh start
# Run `make local-db` after to get clean databases with fresh migrations and seed data
make local-db-nuke
```

#### Using with the Dashboard

The dashboard API (Prisma) can connect to the same Docker MySQL instance. Add this to `dashboard/apps/api/.env`:

```typescript
DATABASE_URL="mysql://root:Testing123!@localhost:3306/augno"
```

#### Using with Tilt / Kubernetes

Services running in minikube cannot reach `localhost` on the host machine. The K8s secret in `infra/development/kubernetes/config/secrets.yaml` uses `host.minikube.internal` to route traffic back to the host. These URIs match the Docker Compose databases started by `make local-db`:


| Service            | URI                                                                         |
| ------------------ | --------------------------------------------------------------------------- |
| Core (MySQL)       | `root:Testing123!@tcp(host.minikube.internal:3306)/augno`                   |
| Agent (PostgreSQL) | `postgres://augno@host.minikube.internal:5432/augno_agents?sslmode=disable` |


The seed script hardcodes the Docker Compose connection details, so no `.env` configuration is needed — just run `make local-db` before `make dev`.

#### Agent endpoint-tools (optional)

Agents can invoke api-gateway endpoints flagged `AgentTool: true` (see `make gen-agent-tools`). This uses a dedicated **internal** api-gateway listener on port 8091, reached over the `api-gateway-internal` ClusterIP Service and gated by a shared token. The token is optional in dev: when it is absent the internal listener does not start and the endpoint-tools are simply unavailable (the rest of the agent works normally).

To enable it locally, add an `internal-service-token` secret to your `infra/development/kubernetes/config/secrets.yaml`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: internal-service-token
type: Opaque
stringData:
  token: "dev-internal-token" # any non-empty value in dev
```

Both `api-gateway` and `agent-service` already consume this secret (as `INTERNAL_SERVICE_TOKEN`) and the `API_GATEWAY_INTERNAL_URL` config value. In production the token is generated and delivered by Terraform (`infra/production/terraform/internal_service_token.tf`).

### Common Commands

Arguments like `auth`, `notification`, or `logging` can be passed to target specific services.

```bash
make sqlc [service]       # Generate database code
make proto                # Generate protobuf code
make mocks [service]          # Generate mock implementations
make test                     # Run all tests
```

## Development Process

### 1. Branching

Create a new branch from `main` for every feature or bug fix. Use descriptive names with the appropriate prefix:

```bash
# For new features
git checkout -b feature/your-feature-name

# For bug fixes
git checkout -b bug/your-bug-name
```

### 2. Committing

We use [Conventional Commits](https://www.conventionalcommits.org/) to maintain a clean history and automate versioning. Every commit message must start with a prefix that indicates the type of change:


| Prefix   | Type of Change   | Version Impact |
| -------- | ---------------- | -------------- |
| `fix:`   | Bug fixes        | Patch (0.0.x)  |
| `feat:`  | New features     | Minor (0.x.0)  |
| `feat!:` | Breaking changes | Major (x.0.0)  |


> **Note:** These prefixes are used by [release-please](https://github.com/googleapis/release-please) to automatically calculate the next version number.

### 3. Deploying Changes

1. **Open a Pull Request:** Once your work is complete, open a PR against the `main` branch.
2. **Merge:** After review and approval, merge your PR.
3. **Release PR:** `release-please` will automatically create or update a "Release PR" that aggregates all pending changes and updates the changelog.
4. **Production Release:** When ready to deploy, merge the "Release PR" into `main`. This triggers the final release process and deployment to production.

### 4. Notes

- minikube might need refreshed, try `minikube delete` and `minikube start`

