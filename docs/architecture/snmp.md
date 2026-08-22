# Dog — SNMP Architecture

## مفهوم: SNMP به‌عنوان Collection Method

SNMP یک **Collection Method** است، نه یک نوع مانیتور جدا. کاربر «SNMP
Monitor» انتخاب نمی‌کند؛ بلکه:

```
Network Device (Resource)
   → Metrics (CPU, Memory, Interface Traffic, Temperature)
   → Collection Method: SNMP
```

معماری فعلی هنوز یک `monitor_types` با `executor_key = 'snmp'` دارد
(`monitors.type`/`executor_key` برای آن seed شده). هدف، تبدیل تدریجی به
Collection Method است که روی Resource Type ها اعمال شود.

## اجزای پیاده‌سازی (`packages/shared/snmp/`)

| فایل | نقش |
|---|---|
| `client.go` | کلاینت SNMP (GET / GETBULK / WALK) |
| `oids.go` | OID Registry (`Registry`/`DefaultRegistry`) برای vendorها |
| `discovery.go` | Discovery: identity (vendor/model)، interfaces، CPU، memory، sensors |
| `poll.go` | Polling دوره‌ای متریک‌ها |
| `observe.go` | Observable تعریف‌شده برای metricهای device |
| `tasks.go` | اجرای taskهای on-demand (test/discovery) — `ExecuteTask` |
| `trap/receiver.go` | Trap Receiver (UDP 162) |
| `normalize.go` | نرمال‌سازی مقادیر |

## جداول PostgreSQL

| جدول | migration | نقش |
|---|---|---|
| `snmp_credentials` | 000023 | credentialهای SNMP (v1/2c/3) — **encrypted** |
| `snmp_devices` | 000023 | binding دستگاه به resource + credential |
| `snmp_discovery` | 000034 | cache نتیجه discovery (interface/sensors) |
| `snmp_interfaces` | 000034 | تنظیمات و آخرین وضعیت interfaceها |
| `snmp_events` | 000034 | رویدادها (trap یا تشخیص تغییر) |
| `snmp_tasks` | 000035 | taskهای test/discovery |

## Discovery

- **WALK** برای discovery (شناسایی ساختار).
- **GET / GETBULK** برای polling.
- نتیجه: `SNMPDiscoveryResult` → ذخیره در `snmp_discovery` + پیشنهاد
  interfaceها/sensors به‌عنوان متریک.
- Partial failures قابل مدیریت (یک interface خراب، کل discovery را نمی‌شکند).

## Trap (Event Source)

```
Device → UDP/162 → TrapReceiver → normalize → SNMPEvent → ذخیره + alerting
```

قانون‌ها:
- Trap یک **Event** است (مثل «Interface Down»).
- Trap جایگزین Time-Series نیست؛ Interface Traffic/CPU **Metric** هستند که از
  polling می‌آیند.
- `ListSnmpMonitorsByTarget` در `monitor_repository.go` به trap receiver اجازه
  می‌دهد event را به resource درست بچسباند بدون دسترسی cross-tenant.

## Credentialها و امنیت

- `snmp_credentials.config` به‌صورت **encrypted** ذخیره می‌شود
  (کلید encryption از env می‌آید).
- Credentialها هرگز log نمی‌شوند.

## UX پیشنهادی (آینده)

1. توضیح: «SNMP به Dog اجازه می‌دهد بدون نصب agent روی device، متریک جمع کند».
2. Connection Method: v2c / v3.
3. Test Connection → Discovery → Select Metrics → Enable Collection.
4. OID/MIB فقط در Advanced Mode — کاربر معمولی مجبور به وارد کردن OID نیست.
