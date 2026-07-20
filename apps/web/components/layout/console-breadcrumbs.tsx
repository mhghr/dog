"use client";

import { Fragment } from "react";
import { useTranslations } from "next-intl";

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { useMonitor } from "@/hooks/use-monitor";
import { Link, usePathname } from "@/i18n/navigation";

const SECTION_LABELS = {
  dashboard: "dashboard",
  nodes: "monitors",
  "status-pages": "statusPages",
  monitors: "monitors",
  locations: "locations",
  system: "system",
  settings: "settings",
} as const;

const UUID_PATTERN =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

function MonitorName({ monitorId }: { monitorId: string }) {
  const monitorQuery = useMonitor(monitorId);
  return <>{monitorQuery.data?.name ?? "…"}</>;
}

export function ConsoleBreadcrumbs() {
  const tNav = useTranslations("navigation");
  const tMonitors = useTranslations("monitors");
  const tCommon = useTranslations("common");
  const pathname = usePathname();

  const segments = pathname.split("/").filter(Boolean);
  if (segments[0] !== "app") {
    return null;
  }

  const crumbs = segments.slice(1).map((segment, index, rest) => {
    const href = `/app/${rest.slice(0, index + 1).join("/")}`;

    let label: React.ReactNode = segment;
    if (segment in SECTION_LABELS) {
      label = tNav(SECTION_LABELS[segment as keyof typeof SECTION_LABELS]);
    } else if (segment === "new") {
      label = tMonitors("newMonitor");
    } else if (segment === "edit") {
      label = tCommon("edit");
    } else if (UUID_PATTERN.test(segment) && rest[index - 1] === "monitors") {
      label = <MonitorName monitorId={segment} />;
    }

    return { href, label, key: href };
  });

  if (crumbs.length === 0) {
    return null;
  }

  return (
    <Breadcrumb className="hidden min-w-0 sm:block">
      <BreadcrumbList className="flex-nowrap">
        {crumbs.map((crumb, index) => {
          const isLast = index === crumbs.length - 1;

          return (
            <Fragment key={crumb.key}>
              {index > 0 ? <BreadcrumbSeparator /> : null}
              <BreadcrumbItem className="min-w-0">
                {isLast ? (
                  <BreadcrumbPage className="truncate font-medium">
                    {crumb.label}
                  </BreadcrumbPage>
                ) : (
                  <BreadcrumbLink asChild>
                    <Link href={crumb.href} className="truncate">
                      {crumb.label}
                    </Link>
                  </BreadcrumbLink>
                )}
              </BreadcrumbItem>
            </Fragment>
          );
        })}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
