"use client";

import type { UseFormReturn } from "react-hook-form";
import { useTranslations } from "next-intl";

import { NumberField } from "@/components/monitors/form-fields";
import type { MonitorFormValues } from "@/lib/schemas";
import type { MonitorType } from "@/types/monitor";

export function MonitorThresholdFields({ type, form }: { type: MonitorType; form: UseFormReturn<MonitorFormValues> }) {
  const t = useTranslations("monitors.fields");

  if (type === "tls" || type === "domain_expiration") return null;

  return (
    <section className="border-t border-border/60 pt-5">
      <h3 className="text-sm font-medium">{t("statusThresholds")}</h3>
      <p className="mt-1 text-xs text-muted-foreground">{t("statusThresholdsHint")}</p>
      <div className="mt-4 grid gap-4 sm:grid-cols-2">
        <NumberField form={form} name="warning_duration_millis" label={t("warningDurationMillis")} min={1} max={60000} />
        <NumberField form={form} name="critical_duration_millis" label={t("criticalDurationMillis")} min={1} max={60000} />
        {type === "ping" ? (
          <>
            <NumberField form={form} name="ping_warning_packet_loss_percent" label={t("warningPacketLossPercent")} min={0} max={100} />
            <NumberField form={form} name="ping_critical_packet_loss_percent" label={t("criticalPacketLossPercent")} min={0} max={100} />
            <NumberField form={form} name="ping_warning_jitter_millis" label={t("warningJitterMillis")} min={0} max={60000} />
            <NumberField form={form} name="ping_critical_jitter_millis" label={t("criticalJitterMillis")} min={0} max={60000} />
          </>
        ) : null}
        {type === "ntp" ? (
          <>
            <NumberField form={form} name="ntp_warning_offset_millis" label={t("warningOffsetMillis")} min={1} max={600000} />
            <NumberField form={form} name="ntp_warning_round_trip_millis" label={t("warningRoundTripMillis")} min={1} max={600000} />
          </>
        ) : null}
      </div>
    </section>
  );
}
