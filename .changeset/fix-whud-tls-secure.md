---
"eratemanager": patch
---

Use proper certificate pool instead of InsecureSkipVerify for WHUD

Embed the GoDaddy G2 intermediate certificate for servers that don't send
complete certificate chains. This maintains proper TLS verification while
working around misconfigured servers like whud.org.
