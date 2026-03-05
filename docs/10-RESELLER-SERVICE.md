# Reseller Service — Service Design

**Document ID**: PVP-DOC-010  
**Version**: 1.0.0  
**Service**: reseller-service  
**Port**: 8084  

---

## 1. Responsibilities

- Quản lý reseller accounts (onboarding, approval)
- Custom pricing per reseller per product
- Reseller wallet management (separate from user wallet)
- Sub-account management (users under reseller)
- Reseller dashboard data aggregation
- API key management cho reseller integration
- Usage reports và commission tracking

---

## 2. Account Hierarchy

```
Platform Admin
    │
    ├── Reseller A  (has own wallet)
    │       ├── User 1  ← pays through Reseller A's wallet
    │       ├── User 2
    │       └── User 3
    │
    └── Reseller B
            ├── User 4
            └── User 5
```

### Billing Flow
```
User under Reseller buys product
    │
    ▼
billing-service deducts from Reseller's wallet
    (at reseller's cost price, not user's sell price)
    │
    ▼
Reseller earns: (sell_price - cost_price) margin
    │
    ▼
Reseller must top up their wallet independently
```

---

## 3. API Endpoints

### Admin (manage resellers)
| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/v1/admin/resellers` | Danh sách resellers |
| POST | `/api/v1/admin/resellers` | Tạo reseller account |
| PUT | `/api/v1/admin/resellers/:id/approve` | Approve reseller |
| PUT | `/api/v1/admin/resellers/:id/suspend` | Suspend reseller |
| PUT | `/api/v1/admin/resellers/:id/pricing` | Set cost price |

### Reseller (self-service)
| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/v1/reseller/dashboard` | Stats: users, revenue, orders |
| GET | `/api/v1/reseller/users` | Sub-accounts |
| POST | `/api/v1/reseller/users` | Create sub-account |
| PUT | `/api/v1/reseller/users/:id/credit` | Set credit limit |
| GET | `/api/v1/reseller/pricing` | Xem pricing config |
| PUT | `/api/v1/reseller/pricing/:product_id` | Set sell price |
| GET | `/api/v1/reseller/reports/orders` | Order report |
| GET | `/api/v1/reseller/reports/revenue` | Revenue report |
| GET | `/api/v1/reseller/wallet` | Reseller wallet |
| GET | `/api/v1/reseller/api-keys` | API keys |
| POST | `/api/v1/reseller/api-keys` | Tạo API key |

---

## 4. Pricing Constraint

```
Platform sets:  cost_price (reseller pays)
Reseller sets:  sell_price (user pays)

Validation:
  sell_price >= cost_price * 1.0   (không được bán lỗ dưới cost)
  sell_price <= cost_price * 100   (sanity check)
  
  Admin có thể set floor_price per product
  → sell_price >= floor_price
```

---

## 5. Reseller Approval Flow

```
1. User registers → requests reseller upgrade
2. Admin reviews application
3. Admin approves: 
   - Create resellers.accounts record
   - Create dedicated wallet
   - Update user role → reseller
   - Set initial cost pricing (copy from global defaults)
4. Notify reseller → onboarding email with API docs
```

---

## 6. Events Published

| Event | Payload |
|---|---|
| `reseller.created` | `{reseller_id, user_id}` |
| `reseller.approved` | `{reseller_id}` |
| `reseller.suspended` | `{reseller_id, reason}` |
| `reseller.pricing.updated` | `{reseller_id, product_id, new_price}` |
