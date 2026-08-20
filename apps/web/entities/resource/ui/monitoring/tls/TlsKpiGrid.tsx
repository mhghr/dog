"use client";

import { MonitorKpiCard, type KpiHealth } from "@/entities/resource/ui/components/monitor-kpi-card";

export interface TlsKpiSeries {
  probeName: string;
  location: string;
  points: Array<{ time: string; value: number }>;
}

interface TlsKpiGridProps {
  isFa: boolean;
  certificateValid: boolean | null;
  certificateExpiryDays: number | null;
  handshakeTimeMs: number | null;
  availability: number | null;
  issuer: string | null;
  notAfter: string | null;
  expirySeries: TlsKpiSeries[];
  statusSeries: TlsKpiSeries[];
  rangeLabel: string;
}

function sparkOf(series: TlsKpiSeries[]) {
  return series
    .map((s) => ({ name: s.probeName || s.location, points: s.points }))
    .filter((s) => s.points.length > 0);
}

function healthOf(availability: number | null): KpiHealth {
  if (availability == null) return "unknown";
  if (availability >= 99) return "healthy";
  if (availability >= 95) return "warning";
  return "critical";
}

function riskOf(days: number | null): { value: string; health: KpiHealth; reason?: string } {
  if (days == null) return { value: "—", health: "unknown" };
  if (days <= 7) return { value: "High", health: "critical", reason: `${days} days` };
  if (days <= 30) return { value: "Medium", health: "warning", reason: `${days} days` };
  return { value: "Low", health: "healthy" };
}

export function TlsKpiGrid({
  isFa,
  certificateValid,
  certificateExpiryDays,
  handshakeTimeMs,
  availability,
  issuer,
  notAfter,
  expirySeries,
  statusSeries,
  rangeLabel,
}: TlsKpiGridProps) {
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const risk = riskOf(certificateExpiryDays);

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-3">
      <MonitorKpiCard
        title={t("Certificate Status", "وضعیت گواهی")}
        value={certificateValid == null ? "—" : certificateValid ? t("Valid", "معتبر") : t("Invalid", "نامعتبر")}
        secondary={{
          value: certificateExpiryDays == null ? "—" : `${certificateExpiryDays} ${t("days", "روز")}`,
          label: t("Expires in", "انقضا تا"),
        }}
        spark={sparkOf(statusSeries)}
        health={certificateValid === false ? "critical" : healthOf(availability)}
        reason={certificateValid === false ? t("Certificate invalid", "گواهی نامعتبر") : undefined}
      />

      <MonitorKpiCard
        title={t("Days Remaining", "روز باقی‌مانده")}
        value={certificateExpiryDays == null ? "—" : String(certificateExpiryDays)}
        unit={t("days", "روز")}
        secondary={{
          value: notAfter
            ? new Date(notAfter).toLocaleDateString(isFa ? "fa-IR" : "en-US")
            : issuer ?? "—",
          label: t("Expiration", "انقضا"),
        }}
        spark={sparkOf(expirySeries)}
        health={healthOf(availability)}
        reason={certificateExpiryDays != null && certificateExpiryDays <= 30 ? `${certificateExpiryDays} days` : undefined}
      />

      <MonitorKpiCard
        title={t("Expiry Risk", "ریسک انقضا")}
        value={risk.value}
        secondary={{
          value: handshakeTimeMs == null ? "—" : `${Math.round(handshakeTimeMs)} ms`,
          label: t("Handshake", "دست‌داد"),
        }}
        spark={sparkOf(expirySeries)}
        health={risk.health}
        reason={risk.reason}
      />
    </div>
  );
}
