"use client";

import { useId, useSyncExternalStore } from "react";
import { useTranslations } from "next-intl";
import { useTheme } from "next-themes";

import { cn } from "@/shared/utils/cn";

const emptySubscribe = () => () => {};

export function ThemeToggle() {
  const t = useTranslations("theme");
  const id = useId();
  const { resolvedTheme, setTheme } = useTheme();
  const mounted = useSyncExternalStore(
    emptySubscribe,
    () => true,
    () => false,
  );

  const isDark = mounted && resolvedTheme === "dark";
  const maskId = `moon-mask-${id}`;

  return (
    <label
      role="switch"
      aria-checked={isDark}
      aria-label={t("label")}
      className={cn(
        "theme-toggle relative inline-flex size-6 shrink-0 cursor-pointer items-center justify-center outline-none focus-visible:rounded-md focus-visible:ring-2 focus-visible:ring-ring/50",
        isDark && "dark",
      )}
    >
      <input
        type="checkbox"
        checked={isDark}
        onChange={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
        className="sr-only"
      />
      <svg
        width="16"
        height="16"
        viewBox="0 0 20 20"
        fill="currentColor"
        stroke="none"
        aria-hidden
        className="size-4"
      >
        <mask id={maskId}>
          <rect x="0" y="0" width="20" height="20" fill="white" />
          <circle className="moon-mask-circle" cx="11" cy="3" r="8" fill="black" />
        </mask>
        <circle
          className="sun-moon"
          cx="10"
          cy="10"
          r="8"
          mask={`url(#${maskId})`}
        />
        <g>
          <circle className="sun-ray" cx="18" cy="10" r="1.5" />
          <circle className="sun-ray" cx="14" cy="16.928" r="1.5" />
          <circle className="sun-ray" cx="6" cy="16.928" r="1.5" />
          <circle className="sun-ray" cx="2" cy="10" r="1.5" />
          <circle className="sun-ray" cx="6" cy="3.1718" r="1.5" />
          <circle className="sun-ray" cx="14" cy="3.1718" r="1.5" />
        </g>
      </svg>
    </label>
  );
}
