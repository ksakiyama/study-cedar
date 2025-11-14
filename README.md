# Cedar Sample Project

A sample project for learning policy-based access control using AWS Cedar.
Learn the basics of Cedar through a document management system built with Go backend + PostgreSQL + Docker.

## What is Cedar?

[Cedar](https://www.cedarpolicy.com/) is an open-source policy-based authorization engine provided by AWS.

### Key Features

- **Policy-based**: Separate authorization logic from code, define with declarative policies
- **Fast**: Low latency and scales to large deployments
- **Type-safe**: Static analysis and validation through schema
- **Analyzable**: Automatically verify policy correctness

## Project Structure

```
study-cedar/
├── api/
│   └── openapi.yaml              # OpenAPI specification
├── cmd/
│   └── server/
│       └── main.go               # Main server
├── internal/
│   ├── api/
│   │   └── handlers.go           # API handlers
│   ├── cedar/
│   │   ├── authorizer.go         # Cedar authorization logic
│   │   └── policies/
│   │       ├── policy.cedar      # Cedar policies
│   │       └── schema.cedarschema # Cedar schema
│   └── models/
│       └── models.go             # Data models
├── scripts/
│   └── init.sql                  # Database initialization script
├── docker-compose.yml            # Docker Compose configuration
├── Dockerfile                    # Docker image definition
└── go.mod                        # Go dependencies
```

## Tech Stack

- **Language**: Go 1.23
- **Database**: PostgreSQL 16
- **Router**: chi
- **Authorization Engine**: Cedar (cedar-go v1.3.0)
- **Containers**: Docker & Docker Compose
- **API Specification**: OpenAPI 3.0

## Setup

### Prerequisites

- Docker
- Docker Compose

### Running the Application

1. Navigate to the project directory

```bash
cd /Users/ksakiyama/Projects/study-cedar
```

2. Build and start Docker containers

```bash
docker-compose up --build
```

The server will start at `http://localhost:8080`.

### Health Check

```bash
# Health check
curl http://localhost:8080/health
```

## API Usage Examples

This sample includes three roles:

- **admin**: Can perform all operations
- **editor**: Can create, update, and view documents
- **viewer**: Can only view documents

### 1. List Documents

```bash
# List as admin
curl -H "X-User-ID: user-1" \
     -H "X-User-Role: admin" \
     http://localhost:8080/api/v1/documents

# List as viewer
curl -H "X-User-ID: user-3" \
     -H "X-User-Role: viewer" \
     http://localhost:8080/api/v1/documents
```

### 2. Get Document

```bash
curl -H "X-User-ID: user-1" \
     -H "X-User-Role: viewer" \
     http://localhost:8080/api/v1/documents/doc-1
```

### 3. Create Document (Editor permission required)

```bash
# Create as editor → Success
curl -X POST \
     -H "X-User-ID: user-2" \
     -H "X-User-Role: editor" \
     -H "Content-Type: application/json" \
     -d '{"title":"New Document","content":"Test content"}' \
     http://localhost:8080/api/v1/documents

# Create as viewer → Denied
curl -X POST \
     -H "X-User-ID: user-3" \
     -H "X-User-Role: viewer" \
     -H "Content-Type: application/json" \
     -d '{"title":"New Document","content":"Test content"}' \
     http://localhost:8080/api/v1/documents
```

### 4. Update Document (Editor permission required)

```bash
curl -X PUT \
     -H "X-User-ID: user-2" \
     -H "X-User-Role: editor" \
     -H "Content-Type: application/json" \
     -d '{"title":"Updated Title","content":"Updated content"}' \
     http://localhost:8080/api/v1/documents/doc-1
```

### 5. Delete Document (Admin or Owner)

```bash
# Delete as owner → Success
curl -X DELETE \
     -H "X-User-ID: user-1" \
     -H "X-User-Role: editor" \
     http://localhost:8080/api/v1/documents/doc-1

# Delete as admin → Success
curl -X DELETE \
     -H "X-User-ID: user-admin" \
     -H "X-User-Role: admin" \
     http://localhost:8080/api/v1/documents/doc-2

# Delete as other user → Denied
curl -X DELETE \
     -H "X-User-ID: user-3" \
     -H "X-User-Role: editor" \
     http://localhost:8080/api/v1/documents/doc-1
```

## Cedar Policies Explained

Policies defined in `internal/cedar/policies/policy.cedar`:

### Policy 1: Admins can perform all operations

```cedar
permit(
    principal,
    action,
    resource
)
when {
    principal.role == "admin"
};
```

Users with the `admin` role are granted access to all actions and resources.

### Policy 2: Editor permissions

```cedar
permit(
    principal,
    action in [
        DocumentApp::Action::"ListDocuments",
        DocumentApp::Action::"GetDocument",
        DocumentApp::Action::"CreateDocument",
        DocumentApp::Action::"UpdateDocument"
    ],
    resource
)
when {
    principal.role == "editor"
};
```

The `editor` role can list, view, create, and update documents (but not delete).

### Policy 3: Viewer permissions

```cedar
permit(
    principal,
    action in [
        DocumentApp::Action::"ListDocuments",
        DocumentApp::Action::"GetDocument"
    ],
    resource
)
when {
    principal.role == "viewer"
};
```

The `viewer` role can only list and view documents.

### Policy 4: Owner can delete their documents

```cedar
permit(
    principal,
    action == DocumentApp::Action::"DeleteDocument",
    resource
)
when {
    resource.owner == principal
};
```

Document owners (creators) can delete their own documents.

## Cedar Learning Points

### 1. Entities and Actions

Cedar defines the following elements:

- **Principal**: The entity performing the action (user)
- **Action**: The operation being performed (ListDocuments, CreateDocument, etc.)
- **Resource**: The target of the operation (document)

### 2. Schema-based Type Definition

Define entity structure in `schema.cedarschema`:

```cedar
entity User = {
    "role": String,
};

entity Document = {
    "owner": User,
};
```

### 3. Policy Evaluation

For each request, the Cedar engine evaluates the policy set and returns:

- **Allow**: At least one policy permits and no policies deny
- **Deny**: At least one policy explicitly denies
- **Deny (implicit)**: No policies permit (default deny)

### 4. Attribute-Based Access Control (ABAC)

Cedar performs access control based on attributes (role, owner, etc.).
This enables flexible and fine-grained permission management.

## Kubernetes Graceful Shutdown

This application is designed to support graceful shutdown for Kubernetes deployments.

### How It Works

1. **SIGTERM Handling**: When the application receives a SIGTERM signal (sent by Kubernetes during pod termination), it:
   - Immediately sets the shutdown flag
   - Returns `503 Service Unavailable` from the `/health` endpoint
   - Stops accepting new connections
   - Waits up to 30 seconds for existing requests to complete
   - Shuts down gracefully

2. **Health Check Behavior**:
   - **Normal operation**: `/health` returns `200 OK` with `{"status": "ok"}`
   - **During shutdown**: `/health` returns `503 Service Unavailable` with `{"status": "shutting_down"}`

### Kubernetes Configuration Example

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: cedar-app
    image: study-cedar-app:latest
    ports:
    - containerPort: 8080
    livenessProbe:
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 10
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 5
    lifecycle:
      preStop:
        exec:
          command: ["/bin/sh", "-c", "sleep 5"]
  terminationGracePeriodSeconds: 35
```

### Testing Graceful Shutdown

Run the test script to verify graceful shutdown behavior:

```bash
./test-graceful-shutdown.sh
```

Expected behavior:
1. Health check returns `200 OK` during normal operation
2. After receiving SIGTERM, health check immediately returns `503`
3. Server waits for in-flight requests to complete
4. Server logs show "Health check now returning 503" and "Server stopped gracefully"

## Troubleshooting

### Database Connection Error

```bash
# Check container status
docker-compose ps

# Check logs
docker-compose logs postgres
docker-compose logs app
```

### Policy Errors

Policy syntax errors will be displayed in the logs at startup:

```bash
docker-compose logs app | grep -i error
```

## Stopping the Application

```bash
# Stop containers
docker-compose down

# Remove containers and volumes
docker-compose down -v
```

## References

- [Cedar Official Documentation](https://docs.cedarpolicy.com/)
- [Cedar Playground](https://www.cedarpolicy.com/playground)
- [cedar-go GitHub](https://github.com/cedar-policy/cedar-go)
- [Cedar Language Specification](https://docs.cedarpolicy.com/policies/syntax.html)

## License

This sample project is created for educational purposes.
