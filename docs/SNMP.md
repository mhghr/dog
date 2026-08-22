# SNMP Monitoring — Public Network Devices

Dog monitors public network devices (Cisco routers/switches, firewalls,
appliances) via their built-in SNMP agent. **No Customer Monitoring Agent,
Probe Agent or Agent Gateway is involved** in this flow: the Dog SNMP
Collector connects directly to the device over UDP/161.

## Architecture

```
Resource (Network Device)
   └─ SNMP Monitor (a monitoring type, not a resource type)
        ├─ Scheduler        →  schedules poll jobs by polling interval
        ├─ NATS JetStream   →  ProbeJob → snmp.tasks (test/discovery)
        ├─ Worker           →  runs the SNMP Collector
        │     ├─ Poll       →  GET / GETBULK on required OIDs
        │     └─ Test/Disc  →  real SNMP request / MIB walk
        ├─ Metric Normalizer → device.* + if_<idx>_* (stable names, controlled labels)
        ├─ Metric Processor  → counter deltas → rates (wrap-safe)
        ├─ VictoriaMetrics  →  time series storage
        └─ Health Engine    →  snmp.* health rules → alerts
```

- `packages/shared/snmp/` — OID/MIB registry (core + Cisco providers), gosnmp
  client, discovery, polling, counter/rate math, failure taxonomy.
- `packages/shared/probe/snmp.go` — `SNMPExecutor` implementing the same
  `probe.Executor` interface as ping/http/tcp/dns/tls.
- On-demand test/discovery run on the **SNMP worker** in NATS mode
  (`TELEMETRY_PIPELINE_MODE=nats`). When no bus is configured the API falls
  back to the inline runner that executes the identical collector code.
- Secrets are encrypted at rest (`security.EncryptSecret`, AES-256-GCM) and
  only a `credential_reference` is exposed; they never appear in logs, API
  responses, telemetry or error messages.

## Local SNMP test environment

The integration tests (`packages/shared/snmp/snmpfake`) run a real UDP SNMPv2c
agent. For manual end-to-end testing against a "real" device, run net-snmp on
your machine:

### Option A — net-snmp via Docker (recommended)

```bash
# Linux/Ubuntu host
docker run --rm -d --name snmpd -p 161:161/udp \
  -e SNMPD_COMMUNITY=dogtest \
  polinux/snmpd

# or directly:
sudo apt-get update && sudo apt-get install -y snmpd snmp
sudo sed -i 's/^rocommunity.*/rocommunity dogtest/' /etc/snmp/snmpd.conf
sudo systemctl restart snmpd
```

Verify locally:

```bash
snmpget -v2c -c dogtest 127.0.0.1:161 SNMPv2-MIB::sysName.0
snmpwalk -v2c -c dogtest 127.0.0.1:161 IF-MIB::ifTable
```

> On Windows you can run the same container via Docker Desktop (the host port
> mapping exposes UDP/161 to the API/worker).

### Manual test from the Dog backend

With the dev stack running, create a `Network Device` resource whose address
points at the simulator (e.g. `127.0.0.1` or the container IP), then in the
SNMP wizard:

1. **Configuration** — SNMPv2c, community `dogtest`, port `161`.
2. **Test Connection** — the collector sends a real SNMP GET and reports the
   staged result (DNS, reachability, UDP/161, SNMP response, authentication).
3. **Discovery** — walks IF-MIB/HOST-RESOURCES and lists interfaces/sensors.
4. Save, then watch the Monitoring tab for CPU/memory/uptime KPIs and the
   interface table.

### Simulating failures (integration tests)

The fake agent (`snmpfake`) covers: success, timeout, wrong community
(SNMPv2c auth), device unreachable, partial collection (system columns
missing), interface down, and counter wrap/rate. Run:

```bash
go test ./packages/shared/snmp/... ./packages/shared/probe/... -v
```

### Cisco-specific mapping

Cisco MIBs (CISCO-PROCESS-MIB CPU, CISCO-MEMORY-POOL-MIB, CISCO-ENVMON-MIB
temperature/fan/power) are wired through the `ciscoProvider` in
`packages/shared/snmp/oids.go`. They activate automatically when a device's
`sysObjectID` is under `1.3.6.1.4.1.9`. A Cisco IOS/IOS-XE virtual lab or
simulator can be added later without touching the core registry.

## Security

- **SNMPv3 is recommended** (authPriv). SNMPv2c communities must be read-only.
- Secrets are encrypted at rest and never returned to the frontend; the UI
  shows only "Configured".
- The SNMP target is bound to the resource's own address
  (`validateSnmpTarget`) — the collector cannot be pointed at arbitrary
  internal IPs (SSRF-like abuse is blocked).
- Allow UDP/161 **only** from the Dog collector source IPs
  (`SNMP_SOURCE_IPS` env; shown in the wizard's firewall guide).

## Collector source IPs

Configure in `.env`:

```
SNMP_SOURCE_IPS=203.0.113.10,203.0.113.11
```

The wizard and the firewall guide read these from the backend
(`GET /api/snmp/source-ips`) — they are never hard-coded in the frontend.
