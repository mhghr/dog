import type { MonitoringTypeSchema } from "../monitoring-schema";

export const dnsSchema: MonitoringTypeSchema = {
  type: "dns",
  title: { en: "DNS", fa: "DNS" },
  description: { en: "DNS record resolution", fa: "تفکیک رکورد DNS" },
  target: {
    label: { en: "Hostname", fa: "نام میزبان" },
    widget: "readonly",
  },
  execution: {
    minimumIntervalSeconds: 30,
    defaultIntervalSeconds: 60,
    defaultTimeoutMillis: 5000,
    defaultRetries: 1,
  },
  configFields: [
    {
      key: "record_type",
      widget: "select",
      label: { en: "Record type", fa: "نوع رکورد" },
      section: "configuration",
      options: ["A", "AAAA", "CNAME", "MX", "TXT", "NS"],
      defaultValue: "A",
    },
    {
      key: "resolver",
      widget: "text",
      label: { en: "Resolver", fa: "حل‌کننده (Resolver)" },
      help: { en: "Leave empty to use the system resolver, e.g. 8.8.8.8:53", fa: "خالی بگذارید تا از resolver سیستم استفاده شود، مثلاً 8.8.8.8:53" },
      section: "configuration",
      placeholder: "System resolver",
    },
    {
      key: "expected_values",
      widget: "text",
      label: { en: "Expected answers", fa: "پاسخ‌های مورد انتظار" },
      help: { en: "Comma-separated values the answer must match", fa: "مقادیر جدا شده با ویرگول که پاسخ باید با آن‌ها مطابقت کند" },
      section: "configuration",
      placeholder: "1.2.3.4",
    },
  ],
  healthRules: [
    {
      key: "reachability",
      label: { en: "Availability", fa: "دسترس‌پذیری" },
      direction: "boolean",
      description: { en: "Query failure is critical", fa: "عدم موفقیت query بحرانی است" },
      defaultEnabled: true,
    },
    {
      key: "response_time_ms",
      label: { en: "Response time", fa: "زمان پاسخ" },
      unit: "ms",
      direction: "higher_is_worse",
      description: { en: "DNS query duration", fa: "مدت query" },
      defaultEnabled: true,
      defaults: { warning: 500, critical: 2000 },
    },
    {
      key: "expected_record_match",
      label: { en: "Answer validation", fa: "اعتبارسنجی پاسخ" },
      direction: "boolean",
      description: { en: "Answer mismatch is critical", fa: "عدم تطابق پاسخ بحرانی است" },
      defaultEnabled: false,
    },
  ],
};
