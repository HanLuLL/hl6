import { useState } from "react";
import type { FormEvent } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { AuthShell } from "@/components/auth/auth-shell";
import { api, getErrorMessage } from "@/lib/api";

export default function RegisterPage() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [email, setEmail] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      await api.requestRegistration({ email, referral_code: searchParams.get("ref") ?? undefined, locale: i18n.resolvedLanguage ?? i18n.language });
      navigate(`/verify-email?email=${encodeURIComponent(email)}`, { replace: true });
    } catch (err) {
      setError(getErrorMessage(err, t));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <AuthShell
      title={t("auth.register.title", { defaultValue: "Create your account" })}
      description={t("auth.register.description", { defaultValue: "We will send a link to verify your email before you choose a password." })}
      footer={<Link to="/login" className="font-medium text-primary hover:underline">{t("auth.register.signIn", { defaultValue: "Already have an account? Sign in" })}</Link>}
    >
      <form className="space-y-4" onSubmit={submit} noValidate>
        <div className="space-y-2">
          <Label htmlFor="register-email">{t("auth.email", { defaultValue: "Email" })}</Label>
          <Input id="register-email" type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} required />
        </div>
        {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
        <Button type="submit" className="w-full" disabled={submitting}>{submitting ? t("common.saving") : t("auth.register.continue", { defaultValue: "Continue" })}</Button>
      </form>
    </AuthShell>
  );
}
