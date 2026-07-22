"use client";

import type { UseFormReturn } from "react-hook-form";
import { useTranslations } from "next-intl";

import {
  NumberField,
  SelectField,
  SwitchField,
  TextAreaField,
  TextField,
} from "@/components/monitors/form-fields";
import type { MonitorFormValues } from "@/lib/schemas";

type ConfigFieldsProps = { form: UseFormReturn<MonitorFormValues> };

export function HTTPConfigFields({ form }: ConfigFieldsProps) {
  const t = useTranslations("monitors.fields");

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <SelectField
        form={form}
        name="http_method"
        label={t("method")}
        options={["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"].map(
          (method) => ({ value: method, label: method }),
        )}
      />
      <TextField
        form={form}
        name="http_expected_status_codes"
        label={t("expectedStatusCodes")}
        hint={t("statusCodesHint")}
        dir="ltr"
        placeholder="200, 204"
      />
      <TextField
        form={form}
        name="http_body_contains"
        label={t("bodyContains")}
        dir="ltr"
      />
      <NumberField
        form={form}
        name="http_max_redirects"
        label={t("maxRedirects")}
        min={0}
        max={20}
      />
      <div className="sm:col-span-2">
        <TextAreaField
          form={form}
          name="http_headers"
          label={t("headers")}
          hint={t("headersHint")}
          placeholder="User-Agent: MonitoringPlatform/1.0"
        />
      </div>
      <div className="sm:col-span-2">
        <TextAreaField
          form={form}
          name="http_body"
          label={t("requestBody")}
          rows={2}
        />
      </div>
      <SwitchField form={form} name="http_follow_redirects" label={t("followRedirects")} />
      <SwitchField form={form} name="http_verify_tls" label={t("verifyTls")} />
    </div>
  );
}

export function TCPConfigFields({ form }: ConfigFieldsProps) {
  const t = useTranslations("monitors.fields");

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <NumberField form={form} name="tcp_port" label={t("port")} min={1} max={65535} />
    </div>
  );
}

export function DNSConfigFields({ form }: ConfigFieldsProps) {
  const t = useTranslations("monitors.fields");

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <TextField
        form={form}
        name="dns_server"
        label={t("dnsServer")}
        dir="ltr"
        placeholder="1.1.1.1:53"
      />
      <SelectField
        form={form}
        name="dns_record_type"
        label={t("recordType")}
        options={["A", "AAAA", "CNAME", "MX", "TXT", "NS"].map((recordType) => ({
          value: recordType,
          label: recordType,
        }))}
      />
      <div className="sm:col-span-2">
        <TextField
          form={form}
          name="dns_expected_values"
          label={t("expectedValues")}
          hint={t("csvHint")}
          dir="ltr"
          placeholder="203.0.113.10"
        />
      </div>
    </div>
  );
}

export function PingConfigFields({ form }: ConfigFieldsProps) {
  const t = useTranslations("monitors.fields");

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <NumberField form={form} name="ping_packet_count" label={t("packetCount")} min={1} max={20} />
      <NumberField
        form={form}
        name="ping_packet_interval_millis"
        label={t("packetInterval")}
        min={10}
        max={10000}
      />
      <div className="sm:col-span-2 mt-2 border-t border-border/60 pt-4">
        <p className="mb-3 text-sm font-medium">{t("statusThresholds")}</p>
        <div className="grid gap-4 sm:grid-cols-2">
          <NumberField form={form} name="ping_warning_latency_millis" label={t("warningLatencyMillis")} min={1} max={60000} />
          <NumberField form={form} name="ping_critical_latency_millis" label={t("criticalLatencyMillis")} min={1} max={60000} />
        </div>
      </div>
    </div>
  );
}

export function TLSConfigFields({ form }: ConfigFieldsProps) {
  const t = useTranslations("monitors.fields");

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <NumberField form={form} name="tls_port" label={t("port")} min={1} max={65535} />
      <TextField
        form={form}
        name="tls_server_name"
        label={t("serverName")}
        dir="ltr"
        placeholder="example.com"
      />
      <SelectField
        form={form}
        name="tls_min_version"
        label={t("minTlsVersion")}
        options={[
          { value: "1.2", label: "TLS 1.2" },
          { value: "1.3", label: "TLS 1.3" },
        ]}
      />
      <div className="grid grid-cols-2 gap-4">
        <NumberField form={form} name="tls_warning_days" label={t("warningDays")} min={1} />
        <NumberField form={form} name="tls_critical_days" label={t("criticalDays")} min={1} />
      </div>
      <TextField
        form={form}
        name="tls_expected_issuer"
        label={t("expectedIssuer")}
        dir="ltr"
      />
      <TextField
        form={form}
        name="tls_expected_fingerprint"
        label={t("expectedFingerprint")}
        dir="ltr"
      />
      <SwitchField form={form} name="tls_verify_chain" label={t("verifyChain")} />
      <SwitchField form={form} name="tls_verify_hostname" label={t("verifyHostname")} />
    </div>
  );
}

export function DomainExpirationConfigFields({ form }: ConfigFieldsProps) {
  const t = useTranslations("monitors.fields");

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <NumberField form={form} name="domain_warning_days" label={t("warningDays")} min={1} />
      <NumberField form={form} name="domain_critical_days" label={t("criticalDays")} min={1} />
      <TextField
        form={form}
        name="domain_expected_registrar"
        label={t("expectedRegistrar")}
        dir="ltr"
      />
      <TextField
        form={form}
        name="domain_expected_nameservers"
        label={t("expectedNameservers")}
        hint={t("csvHint")}
        dir="ltr"
        placeholder="ns1.example.com, ns2.example.com"
      />
      <SwitchField
        form={form}
        name="domain_check_nameservers"
        label={t("checkNameservers")}
      />
    </div>
  );
}

export function SMTPConfigFields({ form }: ConfigFieldsProps) {
  const t = useTranslations("monitors.fields");

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <NumberField form={form} name="smtp_port" label={t("port")} min={1} max={65535} />
      <SelectField
        form={form}
        name="smtp_mode"
        label={t("mode")}
        options={[
          { value: "plain", label: t("modePlain") },
          { value: "starttls", label: t("modeStarttls") },
          { value: "implicit_tls", label: t("modeImplicit") },
        ]}
      />
      <TextField
        form={form}
        name="smtp_ehlo_domain"
        label={t("ehloDomain")}
        dir="ltr"
        placeholder="monitor.example.com"
      />
      <TextField
        form={form}
        name="smtp_expected_banner"
        label={t("expectedBanner")}
        dir="ltr"
      />
      <TextField
        form={form}
        name="smtp_expected_capabilities"
        label={t("expectedCapabilities")}
        hint={t("csvHint")}
        dir="ltr"
        placeholder="STARTTLS, SIZE"
      />
      <div className="grid grid-cols-1 gap-4">
        <SwitchField form={form} name="smtp_require_starttls" label={t("requireStarttls")} />
        <SwitchField form={form} name="smtp_verify_tls" label={t("verifyTls")} />
      </div>
    </div>
  );
}

export function NTPConfigFields({ form }: ConfigFieldsProps) {
  const t = useTranslations("monitors.fields");

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <NumberField form={form} name="ntp_port" label={t("port")} min={1} max={65535} />
      <SelectField
        form={form}
        name="ntp_version"
        label={t("ntpVersion")}
        options={[
          { value: "3", label: "NTPv3" },
          { value: "4", label: "NTPv4" },
        ]}
      />
      <NumberField form={form} name="ntp_max_offset_millis" label={t("maxOffset")} min={1} />
      <NumberField
        form={form}
        name="ntp_max_round_trip_millis"
        label={t("maxRoundTrip")}
        min={1}
      />
      <NumberField form={form} name="ntp_stratum_min" label={t("stratumMin")} min={1} max={16} />
      <NumberField form={form} name="ntp_stratum_max" label={t("stratumMax")} min={1} max={16} />
    </div>
  );
}
