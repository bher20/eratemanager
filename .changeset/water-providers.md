---
"eratemanager": minor
---

Add water provider support with WHUD (White House Utility District) as the first water utility

### New Features
- Support for water utility providers alongside electric providers
- New `/water/providers` endpoint to list water utilities
- New `/water/rates/{provider}` endpoint to get water rates
- WHUD parser extracts water and sewer rates from their website

### Water Rate Data
- Meter base charges by size (5/8", 1", 1.5", etc.)
- Per-gallon water usage rates
- Sewer base and usage rates
- Bill calculation helpers

### API Changes
- Providers now have a `type` field: `"electric"` or `"water"`
- New `/water/*` endpoints for water-specific data
