---
status: accepted
---

# Promoção clube is a distinct Extrator channel

`promocaoClube` is a distinct channel from `promocaoCartao` (loyalty/app/clube without card → clube; store/brand card → cartão). Rejected: a generic vínculo/canal enum that collapses the two. Mutual exclusivity of the four *shapes* was relaxed by ADR 0032 (channel and quantity mechanic may compose); cartão XOR clube remains.
