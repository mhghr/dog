"use client";

import type { LucideIcon } from "lucide-react";
import { Globe, Radio, Network, Plug, Server, Database, Code2, Router } from "lucide-react";

export interface MonitorTypeItem {
  id: string;
  title: string;
  titleFa: string;
  description: string;
  descriptionFa: string;
  icon: LucideIcon;
  tone: string;
}

export const MONITOR_TYPE_ITEMS: MonitorTypeItem[] = [
  {
    id: "http",
    title: "HTTP(S)",
    titleFa: "HTTP(S)",
    description: "Monitor HTTP/HTTPS endpoints",
    descriptionFa: "نقاط پایانی HTTP/HTTPS",
    icon: Globe,
    tone: "bg-primary/15 text-primary",
  },
  {
    id: "ping",
    title: "Ping",
    titleFa: "پینگ",
    description: "Monitor host availability via ICMP",
    descriptionFa: "دسترسی میزبان از طریق ICMP",
    icon: Radio,
    tone: "bg-emerald-500/15 text-emerald-500",
  },
  {
    id: "dns",
    title: "DNS",
    titleFa: "DNS",
    description: "Monitor DNS record resolution",
    descriptionFa: "وضوح رکوردهای DNS",
    icon: Network,
    tone: "bg-amber-500/15 text-amber-500",
  },
  {
    id: "tcp",
    title: "TCP Port",
    titleFa: "پورت TCP",
    description: "Monitor TCP port availability",
    descriptionFa: "در دسترس بودن پورت TCP",
    icon: Plug,
    tone: "bg-blue-500/15 text-blue-500",
  },
  {
    id: "server",
    title: "Server",
    titleFa: "سرور",
    description: "Monitor system resources",
    descriptionFa: "منابع سیستم",
    icon: Server,
    tone: "bg-violet-500/15 text-violet-500",
  },
  {
    id: "snmp",
    title: "SNMP Device",
    titleFa: "دستگاه شبکه",
    description: "Routers, switches and appliances via SNMP",
    descriptionFa: "روتر، سوئیچ و تجهیزات از طریق SNMP",
    icon: Router,
    tone: "bg-teal-500/15 text-teal-500",
  },
  {
    id: "db",
    title: "Database",
    titleFa: "پایگاه داده",
    description: "Monitor database performance",
    descriptionFa: "عملکرد پایگاه داده",
    icon: Database,
    tone: "bg-cyan-500/15 text-cyan-500",
  },
  {
    id: "script",
    title: "Custom Script",
    titleFa: "اسکریپت سفارشی",
    description: "Run custom scripts for monitoring",
    descriptionFa: "اجرای اسکریپت‌های سفارشی",
    icon: Code2,
    tone: "bg-rose-500/15 text-rose-500",
  },
];
