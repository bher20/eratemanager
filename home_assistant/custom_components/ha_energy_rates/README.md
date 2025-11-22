# HA Energy Rates (Go backend)

This custom integration talks to the eRateManager Go backend and exposes:
- Energy rate
- Fuel rate
- Total rate (energy + fuel)
- Fixed (monthly) charge

It supports multiple providers:
- CEMC (`/rates/cemc/residential`)
- NES (`/rates/nes/residential`)
- Demo (`/rates/demo/residential`)
