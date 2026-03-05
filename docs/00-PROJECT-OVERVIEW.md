# Project Overview — ProxyVPS Platform

**Document ID**: PVP-DOC-000  
**Version**: 1.0.0  
**Status**: Approved  
**Last Updated**: 2025-01-01  

---

## 1. Executive Summary

ProxyVPS Platform là hệ thống thương mại điện tử B2B/B2C chuyên về bán lại dịch vụ Proxy và VPS. Hệ thống tích hợp API từ nhiều nhà cung cấp proxy bên thứ ba đồng thời kết nối trực tiếp với hạ tầng Proxmox tự vận hành để cung cấp VPS. Nền tảng hỗ trợ mô hình reseller đa cấp, cho phép đối tác xây dựng kinh doanh riêng dựa trên hạ tầng của platform.

---

## 2. Problem Statement

### 2.1 Bài toán hiện tại
- Quản lý nhiều tài khoản proxy provider thủ công, tốn thời gian
- Không có công cụ thống nhất để theo dõi trạng thái proxy/VPS theo thời gian thực
- Không thể mở rộng sang mô hình reseller mà không có hệ thống tự động
- Proxmox đang vận hành thủ công, không có API layer để tự động hóa provisioning

### 2.2 Hậu quả nếu không giải quyết
- Giới hạn tăng trưởng về số lượng khách hàng
- Chi phí vận hành cao do xử lý thủ công
- Rủi ro sai sót trong billing và credential delivery

---

## 3. Project Goals

### 3.1 Primary Goals
| ID | Goal | KPI |
|---|---|---|
| G-01 | Tự động hóa 100% quy trình từ order → delivery | Zero manual intervention for standard orders |
| G-02 | Hỗ trợ đồng thời 1,000 users active | Response time < 200ms p95 |
| G-03 | Xử lý 1,000 proxy orders/ngày | Order fulfillment < 30s |
| G-04 | Quản lý tối đa 10 VPS cùng lúc (scale up sau) | VM ready < 120s |
| G-05 | Hệ thống reseller tier đầy đủ | Reseller onboard < 5 phút |

### 3.2 Secondary Goals
- Full audit log mọi thao tác (compliance-ready)
- Real-time dashboard cho admin, reseller, và end-user
- White-label capability cho resellers (Phase 2)
- API-first design để reseller tự build frontend

---

## 4. Scope

### 4.1 In Scope
- User registration, authentication, authorization (RBAC)
- Proxy ordering: mua, gia hạn, hủy, xem credentials
- VPS ordering: tạo, quản lý, tạm dừng, xóa VM trên Proxmox
- Wallet system (prepaid) với nạp tiền qua payment gateway
- Billing tự động: flat-fee cho proxy, hourly metering cho VPS
- Reseller management: sub-accounts, custom pricing, reseller wallet
- Real-time event streaming qua WebSocket
- Full audit log với search và filter
- Admin dashboard: quản lý toàn hệ thống
- Email notifications: order confirmation, low balance, expiry alert
- REST API đầy đủ cho tất cả tính năng

### 4.2 Out of Scope (Phase 1)
- Mobile app (iOS/Android)
- White-label domain cho reseller
- Affiliate/commission tracking
- Cryptocurrency payment
- Multi-currency (chỉ VND + USD)
- Advanced DDoS protection for VPS
- Managed services (cPanel, databases...)

---

## 5. Stakeholders

| Role | Name/Team | Responsibility |
|---|---|---|
| Product Owner | Platform Owner | Requirements, acceptance criteria |
| Tech Lead | Backend Team | Architecture decisions |
| DevOps | Infrastructure Team | Proxmox, deployment |
| QA | QA Team | Test cases, UAT |
| End Users | Customers | Final consumers of proxy/VPS |
| Resellers | Partner Accounts | Resell to their own customers |

---

## 6. Technical Constraints

| Constraint | Description |
|---|---|
| Language | Go 1.22+ cho tất cả backend services |
| Proxmox | Phải tương thích Proxmox VE 8.x REST API |
| Database | PostgreSQL (không dùng NoSQL cho transactional data) |
| Deployment | Docker Compose (dev), có thể migrate sang K8s sau |
| Uptime SLA | 99.5% cho production |
| Latency | API response < 200ms p95 (excluding async operations) |
| Security | JWT auth, TLS everywhere, no plaintext credentials |

---

## 7. Assumptions

1. Proxmox nodes có đủ tài nguyên cho 10 VPS đồng thời với headroom 30%
2. Proxy providers đã ký hợp đồng và cấp API credentials
3. Payment gateway (local + international) đã được approved
4. Network infrastructure giữa platform server và Proxmox nodes ổn định < 10ms latency
5. Team có Go development experience

---

## 8. Risks

| ID | Risk | Probability | Impact | Mitigation |
|---|---|---|---|---|
| R-01 | Provider API thay đổi không báo trước | Medium | High | Adapter pattern, monitoring, fallback providers |
| R-02 | Proxmox node downtime | Low | High | Multi-node failover, health checks |
| R-03 | Payment gateway chargeback | Medium | High | Wallet prepaid model, KYC cho reseller |
| R-04 | Over-provision tài nguyên | Low | Medium | Resource reservation before commit |
| R-05 | Log service bottleneck | Low | Medium | Async publish, separate DB partition |

---

## 9. Success Criteria

Dự án được coi là thành công khi:
- [ ] 100% orders được fulfill tự động không cần can thiệp thủ công
- [ ] System uptime >= 99.5% trong 30 ngày đầu production
- [ ] Audit log đầy đủ mọi action với traceability từ user → service → provider
- [ ] Reseller có thể onboard và setup custom pricing mà không cần admin can thiệp
- [ ] Zero billing errors trong 30 ngày đầu production

---

## 10. Milestones

| Phase | Deliverable | Target |
|---|---|---|
| Phase 1 | Core: Auth + Wallet + Admin | Week 6 |
| Phase 2 | Proxy Service + Log Service | Week 10 |
| Phase 3 | VPS Service + Proxmox Adapter | Week 16 |
| Phase 4 | Reseller Tier + API Keys | Week 20 |
| Phase 5 | Production Hardening + Monitoring | Week 22 |
