# Self-hosted memory relay

Reserved service boundary; no runnable server or deployment is implemented yet. Follow [003 Memory sync](../../specs/003-memory-sync.md), [005 Privacy](../../specs/005-privacy.md), and Go DDD: domain → application ← infrastructure, thin interfaces.

The relay stores authenticated ciphertext and routing revisions. It never processes speech or reads plaintext memory. Add its Go module, migrations, concrete wire fixtures and service runbook together when implementing task SYNC-01.
