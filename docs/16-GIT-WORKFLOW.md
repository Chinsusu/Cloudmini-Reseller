# Git Workflow — ProxyVPS Platform

**Document ID**: PVP-DOC-016  
**Version**: 1.0.0  

---

## 1. Branch Strategy

Dự án dùng **GitHub Flow** (simplified GitFlow) phù hợp với microservices và CI/CD liên tục.

```
main
 │
 ├── (always deployable, protected)
 │
 ├── feature/PVP-142-provider-failover-routing
 ├── fix/PVP-156-node-selection-panic
 ├── refactor/billing-wallet-concurrent-fix
 └── release/v1.2.0  (khi chuẩn bị release lớn)
```

---

## 2. Branch Naming

```
<type>/<ticket-id>-<short-description>

# Examples
feature/PVP-142-provider-failover-routing
fix/PVP-156-node-selection-panic
refactor/PVP-180-extract-retry-transport
docs/PVP-191-update-proxmox-adapter-doc
chore/PVP-200-upgrade-chi-router
release/v1.2.0
hotfix/PVP-201-wallet-negative-balance
```

### Rules
- **Lowercase** with hyphens
- **Include ticket ID** khi có (PVP-XXX)
- **Descriptive but concise** (max 50 chars after type/)
- **Không** commit thẳng lên `main` (protected)

---

## 3. Daily Workflow

### Bắt đầu task mới

```bash
# 1. Update main
git checkout main
git pull origin main

# 2. Tạo branch mới
git checkout -b feature/PVP-142-provider-failover-routing

# 3. Code, commit thường xuyên (small commits)
git add -p                          # stage theo patch, không git add .
git commit -m "feat(proxy-service): add provider health check before routing"

# 4. Push lên remote thường xuyên
git push origin feature/PVP-142-provider-failover-routing
```

### Cập nhật branch từ main (rebase, không merge)

```bash
git fetch origin
git rebase origin/main

# Nếu có conflict:
# 1. Resolve conflict trong files
# 2. git add <resolved-files>
# 3. git rebase --continue

# KHÔNG dùng git merge main vào feature branch
```

### Tạo Pull Request

```bash
# Khi feature hoàn thành:
git push origin feature/PVP-142-provider-failover-routing
# → Mở PR trên GitHub
```

---

## 4. Pull Request Process

### PR Checklist (tác giả)
- [ ] Title theo format: `feat(proxy-service): add provider failover routing`
- [ ] Description đầy đủ: What, Why, How to test
- [ ] Self-review trước khi request review
- [ ] Tests pass locally (`make test`)
- [ ] Lint pass (`make lint`)
- [ ] No TODO comments left (hoặc có ticket cho TODO)
- [ ] Documentation updated nếu cần
- [ ] Không có debug code (`fmt.Println`, breakpoints)

### PR Template

```markdown
## Summary
<!-- What does this PR do? -->

## Motivation
<!-- Why is this needed? Link to ticket -->
Closes #PVP-XXX

## Changes
<!-- List of main changes -->
- 

## How to Test
<!-- Steps to verify this works -->
1. 
2. 

## Screenshots (nếu có UI changes)

## Checklist
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] No breaking changes (or BREAKING CHANGE noted in commit)
```

### Review Process
- Minimum **1 approval** để merge (2 approvals cho changes lớn)
- Reviewer có thể **Approve**, **Request Changes**, hoặc **Comment**
- Author **phải** resolve tất cả Request Changes trước khi merge
- Discussions có thể mark resolved bởi reviewer (không phải tác giả)
- **Squash and Merge** là default merge strategy

---

## 5. Protected Branch Rules (main)

Cấu hình trong GitHub Branch Protection:

```yaml
main:
  required_status_checks:
    - ci/test
    - ci/lint
    - ci/build
  required_reviews: 1
  dismiss_stale_reviews: true
  require_signed_commits: false
  enforce_admins: false  # admin có thể bypass khi emergency
  restrict_pushes: true  # chỉ merge qua PR
```

---

## 6. Release Process

### Minor/Patch Release

```bash
# 1. Tạo release branch từ main
git checkout main && git pull
git checkout -b release/v1.2.0

# 2. Update version
# - Update version trong các go.mod hoặc version file
# - Update CHANGELOG.md

git commit -m "chore(release): bump version to v1.2.0"

# 3. PR từ release/v1.2.0 → main
# 4. Sau merge, tag main
git checkout main && git pull
git tag -a v1.2.0 -m "Release v1.2.0"
git push origin v1.2.0
```

### Hotfix

```bash
# Hotfix từ main (current production)
git checkout main
git checkout -b hotfix/PVP-201-wallet-negative-balance

# Fix, commit, test
git commit -m "fix(billing-service)!: prevent negative balance on concurrent deduction

Hotfix for critical production issue where concurrent proxy orders
could result in negative wallet balance due to race condition.

Fixes #PVP-201"

# PR → main (priority review)
# Sau merge: tag patch version
git tag -a v1.1.1 -m "Hotfix v1.1.1: wallet negative balance"
```

---

## 7. Tagging Convention

```
v{MAJOR}.{MINOR}.{PATCH}

v1.0.0  ← initial production release
v1.0.1  ← patch (bug fix)
v1.1.0  ← minor (new feature, backward compatible)
v2.0.0  ← major (breaking change)
```

---

## 8. GitHub Actions CI Pipeline

```yaml
# .github/workflows/ci.yml

on:
  pull_request:
    branches: [main]
  push:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: golangci/golangci-lint-action@v4
        with:
          version: v1.57

  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
      redis:
        image: redis:7
      nats:
        image: nats:2.10
    steps:
      - run: make test-all

  build:
    runs-on: ubuntu-latest
    steps:
      - run: make build-all

  deploy-staging:
    needs: [lint, test, build]
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - run: make deploy-staging
```

---

## 9. Useful Git Aliases

Thêm vào `~/.gitconfig`:

```ini
[alias]
  lg = log --oneline --decorate --graph --all
  st = status -sb
  co = checkout
  br = branch -vv
  uncommit = reset --soft HEAD~1
  unstage = restore --staged
  aliases = config --get-regexp alias
```
