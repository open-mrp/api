# Augno API

Augno is a Go-based microservices platform using an API Gateway and domain-focused services coordinated via gRPC and RabbitMQ.

## Architecture

-   **API Gateway**: Entry point for all public HTTP traffic.
-   **Services**: Coordinate business logic and manage transaction boundaries.
-   **Mediators**: Encapsulate reusable business logic steps.
-   **Repositories**: Handle data persistence.
-   **Transport**: Thin, stateless handlers for HTTP, gRPC, and RabbitMQ.

## Development

### Prerequisites

-   Go 1.25+
-   Docker & Tilt
-   [sqlc](https://sqlc.dev/)
-   [mockgen](https://github.com/uber-go/mock)

### Setup

```bash
make install-tools  # Install dev dependencies
make dev            # Spin up the environment with Tilt
```

### Common Commands

Arguments like `auth`, `notification`, or `logging` can be passed to target specific services.

```bash
make gen-sqlc [service]       # Generate database code
make gen-proto                # Generate protobuf code
make mocks [service]          # Generate mock implementations
make test                     # Run all tests
make migrate-up               # Run database migrations
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
| :------- | :--------------- | :------------- |
| `fix:`   | Bug fixes        | Patch (0.0.x)  |
| `feat:`  | New features     | Minor (0.x.0)  |
| `feat!:` | Breaking changes | Major (x.0.0)  |

> **Note:** These prefixes are used by [release-please](https://github.com/googleapis/release-please) to automatically calculate the next version number.

### 3. Deploying Changes

1.  **Open a Pull Request:** Once your work is complete, open a PR against the `main` branch.
2.  **Merge:** After review and approval, merge your PR.
3.  **Release PR:** `release-please` will automatically create or update a "Release PR" that aggregates all pending changes and updates the changelog.
4.  **Production Release:** When ready to deploy, merge the "Release PR" into `main`. This triggers the final release process and deployment to production.
