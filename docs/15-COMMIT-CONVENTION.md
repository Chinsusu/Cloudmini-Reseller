# Commit Convention — ProxyVPS Platform

**Document ID**: PVP-DOC-015  
**Version**: 1.0.0  

---

## 1. Format

Dự án dùng **Conventional Commits** (https://www.conventionalcommits.org).

```
<type>(<scope>): <subject>

[optional body]

[optional footer(s)]
```

### Example
```
feat(proxy-service): add provider failover routing

When primary provider is unavailable, router now tries next
provider in priority order instead of returning error immediately.

Closes #142
```

---

## 2. Types

| Type | When to use |
|---|---|
| `feat` | Tính năng mới |
| `fix` | Bug fix |
| `refactor` | Refactor code, không thêm tính năng, không fix bug |
| `perf` | Cải thiện hiệu suất |
| `test` | Thêm hoặc sửa tests |
| `docs` | Chỉ thay đổi documentation |
| `style` | Format, whitespace (không ảnh hưởng logic) |
| `chore` | Build scripts, CI, dependencies, tool config |
| `ci` | CI/CD pipeline changes |
| `revert` | Revert commit trước |
| `security` | Security fix hoặc improvement |
| `db` | Database migration, schema changes |

---

## 3. Scopes

Scope = tên service hoặc component bị ảnh hưởng.

| Scope | Description |
|---|---|
| `api-gateway` | API Gateway service |
| `user-service` | User/Auth service |
| `proxy-service` | Proxy service |
| `vps-service` | VPS service |
| `billing-service` | Billing service |
| `log-service` | Log service |
| `notification-service` | Notification service |
| `reseller-service` | Reseller service |
| `proxmox-adapter` | Proxmox integration |
| `provider-adapters` | Proxy provider adapters |
| `db` | Database migrations |
| `deploy` | Deployment configs |
| `docs` | Documentation |
| `ci` | CI/CD |
| `pkg/{name}` | Shared packages |

---

## 4. Subject Rules

- Bắt đầu bằng **động từ nguyên mẫu** (lowercase): `add`, `fix`, `update`, `remove`, `improve`
- **Không** viết hoa chữ đầu
- **Không** có dấu chấm ở cuối
- **Tiếng Anh** (subject + body)
- **Tối đa 72 ký tự** cho subject line

```
# Good
feat(proxy-service): add idempotency key validation
fix(billing-service): prevent negative wallet balance on concurrent deduct
refactor(vps-service): extract node selection into separate strategy

# Bad
feat(proxy-service): Added Idempotency Key Validation.  ← capitalized + period
fix: fix bug                                             ← no scope, vague
FEAT: NEW FEATURE                                        ← all caps
```

---

## 5. Body

- Viết trong 72 ký tự mỗi dòng
- Giải thích **WHY** (lý do), không chỉ **WHAT**
- Cách subject 1 dòng trắng

```
fix(billing-service): prevent race condition in wallet deduction

Previously, concurrent orders could both pass balance check before
either deduction was committed, resulting in negative balance.

Fixed by using SELECT FOR UPDATE in the deduction transaction to
serialize concurrent access to the wallet row.
```

---

## 6. Footer

```
# Reference issues
Closes #142
Fixes #89
Refs #201

# Breaking changes (requires major version bump)
BREAKING CHANGE: wallet API response format changed.
`balance` field renamed to `available_balance`.
Migration: update all clients using the wallet endpoint.

# Co-authors
Co-authored-by: Name <email@example.com>
```

---

## 7. Breaking Changes

Breaking change phải:
1. Có `BREAKING CHANGE:` trong footer
2. Subject có `!` sau scope: `feat(billing-service)!: rename balance field`
3. Được announce trước ít nhất 1 sprint

---

## 8. Commit Examples

```bash
# New feature
git commit -m "feat(proxy-service): add residential proxy product type"

# Bug fix with issue reference
git commit -m "fix(vps-service): resolve node selection panic when cluster empty

SelectNode was panicking with index out of range when all nodes
were offline. Added empty candidates check with proper error return.

Fixes #156"

# Database migration
git commit -m "db(proxy-service): add index on orders.user_id and status"

# Documentation
git commit -m "docs(proxmox-adapter): add WaitForTask timeout configuration"

# Chore
git commit -m "chore(deps): upgrade chi router v5.0.11 to v5.0.12"

# Security fix
git commit -m "security(user-service): increase bcrypt cost from 10 to 12"

# Refactor
git commit -m "refactor(provider-adapters): extract retry logic to shared transport"
```

---

## 9. Enforcement

- `commitlint` chạy trong git hook (pre-commit via `husky`)
- CI pipeline kiểm tra commit format trước khi merge
- Merge commit từ PR được squash theo convention tự động
