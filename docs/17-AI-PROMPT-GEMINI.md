# AI Prompts — Google Gemini / AI Studio

**Document ID**: PVP-DOC-017  
**Version**: 1.0.0  
**Tools**: Google AI Studio, Gemini 1.5 Pro/Flash, Gemini Code Assist  

---

## 1. Overview

File này chứa prompt templates tối ưu hóa cho việc dùng **Google Gemini** (AI Studio / Gemini Code Assist) trong phát triển dự án ProxyVPS Platform. Mỗi prompt đã được thiết kế với context đầy đủ để Gemini cho output chất lượng cao nhất.

---

## 2. System Prompt — Project Context

**Dùng làm System Instruction trong AI Studio hoặc đặt đầu mỗi conversation.**

```
You are a senior Go backend engineer working on ProxyVPS Platform — an enterprise
proxy and VPS reselling system.

## Tech Stack
- Language: Go 1.22 with microservices architecture
- Message Bus: NATS JetStream (event-driven, async operations)
- Database: PostgreSQL 16 (schema-per-service pattern)
- Cache: Redis 7
- Infrastructure: Proxmox VE 8.x (10 nodes, REST API)
- Container: Docker / Docker Compose

## Architecture Principles
- Dependency injection via constructors (no global state)
- Repository pattern with interfaces (domain/repository.go)
- Usecase layer contains ALL business logic
- Handlers only parse input and call usecases
- Every state change publishes a NATS event
- Every NATS event consumed by log-service for audit trail
- Idempotency keys on all order operations
- Saga pattern for multi-step operations (compensating transactions)

## Code Standards
- All errors wrapped with fmt.Errorf("context: %w", err)
- Context passed as first argument to all functions
- Structured logging via slog (JSON format)
- Named return values ONLY when it improves clarity
- Interfaces prefixed with I: IUserRepository, IBillingService
- Error vars prefixed with Err: ErrNotFound, ErrInsufficientFunds

## Services
- api-gateway:8080, user-service:8081, proxy-service:8082
- vps-service:8083, reseller-service:8084, billing-service:8085
- notification-service:8086, log-service:8087

When writing code:
1. Follow the project structure: cmd/ internal/{domain,usecase,handler,repository,events}/
2. Write production-quality code, not examples
3. Include proper error handling and context propagation
4. Add structured log statements at key points
5. Publish NATS events for state changes
6. Always write the full file, not snippets
```

---

## 3. Prompt Templates

### 3.1 Generate New Service Boilerplate

```
Using the project context above, generate the complete boilerplate for 
a new Go microservice called "{service-name}" with the following responsibilities:

{describe responsibilities}

Generate these files:
1. cmd/server/main.go - wire dependencies, start HTTP server
2. internal/config/config.go - env config struct with validation
3. internal/domain/entity.go - domain entities
4. internal/domain/repository.go - repository interfaces
5. internal/domain/errors.go - domain error variables
6. internal/usecase/{primary_usecase}.go - main business logic
7. internal/handler/http/handler.go - HTTP handler struct + methods
8. internal/handler/http/router.go - chi router setup
9. internal/repository/postgres/{entity}_repo.go - PostgreSQL implementation
10. internal/events/publisher.go - NATS event publisher

Use the exact project structure, coding standards, and patterns described
in the system context. Include all imports. Make it production-ready.
```

---

### 3.2 Implement Specific Usecase

```
Context: ProxyVPS Platform, {service-name}
File: internal/usecase/{usecase_name}.go

Implement the following usecase:

Function: {FunctionName}
Input: {describe input struct/params}
Output: {describe output}

Business rules:
1. {rule 1}
2. {rule 2}
3. {rule 3}

Dependencies available (via constructor injection):
- {IRepositoryName}: {describe}
- {IServiceName}: {describe}
- eventPub: IEventPublisher
- logger: *slog.Logger

Requirements:
- Full error handling with wrapped errors
- Context propagation
- Publish NATS event: {event.name} on success
- Log at INFO level on success, ERROR on failure
- Include structured log fields: request_id from ctx, relevant IDs

Write the complete usecase file.
```

---

### 3.3 Generate Database Migration

```
Context: ProxyVPS Platform
Schema: {schema_name}

Generate a PostgreSQL migration file for the following requirement:

{describe what needs to change}

Current table structure (if modifying):
{paste current CREATE TABLE}

Requirements:
- Use schema prefix: {schema_name}.{table_name}
- Include UP and DOWN migrations
- Add appropriate indexes for query patterns:
  {list expected query patterns}
- Use IF NOT EXISTS / IF EXISTS for safety
- Add comments on non-obvious columns
- Follow the project's naming convention (snake_case)

Output format:
-- migrate:up
{sql}

-- migrate:down
{sql}
```

---

### 3.4 Generate REST API Handler

```
Context: ProxyVPS Platform, {service-name}
File: internal/handler/http/handler.go

Generate HTTP handler(s) for:

Endpoint: {METHOD} /api/v1/{path}
Auth required: {yes/no, role required}
Description: {what this endpoint does}

Request body:
{json or struct}

Response (success):
{json or struct}

Usecase to call: {UsecaseName}.{MethodName}

Requirements:
- Parse and validate request body/params
- Extract user_id and request_id from headers (set by API Gateway)
- Call usecase
- Use respondJSON/respondError helpers
- Handle common errors: ErrNotFound→404, ErrForbidden→403, etc.
- Return appropriate HTTP status codes per API Design Standard

Write the complete handler method.
```

---

### 3.5 Generate NATS Event Consumer

```
Context: ProxyVPS Platform, {service-name}
File: internal/events/consumer.go

Generate a NATS JetStream consumer for:

Events to consume:
- {event.topic.one}: {description, what to do when received}
- {event.topic.two}: {description}

For each event:
- Parse event payload (JSON)
- Call appropriate usecase
- Acknowledge on success
- NAK with delay on transient errors (retry)
- Log ERROR + no NAK on permanent errors (dead letter)

Use durable consumer with consumer name: "{service-name}-consumer"
Include proper error handling and structured logging.
```

---

### 3.6 Write Unit Tests

```
Context: ProxyVPS Platform
File to test: {path/to/file.go}

{paste the source code}

Generate comprehensive unit tests for all exported functions/methods.

Requirements:
- Use testify (assert, require, mock)
- Test naming: TestTypeName_MethodName_Scenario
- Test structure: Arrange → Act → Assert
- Cover: happy path, error cases, edge cases, boundary conditions
- Mock all external dependencies (repositories, services, providers)
- Test error propagation
- No real DB/network calls

For each test:
- Describe what scenario is being tested
- Assert both return values and side effects (mock expectations)
```

---

### 3.7 Code Review

```
Please review the following Go code for the ProxyVPS Platform.

Context: {which service, what it does}

{paste code}

Review for:
1. Correctness: logic errors, edge cases not handled
2. Error handling: errors not checked, not wrapped with context
3. Security: credential logging, SQL injection, race conditions
4. Performance: N+1 queries, missing indexes, unnecessary allocations
5. Standards compliance: project coding standards (context propagation,
   DI, no global state, structured logging, event publishing)
6. Missing tests: what should be tested

Format response as:
CRITICAL: (must fix before merge)
IMPORTANT: (should fix)
SUGGESTION: (nice to have)
GOOD: (what's done well)
```

---

### 3.8 Debug / Troubleshoot

```
Context: ProxyVPS Platform
Service: {service-name}
Environment: {dev/staging/production}

Problem description:
{describe the bug/issue}

Error message / stack trace:
{paste error}

Relevant code:
{paste code}

Relevant logs:
{paste logs}

Please:
1. Identify the root cause
2. Explain why it happens
3. Provide the fix with code
4. Suggest how to prevent this class of bug in the future
```

---

## 4. AI Studio Configuration

### Recommended Settings
```
Model: gemini-1.5-pro-latest    ← for complex architecture tasks
Model: gemini-1.5-flash-latest  ← for repetitive code generation (faster)
Temperature: 0.2                ← lower = more deterministic code
Max output tokens: 8192
Top-P: 0.95
```

### Grounding
- Enable "Google Search grounding" khi cần thông tin về libraries/APIs mới nhất
- Tắt grounding khi generate pure code từ project context (tiết kiệm quota)

---

## 5. Gemini Code Assist (IDE Integration)

### Setup trong VS Code / GoLand
1. Cài Gemini Code Assist extension
2. Mở file `.gemini/styleguide.md` trong repo root (tạo file này)
3. Gemini sẽ tự học style từ codebase

### `.gemini/styleguide.md` template
```markdown
# Code Style for ProxyVPS Platform

This is a Go microservices project. Follow these rules:
- Use constructor injection, never global variables
- All errors must be wrapped: fmt.Errorf("context: %w", err)
- Use slog for structured logging
- Repository interfaces in domain/, implementations in repository/postgres/
- Publish NATS events for all state changes
- Context is always first parameter
```
