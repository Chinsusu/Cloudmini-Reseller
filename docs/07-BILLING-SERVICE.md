# Billing Service — Service Design

**Document ID**: PVP-DOC-007  
**Version**: 1.0.0  
**Service**: billing-service  
**Port**: 8085  

---

## 1. Responsibilities

- Quản lý wallet (prepaid balance) cho từng user và reseller
- Xử lý deposits qua payment gateway
- Charge cho proxy orders và VPS metering
- Hold/release funds cho async operations (VPS provisioning)
- Pricing engine: apply reseller markup
- Invoice generation (monthly)
- Low balance detection và alerts

---

## 2. API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/v1/billing/wallet` | Số dư ví |
| GET | `/api/v1/billing/transactions` | Lịch sử giao dịch |
| POST | `/api/v1/billing/deposit` | Tạo payment link nạp tiền |
| GET | `/api/v1/billing/invoices` | Danh sách hóa đơn |
| GET | `/api/v1/billing/invoices/:id` | Chi tiết hóa đơn |
| POST | `/api/v1/billing/webhook/stripe` | Stripe webhook |
| POST | `/api/v1/billing/webhook/vnpay` | VNPay callback |
| GET | `/api/v1/admin/billing/wallets` | Admin: tất cả wallets |
| POST | `/api/v1/admin/billing/adjustment` | Manual balance adjustment |

---

## 3. Wallet Operations

### Deduct (Proxy Order)
```
BEGIN TRANSACTION
    SELECT balance FROM wallets WHERE user_id = ? FOR UPDATE
    IF balance < amount → ROLLBACK → return ErrInsufficientFunds
    UPDATE wallets SET balance = balance - amount
    INSERT INTO transactions (type='order_charge', ...)
COMMIT
Publish: billing.charged {user_id, amount, order_id}
```

### Hold (VPS Provisioning)
```
BEGIN TRANSACTION
    SELECT balance, hold_amount FROM wallets WHERE user_id = ? FOR UPDATE
    available = balance - hold_amount
    IF available < estimated_charge → error
    UPDATE wallets SET hold_amount = hold_amount + amount
    INSERT transactions (type='hold', ...)
COMMIT
```

### Confirm Hold → Actual Charge
```
BEGIN TRANSACTION
    UPDATE wallets SET 
        balance = balance - amount,
        hold_amount = hold_amount - amount
    INSERT transactions (type='order_charge', ...)
COMMIT
```

---

## 4. Pricing Engine

```go
func CalculatePrice(product Product, userID, resellerID uuid.UUID) decimal.Decimal {
    basePrice := product.BasePrice
    
    // Check reseller pricing override first
    if resellerID != uuid.Nil {
        override := GetResellerPricing(resellerID, product.ID)
        if override != nil {
            return override.SellPrice
        }
    }
    
    // Global platform pricing (markup on base)
    rule := GetPricingRule(product.Type, product.ID)
    if rule.MarkupType == "percentage" {
        return basePrice.Mul(decimal.NewFromFloat(1 + rule.MarkupValue))
    }
    return basePrice.Add(rule.MarkupValue)
}
```

---

## 5. Events Consumed & Published

**Consumed:**
- `order.proxy.created` → deduct wallet
- `vps.usage.report` → hourly VPS charge
- `vm.provision.failed` → release hold
- `billing.wallet.empty` (self-trigger) → suspend VPS

**Published:**
- `billing.charged` → log-service
- `billing.deposit.completed` → notification
- `billing.wallet.low` → notification (threshold alert)
- `billing.wallet.empty` → vps-service (auto-suspend)
