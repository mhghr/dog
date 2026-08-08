"use client";

import { useEffect, useRef, useState } from "react";
import { cn } from "@/shared/utils/cn";

interface AnimatedGroupProps {
  variants?: {
    container?: {
      visible?: {
        transition?: {
          staggerChildren?: number;
          delayChildren?: number;
        };
      };
    };
    item?: {
      hidden?: Record<string, unknown>;
      visible?: Record<string, unknown>;
      transition?: Record<string, unknown>;
    };
  };
  className?: string;
  children: React.ReactNode;
}

export function AnimatedGroup({
  variants,
  className,
  children,
}: AnimatedGroupProps) {
  const [inView, setInView] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

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

  const stagger = variants?.container?.visible?.transition?.staggerChildren ?? 0.05;
  const delayChildren = variants?.container?.visible?.transition?.delayChildren ?? 0;

  const itemHidden = variants?.item?.hidden ?? {};
  const itemVisible = variants?.item?.visible ?? {};
  const itemTransition = variants?.item?.transition ?? {};

  const childrenArray = Array.isArray(children) ? children : [children];

  return (
    <div ref={ref} className={className}>
      {childrenArray.map((child, i) => {
        const delay = delayChildren + i * stagger;

        const hiddenStyle: React.CSSProperties = {
          opacity: itemHidden.opacity as number | undefined ?? 0,
          filter: itemHidden.filter as string | undefined,
          transform: itemHidden.y !== undefined
            ? `translateY(${itemHidden.y}px)`
            : itemHidden.scale !== undefined
              ? `scale(${itemHidden.scale})`
              : undefined,
        };

        const visibleStyle: React.CSSProperties = {
          opacity: itemVisible.opacity as number | undefined ?? 1,
          filter: itemVisible.filter as string | undefined ?? "blur(0px)",
          transform: itemVisible.y !== undefined
            ? "translateY(0)"
            : itemVisible.scale !== undefined
              ? "scale(1)"
              : undefined,
          transitionProperty: (itemTransition as Record<string, string>)?.type === "spring"
            ? "opacity, filter, transform"
            : Object.keys({ ...itemHidden, ...itemVisible, filter: "", opacity: "", transform: "" })
                .filter((k) => itemHidden[k] !== undefined || itemVisible[k] !== undefined)
                .join(", "),
          transitionDuration: (itemTransition as Record<string, string>)?.duration as string | undefined ?? "0.5s",
          transitionTimingFunction: "cubic-bezier(0.16, 1, 0.3, 1)",
          transitionDelay: `${delay}s`,
        };

        return (
          <div
            key={i}
            style={inView ? visibleStyle : hiddenStyle}
            className={cn(inView ? "" : "pointer-events-none")}
          >
            {child}
          </div>
        );
      })}
    </div>
  );
}
