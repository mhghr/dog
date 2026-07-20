"use client";

import { useEffect, useState } from "react";
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
import { Sheet, SheetContent, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { useLogout, useMe } from "@/hooks/use-auth";
import { useLiveResults } from "@/hooks/use-live-results";
import { Link, usePathname, useRouter } from "@/i18n/navigation";
import { MonitorCheck, PanelTop, Settings } from "lucide-react";
import { cn } from "@/lib/utils";
import type { AppIcon } from "@/lib/icons";
import {
  Gauge,
  List,
  MapPin,
  SignOut,
  SquaresFour,
  UserCircle,
} from "@/lib/icons";

interface NavigationItem {
  href: string;
  labelKey: "dashboard" | "monitors" | "statusPages" | "locations" | "system" | "settings";
  icon: AppIcon;
}

const NAVIGATION: NavigationItem[] = [
  { href: "/app/dashboard", labelKey: "dashboard", icon: LayoutDashboard },
  { href: "/app/nodes", labelKey: "monitors", icon: MonitorCheck },
  { href: "/app/status-pages", labelKey: "statusPages", icon: PanelTop },
  { href: "/app/locations", labelKey: "locations", icon: MapPin },
  { href: "/app/system", labelKey: "system", icon: CalendarClock },
  { href: "/app/settings", labelKey: "settings", icon: Settings },
];

function SidebarContent({ onNavigate }: { onNavigate?: () => void }) {
  const t = useTranslations("navigation");
  const tCommon = useTranslations("common");
  const pathname = usePathname();

  return (
    <div className="flex h-full flex-col">
      <Link
        href="/app/dashboard"
        className="flex h-14 shrink-0 items-center gap-2.5 border-b border-border/70 px-4"
        onClick={onNavigate}
      >
        <BrandMark />
        <span className="text-xs font-semibold tracking-tight">
          {tCommon("appName")}
        </span>
      </Link>

      <ScrollArea className="flex-1 px-2">
        <nav className="flex flex-col gap-0.5 pb-6 pt-5">
          {NAVIGATION.map((item) => {
            const Icon = item.icon;
            const active =
              pathname === item.href || pathname.startsWith(`${item.href}/`);

            return (
              <Link
                key={item.labelKey}
                href={item.href!}
                onClick={onNavigate}
                aria-current={active ? "page" : undefined}
                className={cn(
                  "flex items-center gap-2.5 rounded-lg px-2 py-1.5 text-sm transition-colors",
                  active
                    ? "bg-sidebar-accent font-medium text-foreground"
                    : "text-muted-foreground hover:bg-sidebar-accent/60 hover:text-foreground",
                )}
              >
                <span
                  className={cn(
                    "grid size-7 shrink-0 place-items-center rounded-full border transition-all",
                    active
                      ? "border-transparent bg-primary text-primary-foreground dark:shadow-[0_0_8px_-2px_var(--primary)_/_50%]"
                      : "border-border/70 bg-background text-muted-foreground",
                  )}
                >
                  <Icon className="size-3.5" aria-hidden />
                </span>
                {t(item.labelKey)}
              </Link>
            );
          })}
        </nav>
      </ScrollArea>
    </div>
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
            // eslint-disable-next-line @next/next/no-img-element
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

// AuthGate redirects to the login page when the session is missing or can
// no longer be refreshed; the SSE bridge only runs for signed-in users.
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
  const tCommon = useTranslations("common");
  const locale = useLocale();
  const [mobileOpen, setMobileOpen] = useState(false);
  const sheetSide = locale === "fa" ? "right" : "left";

  return (
    <div className="flex min-h-screen">
      <AuthGate />

      <aside className="sticky top-0 hidden h-screen w-60 shrink-0 border-e border-sidebar-border bg-sidebar lg:block">
        <SidebarContent />
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="sticky top-0 z-20 flex h-14 items-center gap-2 border-b border-border bg-background/70 px-4 backdrop-blur-xl supports-[backdrop-filter]:bg-background/60 dark:border-primary/10">
          <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
            <SheetTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="lg:hidden"
                aria-label={tCommon("openMenu")}
              >
                <List className="size-4" />
              </Button>
            </SheetTrigger>
            <SheetContent side={sheetSide} className="w-64 bg-sidebar p-0">
              <SheetTitle className="sr-only">{tCommon("appName")}</SheetTitle>
              <SidebarContent onNavigate={() => setMobileOpen(false)} />
            </SheetContent>
          </Sheet>

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
      </div>
    </div>
  );
}
