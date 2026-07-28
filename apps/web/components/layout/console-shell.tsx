"use client";

import { useEffect } from "react";
import { useLocale, useTranslations } from "next-intl";

import { BrandMark } from "@/components/layout/brand-mark";
import { ConsoleBreadcrumbs } from "@/components/layout/console-breadcrumbs";
import { LanguageSwitcher } from "@/components/layout/language-switcher";
import { ThemeToggle } from "@/components/layout/theme-toggle";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Sidebar,
  SidebarContent as SidebarContentWrapper,
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarTrigger,
  useSidebar,
} from "@/components/ui/sidebar";
import { useLogout, useMe } from "@/hooks/use-auth";
import { useLiveResults } from "@/hooks/use-live-results";
import { Link, usePathname, useRouter } from "@/i18n/navigation";
import { MonitorCheck, PanelTop } from "lucide-react";
import type { AppIcon } from "@/lib/icons";
import {
  Gauge,
  GearSix,
  MapPin,
  SignOut,
  SquaresFour,
  UserCircle,
  Warning,
} from "@/lib/icons";

interface NavigationItem {
  href: string;
  labelKey: "dashboard" | "monitors" | "alerts" | "statusPages" | "probes" | "system" | "settings";
  icon: AppIcon;
}

const NAVIGATION: NavigationItem[] = [
  { href: "/app/dashboard", labelKey: "dashboard", icon: SquaresFour },
  { href: "/app/nodes", labelKey: "monitors", icon: MonitorCheck },
  { href: "/app/status-pages", labelKey: "statusPages", icon: PanelTop },
  { href: "/app/alerts", labelKey: "alerts", icon: Warning },
  { href: "/app/probes", labelKey: "probes", icon: MapPin },
  { href: "/app/system", labelKey: "system", icon: Gauge },
  { href: "/app/settings", labelKey: "settings", icon: GearSix },
];

function SidebarNav() {
  const t = useTranslations("navigation");
  const pathname = usePathname();
  const { setOpenMobile } = useSidebar();

  return (
    <SidebarContentWrapper>
      <Link
        href="/app/dashboard"
        className="flex h-14 shrink-0 items-center gap-2.5 border-b border-sidebar-border px-4"
        onClick={() => setOpenMobile(false)}
      >
        <BrandMark />
        <span className="group-data-[collapsible=icon]:hidden text-xs font-semibold tracking-tight">
          {useTranslations("common")("appName")}
        </span>
      </Link>

      <ScrollArea className="flex-1 px-2">
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              {NAVIGATION.map((item) => {
                const Icon = item.icon;
                const active =
                  pathname === item.href || pathname.startsWith(`${item.href}/`);

                return (
                  <SidebarMenuItem key={item.labelKey}>
                    <SidebarMenuButton
                      asChild
                      isActive={active}
                      tooltip={t(item.labelKey)}
                      onClick={() => setOpenMobile(false)}
                    >
                      <Link href={item.href!}>
                        <Icon className="size-4" aria-hidden />
                        {t(item.labelKey)}
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                );
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </ScrollArea>
    </SidebarContentWrapper>
  );
}

function UserMenu() {
  const tAuth = useTranslations("auth");
  const router = useRouter();
  const meQuery = useMe();
  const logoutMutation = useLogout();

  const user = meQuery.data?.user;
  const displayName = user?.name || user?.email || user?.phone || "";

  const handleLogout = async () => {
    try {
      await logoutMutation.mutateAsync();
    } finally {
      router.replace("/login");
    }
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={displayName || "user"}
          className="relative me-3 shrink-0 cursor-pointer rounded-full outline-none transition-shadow focus-visible:ring-3 focus-visible:ring-ring/50 dark:hover:shadow-[0_0_12px_-2px_var(--primary)_/_30%]"
        >
          {user?.avatar_url ? (
            <img
              src={user.avatar_url}
              alt=""
              referrerPolicy="no-referrer"
              className="size-7 rounded-full ring-1 ring-border/60 dark:ring-primary/25"
            />
          ) : (
            <span className="grid size-7 place-items-center rounded-full bg-muted ring-1 ring-border/60 dark:ring-primary/25">
              <UserCircle className="size-4 text-muted-foreground" aria-hidden />
            </span>
          )}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-48">
        {displayName ? (
          <>
            <DropdownMenuLabel className="max-w-56 truncate" dir="auto">
              {displayName}
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
          </>
        ) : null}
        <DropdownMenuItem
          variant="destructive"
          onSelect={() => void handleLogout()}
          disabled={logoutMutation.isPending}
        >
          <SignOut className="size-4" aria-hidden />
          {tAuth("logout")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function AuthGate() {
  const meQuery = useMe();
  const router = useRouter();

  useLiveResults(meQuery.isSuccess);

  useEffect(() => {
    if (meQuery.isError) {
      router.replace("/login");
    }
  }, [meQuery.isError, router]);

  return null;
}

export function ConsoleShell({ children }: { children: React.ReactNode }) {
  const locale = useLocale();
  const sidebarSide = locale === "fa" ? "right" : "left";

  return (
    <SidebarProvider>
      <AuthGate />

      <Sidebar side={sidebarSide} collapsible="icon">
        <SidebarNav />
      </Sidebar>

      <main className="flex min-w-0 flex-1 flex-col">
        <header className="sticky top-0 z-20 flex h-14 items-center gap-2 border-b border-border bg-background/70 px-4 backdrop-blur-xl supports-[backdrop-filter]:bg-background/60 dark:border-primary/10">
          <SidebarTrigger />

          <ConsoleBreadcrumbs />

          <div className="flex-1" />

          <div className="flex items-center gap-1">
            <LanguageSwitcher />
            <span className="mx-1 h-5 w-px bg-border/70" aria-hidden />
            <ThemeToggle />
            <span className="mx-1 h-5 w-px bg-border/70" aria-hidden />
            <UserMenu />
          </div>
        </header>

        <main className="flex-1 px-6 py-6 lg:px-10">
          {children}
        </main>
      </main>
    </SidebarProvider>
  );
}
