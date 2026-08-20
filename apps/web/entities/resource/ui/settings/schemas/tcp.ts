import type { MonitoringTypeSchema } from "../monitoring-schema";

export const tcpSchema: MonitoringTypeSchema = {
  type: "tcp",
  title: { en: "TCP Port", fa: "پورت TCP" },
  description: { en: "TCP port reachability", fa: "دسترس‌پذیری پورت TCP" },
  target: {
    label: { en: "Target", fa: "هدف" },
    widget: "readonly",
  },
  execution: {
    minimumIntervalSeconds: 10,
    defaultIntervalSeconds: 60,
    defaultTimeoutMillis: 5000,
    defaultRetries: 1,
  },
  configFields: [
    {
      key: "port",
      widget: "number",
      label: { en: "Port", fa: "پورت" },
      section: "configuration",
      min: 1,
      max: 65535,
      required: true,
      defaultValue: 80,
    },
  ],
  healthRules: [
    {
      key: "reachability",
      label: { en: "Availability", fa: "دسترس‌پذیری" },
      direction: "boolean",
      description: { en: "Connection failure is critical", fa: "عدم برقراری اتصال بحرانی است" },
      defaultEnabled: true,
    },
    {
      key: "connect_time_ms",
      label: { en: "Connect time", fa: "زمان اتصال" },
      unit: "ms",
      direction: "higher_is_worse",
      description: { en: "Time to establish the connection", fa: "زمان برقراری اتصال" },
      defaultEnabled: true,
      defaults: { warning: 500, critical: 2000 },
    },
  ],
};
