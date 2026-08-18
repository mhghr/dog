"use client";

import { useEffect, useRef, useState } from "react";
import { cn } from "@/shared/utils/cn";

type Preset = "fade-in-blur" | "fade-in" | "slide-up";

interface TextEffectProps {
  as?: "h1" | "h2" | "h3" | "p" | "span";
  preset?: Preset;
  per?: "word" | "line" | "char";
  speedSegment?: number;
  delay?: number;
  className?: string;
  children: string;
}

const BASE_DELAY = 0.04;

export function TextEffect({
  as: Tag = "p",
  preset = "fade-in-blur",
  per = "word",
  speedSegment = 0.3,
  delay = 0,
  className,
  children,
}: TextEffectProps) {
  const [inView, setInView] = useState(false);
  const ref = useRef<HTMLElement>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setInView(true);
          observer.disconnect();
        }
      },
      { threshold: 0.1 },
    );

    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  const tokens = per === "line"
    ? children.split("\n")
    : per === "char"
      ? children.split("")
      : children.split(" ");

  const segmentMs = speedSegment * 1000;
  const totalSegments = tokens.length;

  return (
    <Tag ref={ref as never} className={className}>
      {tokens.map((token, i) => (
        <span
          key={i}
          aria-hidden={!inView}
          className={cn(
            "inline-block transition-[opacity,transform,filter] ease-out",
            per === "line" && "block",
            inView
              ? cn(
                  preset === "fade-in-blur" && "translate-y-0 blur-0 opacity-100",
                  preset === "fade-in" && "opacity-100",
                  preset === "slide-up" && "translate-y-0 opacity-100",
                )
              : cn(
                  preset === "fade-in-blur" && "translate-y-3 blur-sm opacity-0",
                  preset === "fade-in" && "opacity-0",
                  preset === "slide-up" && "translate-y-4 opacity-0",
                ),
          )}
          style={{
            transitionDuration: `${segmentMs}ms`,
            transitionDelay: `${delay * 1000 + i * segmentMs / totalSegments * (totalSegments * BASE_DELAY * 1000)}ms`,
          }}
        >
          {token}
          {per === "word" && i < tokens.length - 1 ? "\u00A0" : null}
        </span>
      ))}
    </Tag>
  );
}
