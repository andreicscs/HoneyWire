# HoneyWire v1.1.0

## What's New

**Asynchronous Alert Processing** - The Hub now queues external notifications (webhooks and SIEM forwarding) instead of spawning concurrent connections. This prevents socket exhaustion and API rate limiting during high-volume attacks or network scans.

**SIEM Integration** - Optional real-time event forwarding to SIEM solutions (Splunk, ELK, Graylog, etc.) in RFC3164 syslog format via TCP or UDP.

**Graceful Shutdown** - Alert queues are properly drained before container restart, preventing lost events.

**SIEM settings** - SIEM events forwarding now configured directly from the Settings dashboard.

## Configuration

**SIEM Settings** (Dashboard → Settings → SIEM Forwarding):
- Server Address: `host:port` (e.g., `elk.example.com:514`)  
- Protocol: TCP or UDP
- Leave blank to disable

**Push Notifications** (unchanged):
- Webhooks: Discord, Slack, Ntfy, Gotify
- Rate-limited to 500ms/send (prevents anti-spam throttling)

## Architecture

- **Webhook Queue**: 1,000 events, 500ms delays between sends
- **SIEM Queue**: 5,000 events, sent sequentially with network timeouts
- **On Shutdown**: Queues drained gracefully over 10 seconds max

## Migration

No database schema changes. Simply deploy v1.1.0 - existing sensors and configurations work unchanged.

## Performance

- High-volume alerts (5,000+ simultaneously): Previously crashed, now handled gracefully
- SIEM/webhook latency: No longer impacts API response time
- Network outages: Queue buffers events until service recovers

---

*Changelog: v1.0.0 → v1.1.0 | 2026-04-20*
