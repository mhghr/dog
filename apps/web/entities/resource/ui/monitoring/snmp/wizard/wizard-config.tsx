"use client";

import { useEffect, useState } from "react";
import { Check, Copy, ShieldCheck } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";

import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import { cn } from "@/shared/utils/cn";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/shared/ui/select";
import { resourcesApi } from "@/entities/resource/api/resource.api";

const SECRET_MASK = "••••••••";

function SecretField({
  value,
  placeholder,
  configured,
  isFa,
  onChange,
}: {
  value: string;
  placeholder: string;
  configured: boolean;
  isFa: boolean;
  onChange: (value: string) => void;
}) {
  return (
    <Input
      type="password"
      autoComplete="new-password"
      value={configured && value === "" ? SECRET_MASK : value}
      placeholder={configured ? "•••••••• (unchanged)" : placeholder}
      className="h-10"
      dir="ltr"
      onChange={(e) => {
        // Empty/masked means "keep the stored value".
        if (e.target.value === SECRET_MASK) return;
        onChange(e.target.value);
      }}
    />
  );
}

function LabeledField({
  label,
  children,
  hint,
}: {
  label: string;
  children: React.ReactNode;
  hint?: string;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label className="text-xs font-medium text-muted-foreground">{label}</Label>
      {children}
      {hint && <p className="text-[10px] text-muted-foreground/70">{hint}</p>}
    </div>
  );
}

export function SnmpConfigForm({
  config,
  isFa,
  resourceId,
  monitorId,
  onChange,
  credentialConfigured,
}: {
  config: Record<string, unknown>;
  isFa: boolean;
  resourceId: string;
  monitorId?: string;
  onChange: (key: string, value: unknown) => void;
  credentialConfigured: boolean;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const version = String(config.version ?? "3");
  const securityLevel = String(config.security_level ?? "authPriv");

  const sourceIps = useQuery({
    queryKey: ["snmp", "source-ips"],
    queryFn: () => resourcesApi.snmpSourceIps(),
    staleTime: 60_000,
  });

  const [copied, setCopied] = useState(false);
  const copyIps = () => {
    const ips = sourceIps.data?.ips ?? [];
    if (ips.length === 0) return;
    void navigator.clipboard.writeText(ips.join(", "));
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  const str = (key: string, fallback = "") => (typeof config[key] === "string" ? (config[key] as string) : fallback);
  const num = (key: string, fallback: number) => (typeof config[key] === "number" ? (config[key] as number) : fallback);

  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
      {/* SNMP configuration */}
      <div className="flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold text-foreground">{t("SNMP Configuration", "پیکربندی SNMP")}</h3>
          {credentialConfigured && (
            <Badge variant="outline" className="gap-1 border-emerald-500/30 bg-emerald-500/10 text-emerald-500">
              <Check className="size-3" />
              {t("Configured", "پیکربندی شده")}
            </Badge>
          )}
        </div>

        <div className="inline-flex w-fit items-center gap-0.5 rounded-lg border border-border/60 bg-muted/25 p-0.5">
          {(["3", "2c"] as const).map((v) => (
            <Button
              key={v}
              type="button"
              size="sm"
              variant="ghost"
              className={cn("h-8 px-3 text-xs", version === v ? "bg-card text-foreground shadow-sm" : "text-muted-foreground")}
              onClick={() => onChange("version", v)}
            >
              {v === "3" ? "SNMPv3" : "SNMPv2c"}
              {v === "3" && (
                <ShieldCheck className="ms-1 size-3.5 text-emerald-500" />
              )}
            </Button>
          ))}
        </div>
        {version === "3" && (
          <p className="text-[11px] text-emerald-500">
            {t("SNMPv3 recommended for production", "SNMPv3 برای تولید پیشنهاد می‌شود")}
          </p>
        )}

        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <LabeledField label={t("Port", "پورت")}>
            <Input
              type="number"
              value={num("port", 161)}
              min={1}
              max={65535}
              className="h-10"
              dir="ltr"
              onChange={(e) => onChange("port", Number(e.target.value))}
            />
          </LabeledField>

          {version === "2c" ? (
            <LabeledField label={t("Community string", "رشته Community")} hint={t("Use a read-only community", "از Community فقط‌خواندنی استفاده کنید")}>
              <SecretField
                value={str("community")}
                placeholder="public"
                configured={credentialConfigured}
                isFa={isFa}
                onChange={(value) => onChange("community", value)}
              />
            </LabeledField>
          ) : (
            <LabeledField label={t("Username", "نام کاربری")}>
              <Input
                value={str("username")}
                className="h-10"
                dir="ltr"
                onChange={(e) => onChange("username", e.target.value)}
              />
            </LabeledField>
          )}
        </div>

        {version === "3" && (
          <>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <LabeledField label={t("Security level", "سطح امنیت")}>
                <Select value={securityLevel} onValueChange={(value) => onChange("security_level", value)}>
                  <SelectTrigger className="w-full" style={{ height: "2.5rem" }}>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="authPriv">authPriv (recommended)</SelectItem>
                    <SelectItem value="authNoPriv">authNoPriv</SelectItem>
                    <SelectItem value="noAuthNoPriv">noAuthNoPriv</SelectItem>
                  </SelectContent>
                </Select>
              </LabeledField>
              <LabeledField label={t("Authentication protocol", "پروتکل احراز هویت")}>
                <Select
                  value={str("authentication_protocol", "SHA-256")}
                  onValueChange={(value) => onChange("authentication_protocol", value)}
                >
                  <SelectTrigger className="w-full" style={{ height: "2.5rem" }}>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="SHA-256">SHA-256 (recommended)</SelectItem>
                    <SelectItem value="SHA">SHA</SelectItem>
                    <SelectItem value="MD5">MD5</SelectItem>
                  </SelectContent>
                </Select>
              </LabeledField>
            </div>

            {(securityLevel === "authNoPriv" || securityLevel === "authPriv") && (
              <LabeledField label={t("Authentication password", "رمز احراز هویت")}>
                <SecretField
                  value={str("authentication_secret")}
                  placeholder="••••••••"
                  configured={credentialConfigured}
                  isFa={isFa}
                  onChange={(value) => onChange("authentication_secret", value)}
                />
              </LabeledField>
            )}

            {securityLevel === "authPriv" && (
              <>
                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                  <LabeledField label={t("Privacy protocol", "پروتکل رمزنگاری")}>
                    <Select
                      value={str("privacy_protocol", "AES-256")}
                      onValueChange={(value) => onChange("privacy_protocol", value)}
                    >
                      <SelectTrigger className="w-full" style={{ height: "2.5rem" }}>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="AES-256">AES-256 (recommended)</SelectItem>
                        <SelectItem value="AES">AES</SelectItem>
                        <SelectItem value="DES">DES</SelectItem>
                      </SelectContent>
                    </Select>
                  </LabeledField>
                  <div />
                </div>
                <LabeledField label={t("Privacy password", "رمز رمزنگاری")}>
                  <SecretField
                    value={str("privacy_secret")}
                    placeholder="••••••••"
                    configured={credentialConfigured}
                    isFa={isFa}
                    onChange={(value) => onChange("privacy_secret", value)}
                  />
                </LabeledField>
              </>
            )}
          </>
        )}

        <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <LabeledField label={t("Timeout (s)", "تایم‌اوت (ثانیه)")}>
            <Input type="number" value={num("timeout_seconds", 3)} min={1} max={30} className="h-10" dir="ltr" onChange={(e) => onChange("timeout_seconds", Number(e.target.value))} />
          </LabeledField>
          <LabeledField label={t("Retries", "تلاش مجدد")}>
            <Input type="number" value={num("retries", 1)} min={0} max={5} className="h-10" dir="ltr" onChange={(e) => onChange("retries", Number(e.target.value))} />
          </LabeledField>
          <LabeledField label={t("GETBULK reps", "تکرار GETBULK")}>
            <Input type="number" value={num("max_repetitions", 10)} min={1} max={100} className="h-10" dir="ltr" onChange={(e) => onChange("max_repetitions", Number(e.target.value))} />
          </LabeledField>
        </div>
      </div>

      {/* Device Configuration Guide */}
      <div className="flex flex-col gap-4">
        <h3 className="text-sm font-semibold text-foreground">{t("Device Configuration Guide", "راهنمای پیکربندی دستگاه")}</h3>
        <p className="text-xs text-muted-foreground">
          {t(
            "Enable the SNMP agent on the device and allow UDP/161 from the Dog SNMP collector IPs below. These CLI commands are guidance only — Dog never executes commands on your device.",
            "SNMP Agent را روی دستگاه فعال و UDP/161 را برای IPهای کلکتور Dog باز کنید. این دستورات فقط راهنما هستند — Dog هیچ دستوری روی دستگاه اجرا نمی‌کند.",
          )}
        </p>

        <div className="rounded-lg border border-border/40 p-3">
          <span className="text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">
            {t("Cisco IOS — SNMPv2c read-only", "Cisco IOS — SNMPv2c فقط‌خواندنی")}
          </span>
          <pre className="mt-2 overflow-x-auto rounded-md bg-muted/40 p-3 font-mono text-[11px] leading-relaxed text-foreground" dir="ltr">
{`snmp-server community dog-ro RO
snmp-server location DATA-CENTER
snmp-server contact noc@example.com`}
          </pre>
        </div>

        <div className="rounded-lg border border-border/40 p-3">
          <span className="text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">
            {t("Cisco IOS — SNMPv3 authPriv (recommended)", "Cisco IOS — SNMPv3 authPriv (پیشنهادی)")}
          </span>
          <pre className="mt-2 overflow-x-auto rounded-md bg-muted/40 p-3 font-mono text-[11px] leading-relaxed text-foreground" dir="ltr">
{`snmp-server group dog-grp v3 priv
snmp-server user dog-user dog-grp v3 auth sha <auth-pass> priv aes 256 <priv-pass>
snmp-server location DATA-CENTER`}
          </pre>
        </div>

        <div className="rounded-lg border border-amber-500/25 bg-amber-500/5 p-3">
          <span className="text-[11px] font-semibold text-amber-600 dark:text-amber-400">
            {t("Firewall note", "نکته فایروال")}
          </span>
          <p className="mt-1 text-[11px] text-muted-foreground">
            {t(
              "Do not open UDP/161 to the whole internet. Allow it only from the Dog SNMP collector source IPs below.",
              "UDP/161 را برای کل اینترنت باز نکنید. فقط از IPهای کلکتور SNMP دوج اجازه دهید.",
            )}
          </p>
          <div className="mt-2 flex flex-wrap items-center gap-1.5">
            {(sourceIps.data?.ips ?? []).map((ip) => (
              <code key={ip} className="rounded bg-muted/60 px-1.5 py-0.5 font-mono text-[10px] text-foreground" dir="ltr">
                {ip}
              </code>
            ))}
            {sourceIps.data?.ips.length === 0 && (
              <span className="text-[10px] text-muted-foreground">
                {t("No collector IPs configured (SNMP_SOURCE_IPS)", "IP کلکتوری تنظیم نشده (SNMP_SOURCE_IPS)")}
              </span>
            )}
          </div>
          {sourceIps.data && sourceIps.data.ips.length > 0 && (
            <Button type="button" size="sm" variant="outline" className="mt-2 h-7 text-xs" onClick={copyIps}>
              {copied ? <Check className="size-3.5 text-emerald-500" /> : <Copy className="size-3.5" />}
              {copied ? t("Copied", "کپی شد") : t("Copy IPs", "کپی IPها")}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
