# Project Context Workflow for AI Development

**Document ID**: PVP-DOC-018  
**Version**: 1.0.0  

---

## 1. Overview

Tài liệu này mô tả quy trình tạo, duy trì, và sử dụng **Project Context** khi làm việc với AI (Gemini, Claude, GitHub Copilot). Context tốt = code output chất lượng cao hơn, ít hallucination hơn, consistent với codebase.

---

## 2. Nguyên tắc Context-First Development

```
Trước khi đưa task cho AI, luôn cung cấp:

1. WHAT   — Mục tiêu cụ thể của task
2. WHERE  — File nào, service nào, layer nào
3. HOW    — Pattern hiện tại trong project (paste example)
4. RULES  — Coding standard áp dụng
5. DEPS   — Dependencies có sẵn (interfaces, types)
```

**Rule vàng**: AI chỉ có thể cho output tốt nếu input context đủ tốt.

---

## 3. Cấu trúc Context Chuẩn

### Level 1 — Minimal Context (quick tasks)
```
Service: proxy-service
Layer: usecase
Task: {task description}
Existing code to reference: {paste 1 relevant file}
```

### Level 2 — Standard Context (feature tasks)
```
[System prompt từ PVP-DOC-017 section 2]

Current work context:
- Service: proxy-service
- Feature: Provider failover routing
- Files affected: 
  * internal/usecase/order_usecase.go (modify)
  * internal/domain/repository.go (add interface method)
  * internal/repository/postgres/provider_repo.go (implement)

Existing code:
--- internal/domain/repository.go ---
{paste current file}

--- internal/usecase/order_usecase.go ---
{paste current file}

Task: {detailed task description}
```

### Level 3 — Full Context (complex features)
Paste toàn bộ:
1. System prompt (PVP-DOC-017 section 2)
2. Tất cả files liên quan (domain, usecase, handler, repo)
3. Database schema của service
4. NATS events đang dùng
5. Mô tả chi tiết task

---

## 4. Workflow: Bắt đầu Task Mới

### Bước 1 — Đọc tài liệu liên quan

```bash
# Trước khi viết code, đọc lại:
cat docs/services/05-PROXY-SERVICE.md    # nếu làm proxy-service
cat docs/standards/13-CODING-STANDARD-GO.md
cat docs/02-DATABASE-DESIGN.md           # nếu thêm DB queries
```

### Bước 2 — Tạo Context File

```bash
# Script tự động tạo context cho 1 service
./scripts/gen-context.sh proxy-service > /tmp/context-proxy.txt
```

Nội dung script `gen-context.sh`:
```bash
#!/bin/bash
SERVICE=$1
echo "=== SYSTEM PROMPT ===" 
cat docs/ai/SYSTEM_PROMPT.txt

echo ""
echo "=== SERVICE: $SERVICE ==="
echo "--- Service Design ---"
cat docs/services/*${SERVICE}*.md 2>/dev/null

echo ""
echo "--- Domain ---"
cat services/${SERVICE}/internal/domain/*.go 2>/dev/null

echo ""
echo "--- Usecases ---"
cat services/${SERVICE}/internal/usecase/*.go 2>/dev/null

echo ""
echo "--- Current Handlers ---"
cat services/${SERVICE}/internal/handler/http/*.go 2>/dev/null
```

### Bước 3 — Craft Prompt

Dùng template phù hợp từ PVP-DOC-017 (section 3).
Điền đầy đủ thông tin, không để placeholder.

### Bước 4 — Review AI Output

Checklist khi nhận code từ AI:

```
✅ LUÔN kiểm tra:
□ Import paths đúng với project structure
□ Error handling đầy đủ (không có `_` discard)
□ Context propagated
□ Slog logger dùng đúng format
□ NATS events published nếu state thay đổi
□ Interface đúng (không implement trực tiếp vào usecase)
□ Test coverage hợp lý

⚠️ CẢNH BÁO AI hay sai:
□ Hay generate global variables → refactor sang DI
□ Hay dùng log.Printf thay vì slog
□ Hay quên context.WithTimeout cho external calls
□ Hay viết business logic trong handler (sai layer)
□ Hay generate `panic` trong business logic
```

### Bước 5 — Integrate & Test

```bash
# Sau khi nhận code từ AI:
go build ./...              # check compile
go vet ./...                # static analysis
golangci-lint run           # full lint
go test ./...               # run tests
```

---

## 5. Workflow: Bảo Trì Context (Luôn Cập Nhật)

### Khi nào cần update docs?

| Thay đổi | Docs cần update |
|---|---|
| Thêm API endpoint | `docs/services/{service}.md` |
| Thêm DB table/column | `docs/02-DATABASE-DESIGN.md` |
| Thêm NATS event | `docs/01-ARCHITECTURE.md` (events table) + service doc |
| Thêm provider | `docs/infrastructure/12-PROVIDER-ADAPTERS.md` |
| Thay đổi coding standard | `docs/standards/13-CODING-STANDARD-GO.md` |
| Thêm AI prompt mới | `docs/ai/17-AI-PROMPT-GEMINI.md` |

**Rule**: Không merge PR nếu code thay đổi mà docs chưa được update.

---

## 6. CONTEXT.md — Living Document

Tạo file `CONTEXT.md` ở root, cập nhật sau mỗi sprint:

```markdown
# CONTEXT.md — Current Development State
Last updated: 2025-01-15

## What's built (Production-ready)
- User auth service: registration, login, JWT, API keys ✅
- Billing service: wallet, deposits, proxy order charging ✅

## What's in progress
- Proxy service: provider adapter for SmartProxy (80% done)
- Log service: WebSocket streaming (in review)

## What's not built yet  
- VPS service
- Reseller service
- Proxmox adapter

## Recent decisions
- 2025-01-10: Switched from pgx directly to sqlx for better query ergonomics
- 2025-01-08: NATS JetStream consumer changed to pull-based (better backpressure)

## Known issues / tech debt
- Billing deduction doesn't use DB-level advisory lock yet (task #PVP-198)
- Provider adapter timeout not configurable per-provider yet

## Current DB schema notes
- Wallet balance uses DECIMAL(14,4) not FLOAT to avoid floating point issues
- Log entries partitioned by month — remember to create next month's partition
```

---

## 7. Prompt cho Tạo/Cập Nhật CONTEXT.md

```
Based on the following recent git log and changed files, 
update the CONTEXT.md for this project:

Git log (last 2 weeks):
{git log --oneline --since="2 weeks ago"}

Current CONTEXT.md:
{paste current CONTEXT.md}

Generate an updated CONTEXT.md that:
1. Moves completed items to "What's built"
2. Updates "What's in progress" based on recent commits
3. Adds new "Recent decisions" from commit messages
4. Notes any new tech debt or known issues mentioned in commits
Keep it concise and practical — this is used as AI context input.
```

---

## 8. Multi-file Context với Gemini 1.5 Pro

Gemini 1.5 Pro hỗ trợ 1M token context — có thể upload toàn bộ codebase:

```bash
# Tạo single context file từ toàn bộ service
find services/proxy-service -name "*.go" | while read f; do
    echo "=== $f ==="
    cat "$f"
    echo ""
done > /tmp/full-proxy-service-context.txt

# Upload lên AI Studio → attach file → ask questions
```

Sau đó prompt:
```
I've provided the complete source code for proxy-service above.

Task: {task description}

Please implement this following all existing patterns in the codebase.
Reference specific existing code where relevant.
```

---

## 9. Gemini API Integration (Programmatic)

Cho automation tasks (generate migrations, generate tests tự động):

```go
// scripts/ai/generate_test.go
package main

import (
    "context"
    "fmt"
    "os"
    
    "google.golang.org/genai"
)

func generateTests(sourceFile string) error {
    ctx := context.Background()
    client, _ := genai.NewClient(ctx, &genai.ClientConfig{
        APIKey: os.Getenv("GEMINI_API_KEY"),
    })
    
    systemPrompt, _ := os.ReadFile("docs/ai/SYSTEM_PROMPT.txt")
    source, _ := os.ReadFile(sourceFile)
    
    resp, _ := client.Models.GenerateContent(ctx,
        "gemini-1.5-flash",
        genai.Text(fmt.Sprintf(`%s
        
Generate comprehensive unit tests for this file:
%s`, systemPrompt, source)),
        nil,
    )
    
    testFile := strings.Replace(sourceFile, ".go", "_test.go", 1)
    return os.WriteFile(testFile, []byte(resp.Text()), 0644)
}
```
