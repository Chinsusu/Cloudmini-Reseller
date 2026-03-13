# Trang: `/reseller/pricing` — Quản lý Giá Bán

> Role: **reseller**

---

## Mục đích

Reseller đặt giá bán riêng cho từng sản phẩm proxy, không được thấp hơn floor price (giá sàn do admin đặt).

---

## Layout

```
┌──────────────────────────────────────────────────────────────────┐
│  Header: "Pricing Rules"                                         │
├──────────────────────────────────────────────────────────────────┤
│  Table: Product | Type | Location | Floor Price | Sell Price | % │
└──────────────────────────────────────────────────────────────────┘
```

---

## Table Columns

| Column | Mô tả |
|--------|-------|
| Product | Tên sản phẩm |
| Type | Badge: residential / datacenter / mobile |
| Location | Country code hoặc "Global" |
| Floor Price | Giá sàn tối thiểu — màu `var(--error)`, read-only |
| Sell Price | Giá bán ra — click để edit inline |
| Markup % | `(sell - floor) / floor × 100%` — auto-computed, xanh nếu ≥ 0 |
| Actions | Edit / Save / Cancel |

---

## Inline Edit Flow

1. Click nút "Edit" (hoặc click trực tiếp vào ô Sell Price)
2. Input xuất hiện in-place với giá trị hiện tại
3. Validation realtime: `sell_price >= floor_price`
   - Nếu vi phạm: border đỏ + tooltip "Cannot be less than floor price"
4. Click "Save" → `PUT /api/v1/reseller/pricing/{product_id}   { sell_price }`
5. Click "Cancel" → khôi phục giá trị cũ
6. Toast success/error

---

## Logic tính Markup

```
markup = ((sell_price - floor_price) / floor_price) × 100
```

- Hiển thị dạng `+X.X%`
- Màu xanh nếu > 0, màu đỏ nếu < 0 (không nên xảy ra do validation)

---

## API Calls

```
GET /api/v1/reseller/pricing
PUT /api/v1/reseller/pricing/{product_id}   { sell_price }
```

---

## File

`frontend/src/app/reseller/pricing/page.tsx`
