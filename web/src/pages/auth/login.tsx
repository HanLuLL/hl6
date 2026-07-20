import { useState, useEffect } from "react";
import type { FormEvent } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { AuthShell } from "@/components/auth/auth-shell";
import { PasswordField } from "@/components/auth/password-field";
import { api, getErrorMessage, ApiError } from "@/lib/api";
import { isNativeClient } from "@/lib/client-runtime";
import { signInNative } from "@/lib/native-client";

export default function LoginPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchParams] = useSearchParams();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [needsActivation, setNeedsActivation] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [kickedOut, setKickedOut] = useState(false);

  useEffect(() => {
    if (sessionStorage.getItem("hl6_kicked_out") === "1") {
      sessionStorage.removeItem("hl6_kicked_out");
      setKickedOut(true);
    }
  }, []);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setError("");
    setNeedsActivation(false);
    setSubmitting(true);
    try {
      const result = isNativeClient ? await signInNative(email, password) : (await api.login({ email, password })).data;

      if (result.banned) {
        navigate("/banned", { replace: true });
        return;
      }

      // Update React Query cache with user data
      queryClient.setQueryData(["me"], { code: 0, message: "", data: { user: result.user, credits: 0 } });

      // Browser mode: Cookie is set by server. Redirect to dashboard
      // where the page reload will automatically send the cookie
      if (!isNativeClient) {
        // Force full page reload to ensure cookie is sent with subsequent requests
        window.location.replace("/dashboard");
        return;
      }

      // Native mode: access_token is available and stored
      navigate("/dashboard", { replace: true });
    } catch (err) {
      if (err instanceof ApiError && err.messageKey === "error.accountActivationRequired") {
        setNeedsActivation(true);
      }
      setError(getErrorMessage(err, t));
    } finally {
      setSubmitting(false);
    }
  };

  const referral = searchParams.get("ref");
  return (
    <AuthShell
      title={t("auth.login.title", { defaultValue: "Welcome back" })}
      description={t("auth.login.description", { defaultValue: "Sign in to continue to your dashboard." })}
      footer={
        <span>
          {t("auth.login.newHere", { defaultValue: "New here?" })} {" "}
          <Link to={referral ? `/register?ref=${encodeURIComponent(referral)}` : "/register"} className="font-medium text-primary hover:underline">
            {t("auth.login.createAccount", { defaultValue: "Create an account" })}
          </Link>
        </span>
      }
    >
      <form className="space-y-4" onSubmit={submit} noValidate>
        <div className="space-y-2">
          <Label htmlFor="login-email">{t("auth.email", { defaultValue: "Email" })}</Label>
          <Input id="login-email" type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} required />
        </div>
        <div className="space-y-2">
          <div className="flex items-center justify-between gap-3">
            <Label htmlFor="login-password">{t("auth.password", { defaultValue: "Password" })}</Label>
            <Link to="/forgot-password" className="text-xs font-medium text-primary hover:underline">
              {t("auth.login.forgotPassword", { defaultValue: "Forgot password?" })}
            </Link>
          </div>
          <PasswordField id="login-password" autoComplete="current-password" value={password} onChange={(event) => setPassword(event.target.value)} required />
        </div>
        {kickedOut ? <p role="alert" className="text-sm text-amber-600 dark:text-amber-500">{t("auth.login.kickedOut", { defaultValue: "您的账号已在其他设备登录，请重新登录" })}</p> : null}
        {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
        {needsActivation ? (
          <p className="text-sm text-muted-foreground">
            <Link to="/activate-account" className="font-medium text-primary hover:underline">
              {t("auth.login.activate", { defaultValue: "Activate an existing account" })}
            </Link>
          </p>
        ) : null}
        <Button type="submit" className="w-full" disabled={submitting}>
          {submitting ? t("common.signingIn") : t("common.signIn")}
        </Button>
      </form>
      <div className="mt-5 text-center text-sm">
        <Link to="/activate-account" className="text-muted-foreground hover:text-foreground hover:underline">
          {t("auth.login.activate", { defaultValue: "Activate an existing account" })}
        </Link>
      </div>
    </AuthShell>
  );
}
