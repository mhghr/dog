"use client";

import { Monitor, Moon, Sun } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";
import { useTheme } from "next-themes";

import { PageHeader } from "@/components/common/page-header";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { usePathname, useRouter } from "@/i18n/navigation";
import { routing } from "@/i18n/routing";
import { cn } from "@/lib/utils";

export default function SettingsPage() {
  const t = useTranslations("settings");
  const tTheme = useTranslations("theme");
  const tLanguage = useTranslations("language");

  const { theme, setTheme } = useTheme();
  const locale = useLocale();
  const router = useRouter();
  const pathname = usePathname();

  const themeOptions = [
    { value: "system", label: tTheme("system"), icon: Monitor },
    { value: "light", label: tTheme("light"), icon: Sun },
    { value: "dark", label: tTheme("dark"), icon: Moon },
  ];

  return (
    <div className="mx-auto max-w-2xl">
      <PageHeader title={t("title")} subtitle={t("subtitle")} />

      <Card>
        <CardHeader>
          <CardTitle>{t("appearance")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-8">
          <div>
            <p className="text-sm font-medium">{t("themeTitle")}</p>
            <CardDescription className="mb-3">
              {t("themeDescription")}
            </CardDescription>
            <div className="grid grid-cols-3 gap-2">
              {themeOptions.map((option) => {
                const Icon = option.icon;
                const selected = (theme ?? "system") === option.value;

                return (
                  <button
                    key={option.value}
                    type="button"
                    onClick={() => setTheme(option.value)}
                    aria-pressed={selected}
                    className={cn(
                      "flex flex-col items-center gap-2 rounded-lg border px-3 py-4 text-sm transition-colors",
                      selected
                        ? "border-primary bg-primary/5 font-medium ring-1 ring-primary"
                        : "border-border text-muted-foreground hover:border-primary/40 hover:text-foreground",
                    )}
                  >
                    <Icon className="size-5" aria-hidden />
                    {option.label}
                  </button>
                );
              })}
            </div>
          </div>

          <div>
            <p className="text-sm font-medium">{t("languageTitle")}</p>
            <CardDescription className="mb-3">
              {t("languageDescription")}
            </CardDescription>
            <RadioGroup
              value={locale}
              onValueChange={(nextLocale) =>
                router.replace(pathname, {
                  locale: nextLocale as (typeof routing.locales)[number],
                })
              }
              className="flex flex-col gap-2"
            >
              {routing.locales.map((availableLocale) => (
                <div
                  key={availableLocale}
                  className="flex items-center gap-3 rounded-lg border border-border px-3 py-2.5"
                >
                  <RadioGroupItem
                    value={availableLocale}
                    id={`locale-${availableLocale}`}
                  />
                  <Label
                    htmlFor={`locale-${availableLocale}`}
                    className="cursor-pointer font-normal"
                  >
                    {tLanguage(availableLocale)}
                  </Label>
                </div>
              ))}
            </RadioGroup>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
