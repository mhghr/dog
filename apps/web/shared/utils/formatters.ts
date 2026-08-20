export function formatNumber(value: number, locale: string): string {
  return new Intl.NumberFormat(locale).format(value);
}

export function formatPercent(
  value: number | null | undefined,
  locale: string,
  digits = 2,
): string {
  if (value === null || value === undefined || Number.isNaN(value)) {
    return "—";
  }

  return `${new Intl.NumberFormat(locale, {
    maximumFractionDigits: digits,
  }).format(value)}%`;
}

export function formatDuration(
  millis: number | null | undefined,
  locale: string,
): string {
  if (millis === null || millis === undefined || Number.isNaN(millis)) {
    return "—";
  }

  if (millis < 1000) {
    return `${formatNumber(Math.round(millis), locale)} ms`;
  }

  return `${new Intl.NumberFormat(locale, {
    maximumFractionDigits: 2,
  }).format(millis / 1000)} s`;
}

export function formatBytes(
  value: number | null | undefined,
  locale: string,
): string {
  if (value === null || value === undefined || Number.isNaN(value)) {
    return "—";
  }

  if (value < 1024) {
    return `${formatNumber(Math.round(value), locale)} B`;
  }
  if (value < 1024 * 1024) {
    return `${new Intl.NumberFormat(locale, { maximumFractionDigits: 1 }).format(value / 1024)} KB`;
  }
  return `${new Intl.NumberFormat(locale, { maximumFractionDigits: 1 }).format(value / (1024 * 1024))} MB`;
}

export function formatDateTime(
  value: string | null | undefined,
  locale: string,
): string {
  if (!value) {
    return "—";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "—";
  }

  return new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeStyle: "medium",
  }).format(date);
}

const RELATIVE_STEPS: Array<{ unit: Intl.RelativeTimeFormatUnit; seconds: number }> = [
  { unit: "year", seconds: 31536000 },
  { unit: "month", seconds: 2592000 },
  { unit: "week", seconds: 604800 },
  { unit: "day", seconds: 86400 },
  { unit: "hour", seconds: 3600 },
  { unit: "minute", seconds: 60 },
  { unit: "second", seconds: 1 },
];

export function formatRelativeTime(
  value: string | null | undefined,
  locale: string,
): string {
  if (!value) {
    return "—";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "—";
  }

  const deltaSeconds = (date.getTime() - Date.now()) / 1000;
  const absolute = Math.abs(deltaSeconds);

  const formatter = new Intl.RelativeTimeFormat(locale, { numeric: "auto" });

  for (const step of RELATIVE_STEPS) {
    if (absolute >= step.seconds || step.unit === "second") {
      return formatter.format(
        Math.round(deltaSeconds / step.seconds),
        step.unit,
      );
    }
  }

  return "—";
}

export function formatInterval(seconds: number, locale: string): string {
  if (seconds % 86400 === 0 && seconds >= 86400) {
    return new Intl.NumberFormat(locale).format(seconds / 86400) + "d";
  }
  if (seconds % 3600 === 0 && seconds >= 3600) {
    return new Intl.NumberFormat(locale).format(seconds / 3600) + "h";
  }
  if (seconds % 60 === 0 && seconds >= 60) {
    return new Intl.NumberFormat(locale).format(seconds / 60) + "m";
  }
  return new Intl.NumberFormat(locale).format(seconds) + "s";
}
