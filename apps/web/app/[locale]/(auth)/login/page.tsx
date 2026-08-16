"use client";

import { useEffect, useState } from "react";
import { Loader2 } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";
import { toast } from "sonner";

import { Button } from "@/shared/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import { Separator } from "@/shared/ui/separator";
import { useRequestOtp, useVerifyOtp } from "@/platform/auth/use-auth";
import { useAuth } from "@/platform/auth/auth-provider";
import { useRouter } from "@/i18n/navigation";
import { ApiError } from "@/shared/api";

function GoogleIcon() {
  return (
    <svg viewBox="0 0 24 24" className="size-4" aria-hidden>
      <path
        fill="#4285F4"
        d="M23.49 12.27c0-.79-.07-1.54-.19-2.27H12v4.51h6.47a5.57 5.57 0 0 1-2.4 3.58v3h3.86c2.26-2.09 3.56-5.17 3.56-8.82Z"
      />
      <path
        fill="#34A853"
        d="M12 24c3.24 0 5.95-1.08 7.93-2.91l-3.86-3c-1.08.72-2.45 1.16-4.07 1.16-3.13 0-5.78-2.11-6.73-4.96H1.29v3.09A11.99 11.99 0 0 0 12 24Z"
      />
      <path
        fill="#FBBC05"
        d="M5.27 14.29A7.1 7.1 0 0 1 4.89 12c0-.8.14-1.57.38-2.29V6.62H1.29a11.86 11.86 0 0 0 0 10.76l3.98-3.09Z"
      />
      <path
        fill="#EA4335"
        d="M12 4.75c1.77 0 3.35.61 4.6 1.8l3.42-3.42C17.95 1.19 15.24 0 12 0 7.31 0 3.26 2.69 1.29 6.62l3.98 3.09C6.22 6.86 8.87 4.75 12 4.75Z"
      />
    </svg>
  );
}

export default function LoginPage() {
  const t = useTranslations("auth");
  const locale = useLocale();
  const router = useRouter();

  const [step, setStep] = useState<"phone" | "code">("phone");
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [resendIn, setResendIn] = useState(0);

  const requestOtp = useRequestOtp();
  const verifyOtp = useVerifyOtp();
  const { isSignedIn } = useAuth();

  // An already-authenticated visitor never sees the login form.
  useEffect(() => {
    if (isSignedIn) {
      router.replace("/app/dashboard");
    }
  }, [isSignedIn, router]);

  // Surface OAuth callback failures (?error=oauth) once.
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.get("error") === "oauth") {
      toast.error(t("oauthError"));
      params.delete("error");
      const query = params.toString();
      window.history.replaceState(
        null,
        "",
        window.location.pathname + (query ? `?${query}` : ""),
      );
    }
  }, [t]);

  useEffect(() => {
    if (resendIn <= 0) {
      return;
    }

    const handle = window.setInterval(
      () => setResendIn((current) => Math.max(0, current - 1)),
      1000,
    );
    return () => window.clearInterval(handle);
  }, [resendIn]);

  const sendCode = async () => {
    if (!phone.trim()) {
      toast.error(t("invalidPhone"));
      return;
    }

    try {
      const response = await requestOtp.mutateAsync(phone.trim());
      setStep("code");
      setCode("");
      setResendIn(response.retry_after || 60);

      if (response.dev_code) {
        toast.info(t("devCode", { code: response.dev_code }), {
          duration: 20000,
        });
      } else {
        toast.success(t("codeSent"));
      }
    } catch (error) {
      if (error instanceof ApiError && error.status === 429) {
        toast.error(t("rateLimited"));
      } else {
        toast.error(t("invalidPhone"));
      }
    }
  };

  const verifyCode = async () => {
    if (code.trim().length === 0) {
      return;
    }

    try {
      await verifyOtp.mutateAsync({ phone: phone.trim(), code: code.trim() });
      router.replace("/app/dashboard");
    } catch {
      toast.error(t("invalidCode"));
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("loginTitle")}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-5">
        <Button
          variant="outline"
          className="w-full gap-2"
          onClick={() =>
            window.location.assign(`/api/auth/google/start?locale=${locale}`)
          }
        >
          <GoogleIcon />
          {t("googleCta")}
        </Button>

        <div className="flex items-center gap-3">
          <Separator className="flex-1" />
          <span className="text-xs text-muted-foreground">{t("or")}</span>
          <Separator className="flex-1" />
        </div>

        {step === "phone" ? (
          <form
            className="flex flex-col gap-4"
            onSubmit={(event) => {
              event.preventDefault();
              void sendCode();
            }}
          >
            <div className="flex flex-col gap-2">
              <Label htmlFor="phone">{t("phoneLabel")}</Label>
              <Input
                id="phone"
                dir="ltr"
                inputMode="tel"
                autoComplete="tel"
                placeholder="0912 345 6789"
                value={phone}
                onChange={(event) => setPhone(event.target.value)}
              />
            </div>
            <Button
              type="submit"
              className="w-full gap-2"
              disabled={requestOtp.isPending}
            >
              {requestOtp.isPending ? (
                <Loader2 className="size-4 animate-spin" aria-hidden />
              ) : null}
              {t("sendCode")}
            </Button>
          </form>
        ) : (
          <form
            className="flex flex-col gap-4"
            onSubmit={(event) => {
              event.preventDefault();
              void verifyCode();
            }}
          >
            <p className="text-sm text-muted-foreground">
              {t("codeSentTo", { phone })}
            </p>
            <div className="flex flex-col gap-2">
              <Label htmlFor="otp-code">{t("codeLabel")}</Label>
              <Input
                id="otp-code"
                dir="ltr"
                inputMode="numeric"
                autoComplete="one-time-code"
                maxLength={6}
                className="text-center font-mono text-lg tracking-[0.5em]"
                value={code}
                onChange={(event) =>
                  setCode(event.target.value.replace(/[^0-9۰-۹]/g, ""))
                }
              />
            </div>
            <Button
              type="submit"
              className="w-full"
              disabled={verifyOtp.isPending || code.length < 6}
            >
              {verifyOtp.isPending ? (
                <Loader2 className="size-4 animate-spin" aria-hidden />
              ) : null}
              {t("verifyCta")}
            </Button>
            <div className="flex items-center justify-between text-sm">
              <button
                type="button"
                className="text-muted-foreground underline-offset-4 hover:underline"
                onClick={() => {
                  setStep("phone");
                  setCode("");
                }}
              >
                {t("changePhone")}
              </button>
              <button
                type="button"
                disabled={resendIn > 0 || requestOtp.isPending}
                className="text-primary underline-offset-4 hover:underline disabled:cursor-not-allowed disabled:text-muted-foreground disabled:no-underline"
                onClick={() => void sendCode()}
              >
                {resendIn > 0
                  ? t("resendIn", { seconds: resendIn })
                  : t("resend")}
              </button>
            </div>
          </form>
        )}
      </CardContent>
    </Card>
  );
}
