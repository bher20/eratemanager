---
"eratemanager": patch
---

Fix TLS certificate verification for WHUD water provider

The WHUD server has a misconfigured SSL certificate chain (missing intermediate certificates).
Added an insecure HTTP client as a workaround for servers with broken certificate chains.
