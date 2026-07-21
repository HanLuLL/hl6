import { useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { MailCheck } from "lucide-react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { AuthShell } from "@/components/auth/auth-shell";
import { Button } from "@/components/ui/button";
import { api, getErrorMessage } from "@/lib/api";

type VerifyMode = "registration" | "activation" | "reset";

function resolveMode(value: string | null): VerifyMode {
  if (value === "activation") return "activation";
  if (value === "reset") return "reset";
  return "registration";
}

const RESEND_COOLDOWN_SECONDS = 60;

export default function VerifyEmailPage() {
  const { t, i18n } = useTranslation();
  const [searchParams] = useSearchParams();
  const email = searchParams.get("email") ?? "";
  const mode = resolveMode(searchParams.get("mode"));
  const activation = mode === "activation";
  const [cooldown, setCooldown] = useState(0);
  const [resending, setResending] = useState(false);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, []);

  const startCooldown = () => {
    setCooldown(RESEND_COOLDOWN_SECONDS);
    if (timerRef.current) clearInterval(timerRef.current);
    timerRef.current = setInterval(() => {
      setCooldown((prev) => {
        if (prev <= 1) {
          if (timerRef.current) clearInterval(timerRef.current);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  };

  const resend = async () => {
    if (!email || cooldown > 0 || resending) return;
    setResending(true);
    try {
      const locale = i18n.resolvedLanguage ?? i18n.language;
      if (mode === "activation") {
        await api.requestActivation({ email, locale });
      } else if (mode === "reset") {
        await api.requestPasswordReset({ email, locale });
      } else {
        await api.requestRegistration({ email, locale });
      }
      toast.success(t("auth.verify.resendDone", { defaultValue: "Email resent, please check your inbox" }));
      startCooldown();
    } catch (err) {
      toast.error(getErrorMessage(err, t));
    } finally {
      setResending(false);
    }
  };

  const resendLabel = cooldown > 0
    ? t("auth.verify.resendCooldown", { seconds: cooldown, defaultValue: `Resend in ${cooldown}s` })
    : resending
      ? t("auth.verify.resending", { defaultValue: "Sending..." })
      : t("auth.verify.resend", { defaultValue: "Resend" });

  return (
    <AuthShell
      title={activation ? t("auth.verify.activationTitle", { defaultValue: "Check your inbox" }) : t("auth.verify.title", { defaultValue: "Verify your email" })}
      description={activation ? t("auth.verify.activationDescription", { defaultValue: "If this account needs activation, a link has been sent to your email. If already activated, you can sign in directly." }) : t("auth.verify.description", { defaultValue: "We sent a one-time verification link to your email address." })}
      footer={<Link to="/login" className="font-medium text-primary hover:underline">{t("auth.verify.back", { defaultValue: "Back to sign in" })}</Link>}
    >
      <div className="flex flex-col items-center gap-4 py-5 text-center">
        <div className="flex h-12 w-12 items-center justify-center rounded-md bg-primary/10 text-primary"><MailCheck className="h-6 w-6" /></div>
        {email ? <p className="break-all text-sm font-medium text-foreground">{email}</p> : null}
        <p className="text-xs text-muted-foreground">{t("auth.verify.resendHint", { defaultValue: "If you don't receive an email within a few minutes, the address may already be registered, or try signing in." })}</p>
        {email ? (
          <Button type="button" variant="outline" size="sm" onClick={() => void resend()} disabled={cooldown > 0 || resending}>
            {resendLabel}
          </Button>
        ) : null}
      </div>
    </AuthShell>
  );
}
