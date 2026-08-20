"use client";

import type { UseFormReturn } from "react-hook-form";
import { useTranslations } from "next-intl";
import { Activity, Radio, Waves, Gauge } from "lucide-react";

import { NumberField, SelectField, SwitchField, TextField } from "@/features/monitor-management/ui/form-fields";
import type { MonitorFormValues } from "@/entities/monitor/model/form-values";
import { Label } from "@/shared/ui/label";
import { cn } from "@/shared/utils/cn";

type ConfigFieldsProps = { form: UseFormReturn<MonitorFormValues> };

function InputGroup({
  label,
  icon: Icon,
  children,
}: {
  label: string;
  icon: React.ElementType;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-xl border border-border/60 bg-card/30 p-4 sm:p-5">
      <div className="mb-4 flex items-center gap-2.5">
        <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
          <Icon className="size-4" aria-hidden />
        </span>
        <span className="text-sm font-semibold">{label}</span>
      </div>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {children}
      </div>
    </div>
  );
}

const HEALTH_LEVELS = {
  healthy: { label: "healthy", color: "border-emerald-500/40 bg-emerald-500/[0.06]", dot: "bg-emerald-500", ring: "ring-emerald-500/20" },
  degraded: { label: "degraded", color: "border-amber-500/40 bg-amber-500/[0.06]", dot: "bg-amber-500", ring: "ring-amber-500/20" },
  down: { label: "down", color: "border-red-500/40 bg-red-500/[0.06]", dot: "bg-red-500", ring: "ring-red-500/20" },
} as const;

function ThresholdPair({
  form,
  warningName,
  criticalName,
  warningLabel,
  criticalLabel,
}: {
  form: UseFormReturn<MonitorFormValues>;
  warningName: keyof MonitorFormValues;
  criticalName: keyof MonitorFormValues;
  warningLabel: string;
  criticalLabel: string;
}) {
  return (
    <>
      <ThresholdSlot form={form} name={warningName} label={warningLabel} level="degraded" />
      <ThresholdSlot form={form} name={criticalName} label={criticalLabel} level="down" />
    </>
  );
}

function ThresholdSlot({
  form,
  name,
  label,
  level,
}: {
  form: UseFormReturn<MonitorFormValues>;
  name: keyof MonitorFormValues;
  label: string;
  level: keyof typeof HEALTH_LEVELS;
}) {
  const colors = HEALTH_LEVELS[level];

  return (
    <div
      className={cn(
        "flex flex-col gap-1.5 rounded-xl border p-3.5 transition-[border-color,background-color,box-shadow]",
        "focus-within:ring-[3px]",
        colors.color,
        colors.ring,
      )}
    >
      <Label htmlFor={name} className="flex items-center gap-2 text-xs font-medium text-foreground/70">
        <span className={cn("size-2 shrink-0 rounded-full", colors.dot)} />
        {label}
      </Label>
      <NumberField form={form} name={name} label="" bare />
    </div>
  );
}

export function PingConfigFields({ form }: ConfigFieldsProps) {
  const t = useTranslations("monitors.fields");

  return (
    <div className="flex flex-col gap-5">
      <InputGroup label={t("packetSettings")} icon={Activity}>
        <NumberField form={form} name="ping_packet_count" label={t("packetCount")} min={1} max={20} suffix={t("packets")} fullWidth />
        <NumberField form={form} name="ping_packet_interval_millis" label={t("packetInterval")} min={10} max={10000} suffix="ms" fullWidth />
      </InputGroup>

      <InputGroup label={t("latencyThresholds")} icon={Gauge}>
        <ThresholdPair
          form={form}
          warningName="ping_warning_latency_millis"
          criticalName="ping_critical_latency_millis"
          warningLabel={t("warningLatencyMillis")}
          criticalLabel={t("criticalLatencyMillis")}
        />
      </InputGroup>

      <InputGroup label={t("packetLossThresholds")} icon={Radio}>
        <ThresholdPair
          form={form}
          warningName="ping_warning_packet_loss_percent"
          criticalName="ping_critical_packet_loss_percent"
          warningLabel={t("warningPacketLossPercent")}
          criticalLabel={t("criticalPacketLossPercent")}
        />
      </InputGroup>

      <InputGroup label={t("jitterThresholds")} icon={Waves}>
        <ThresholdPair
          form={form}
          warningName="ping_warning_jitter_millis"
          criticalName="ping_critical_jitter_millis"
          warningLabel={t("warningJitterMillis")}
          criticalLabel={t("criticalJitterMillis")}
        />
      </InputGroup>
    </div>
  );
}

export function HTTPConfigFields({ form }: ConfigFieldsProps) {
  const t = useTranslations("monitors.fields");
  return (
    <div className="flex flex-col gap-5">
      <InputGroup label={t("httpSettings")} icon={Gauge}>
        <SelectField form={form} name="http_method" label={t("method")} options={["GET","POST","PUT","PATCH","DELETE","HEAD","OPTIONS"].map((v) => ({ value: v, label: v }))} />
        <TextField form={form} name="http_expected_status_codes" label={t("expectedStatusCodes")} hint={t("statusCodesHint")} dir="ltr" placeholder="200, 204" />
        <NumberField form={form} name="http_max_redirects" label={t("maxRedirects")} min={0} max={20} fullWidth />
        <TextField form={form} name="http_body_contains" label={t("bodyContains")} hint={t("bodyContainsHint")} placeholder="healthy, success" />
        <TextField form={form} name="http_body" label={t("requestBody")} hint={t("requestBodyHint")} placeholder='{"ping":1}' />
      </InputGroup>

      <InputGroup label={t("httpBehavior")} icon={Activity}>
        <SwitchField form={form} name="http_follow_redirects" label={t("followRedirects")} />
        <SwitchField form={form} name="http_verify_tls" label={t("verifyTls")} />
      </InputGroup>
    </div>
  );
}

export function TCPConfigFields({ form }: ConfigFieldsProps) {
  const t = useTranslations("monitors.fields");
  return (
    <div className="flex flex-col gap-5">
      <InputGroup label="TCP" icon={Gauge}>
        <NumberField form={form} name="tcp_port" label={t("port")} min={1} max={65535} fullWidth />
      </InputGroup>
    </div>
  );
}

export function DNSConfigFields({ form }: ConfigFieldsProps) {
  const t = useTranslations("monitors.fields");
  return (
    <div className="flex flex-col gap-5">
      <InputGroup label="DNS" icon={Gauge}>
        <TextField form={form} name="dns_server" label={t("dnsServer")} dir="ltr" placeholder="1.1.1.1:53" />
        <SelectField form={form} name="dns_record_type" label={t("recordType")} options={["A","AAAA","CNAME","MX","TXT","NS"].map((v) => ({ value: v, label: v }))} />
      </InputGroup>
    </div>
  );
}

export function TLSConfigFields({ form }: ConfigFieldsProps) {
  const t = useTranslations("monitors.fields");
  return (
    <div className="flex flex-col gap-5">
      <InputGroup label="TLS" icon={Gauge}>
        <NumberField form={form} name="tls_port" label={t("port")} min={1} max={65535} fullWidth />
        <TextField form={form} name="tls_server_name" label={t("serverName")} dir="ltr" placeholder="example.com" />
        <NumberField form={form} name="tls_warning_days" label={t("warningDays")} min={1} suffix="d" fullWidth />
        <NumberField form={form} name="tls_critical_days" label={t("criticalDays")} min={1} suffix="d" fullWidth />
      </InputGroup>
    </div>
  );
}

export function DomainExpirationConfigFields({ form }: ConfigFieldsProps) {
  const t = useTranslations("monitors.fields");
  return (
    <div className="flex flex-col gap-5">
      <InputGroup label={t("domainSettings")} icon={Gauge}>
        <NumberField form={form} name="domain_warning_days" label={t("warningDays")} min={1} suffix="d" fullWidth />
        <NumberField form={form} name="domain_critical_days" label={t("criticalDays")} min={1} suffix="d" fullWidth />
      </InputGroup>
    </div>
  );
}

export function SMTPConfigFields({ form }: ConfigFieldsProps) {
  const t = useTranslations("monitors.fields");
  return (
    <div className="flex flex-col gap-5">
      <InputGroup label="SMTP" icon={Gauge}>
        <NumberField form={form} name="smtp_port" label={t("port")} min={1} max={65535} fullWidth />
        <SelectField form={form} name="smtp_mode" label={t("mode")} options={[
          { value: "plain", label: t("modePlain") },
          { value: "starttls", label: t("modeStarttls") },
        ]} />
      </InputGroup>
    </div>
  );
}

export function NTPConfigFields({ form }: ConfigFieldsProps) {
  const t = useTranslations("monitors.fields");
  return (
    <div className="flex flex-col gap-5">
      <InputGroup label="NTP" icon={Gauge}>
        <NumberField form={form} name="ntp_port" label={t("port")} min={1} max={65535} fullWidth />
        <NumberField form={form} name="ntp_version" label={t("ntpVersion")} min={3} max={4} fullWidth />
        <NumberField form={form} name="ntp_max_offset_millis" label={t("maxOffset")} min={1} fullWidth />
        <NumberField form={form} name="ntp_max_round_trip_millis" label={t("maxRoundTrip")} min={1} fullWidth />
      </InputGroup>
    </div>
  );
}
