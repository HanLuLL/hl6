import { useState } from "react";
import type { FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { AuthShell } from "@/components/auth/auth-shell";
import { api, getErrorMessage } from "@/lib/api";

export default function ForgotPasswordPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      await api.requestPasswordReset({ email });
      navigate(`/verify-email?mode=reset&email=${encodeURIComponent(email)}`, { replace: true });
    } catch (err) {
      setError(getErrorMessage(err, t));
    } finally {
      setSubmitting(false);
    }
  };
  return (
    <AuthShell title={t("auth.forgot.title", { defaultValue: "Reset your password" })} description={t("auth.forgot.description", { defaultValue: "Enter your email and we will send a password reset link." })} footer={<Link to="/login" className="font-medium text-primary hover:underline">{t("auth.verify.back", { defaultValue: "Back to sign in" })}</Link>}>
      <form className="space-y-4" onSubmit={submit} noValidate>
        <div className="space-y-2"><Label htmlFor="forgot-email">{t("auth.email", { defaultValue: "Email" })}</Label><Input id="forgot-email" type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} required /></div>
        {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
        <Button type="submit" className="w-full" disabled={submitting}>{submitting ? t("common.saving") : t("auth.forgot.continue", { defaultValue: "Send reset link" })}</Button>
      </form>
    </AuthShell>
  );
}
