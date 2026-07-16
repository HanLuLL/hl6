import { Link, useSearchParams } from "react-router-dom";
import { MailCheck } from "lucide-react";
import { useTranslation } from "react-i18next";
import { AuthShell } from "@/components/auth/auth-shell";

export default function VerifyEmailPage() {
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const email = searchParams.get("email") ?? "";
  const activation = searchParams.get("mode") === "activation";
  return (
    <AuthShell
      title={activation ? t("auth.verify.activationTitle", { defaultValue: "Check your inbox" }) : t("auth.verify.title", { defaultValue: "Verify your email" })}
      description={activation ? t("auth.verify.activationDescription", { defaultValue: "We sent an activation link to your email address." }) : t("auth.verify.description", { defaultValue: "We sent a one-time verification link to your email address." })}
      footer={<Link to="/login" className="font-medium text-primary hover:underline">{t("auth.verify.back", { defaultValue: "Back to sign in" })}</Link>}
    >
      <div className="flex flex-col items-center gap-4 py-5 text-center">
        <div className="flex h-12 w-12 items-center justify-center rounded-md bg-primary/10 text-primary"><MailCheck className="h-6 w-6" /></div>
        {email ? <p className="break-all text-sm font-medium text-foreground">{email}</p> : null}
      </div>
    </AuthShell>
  );
}
