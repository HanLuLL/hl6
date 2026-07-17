import { useState } from "react";
import type { FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { AuthShell } from "@/components/auth/auth-shell";
import { api, getErrorMessage } from "@/lib/api";

export default function ActivateAccountPage() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      await api.requestActivation({ email, locale: i18n.resolvedLanguage ?? i18n.language });
      navigate(`/verify-email?mode=activation&email=${encodeURIComponent(email)}`, { replace: true });
    } catch (err) {
      setError(getErrorMessage(err, t));
    } finally {
      setSubmitting(false);
    }
  };
  return (
    <AuthShell title={t("auth.activate.title", { defaultValue: "Activate your account" })} description={t("auth.activate.description", { defaultValue: "Use the email address associated with your existing account." })} footer={<Link to="/login" className="font-medium text-primary hover:underline">{t("auth.verify.back", { defaultValue: "Back to sign in" })}</Link>}>
      <form className="space-y-4" onSubmit={submit} noValidate>
        <div className="space-y-2"><Label htmlFor="activate-email">{t("auth.email", { defaultValue: "Email" })}</Label><Input id="activate-email" type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} required /></div>
        {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
        <Button type="submit" className="w-full" disabled={submitting}>{submitting ? t("common.saving") : t("auth.activate.continue", { defaultValue: "Send activation link" })}</Button>
      </form>
    </AuthShell>
  );
}
