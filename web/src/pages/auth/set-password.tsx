import { useState } from "react";
import type { FormEvent } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { AuthShell } from "@/components/auth/auth-shell";
import { PasswordField } from "@/components/auth/password-field";
import { api, getErrorMessage } from "@/lib/api";
import { isNativeClient } from "@/lib/client-runtime";
import { setNativeAccessToken } from "@/lib/client-runtime";
import { SecureStoragePlugin } from "capacitor-secure-storage-plugin";

type SetPasswordPageProps = { reset?: boolean };

const PASSWORD_MIN_LENGTH = 8;
const PASSWORD_MAX_LENGTH = 128;

export default function SetPasswordPage({ reset = false }: SetPasswordPageProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchParams] = useSearchParams();
  const token = searchParams.get("token")?.trim() ?? "";
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    const passwordLength = Array.from(password).length;
    if (passwordLength < PASSWORD_MIN_LENGTH || passwordLength > PASSWORD_MAX_LENGTH) {
      setError(t("auth.setPassword.description", { defaultValue: "Choose a password with 8 to 128 characters." }));
      return;
    }
    if (password !== confirmation) {
      setError(t("auth.passwordMismatch", { defaultValue: "Passwords do not match." }));
      return;
    }
    setError("");
    setSubmitting(true);
    try {
      const result = await api.completePassword({ token, password });
      const session = result.data;
      const accessToken = session.access_token?.trim();
      if (isNativeClient && accessToken) {
        setNativeAccessToken(accessToken);
        await SecureStoragePlugin.set({ key: "hl6_native_session", value: accessToken });
      }
      // 直接用响应数据更新 React Query 缓存，使 isAuthenticated 立即变为 true
      queryClient.setQueryData(["me"], { code: 0, message: "", data: { user: session.user, credits: 0 } });
      // 清除 URL 中的 token（安全措施，仅在成功后执行）
      window.history.replaceState(null, "", window.location.pathname);
      navigate(session.banned ? "/banned" : "/dashboard", { replace: true });
    } catch (err) {
      setError(getErrorMessage(err, t));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <AuthShell
      title={reset ? t("auth.reset.title", { defaultValue: "Set a new password" }) : t("auth.setPassword.title", { defaultValue: "Set your password" })}
      description={t("auth.setPassword.description", { defaultValue: "Choose a password with 8 to 128 characters." })}
    >
      <form className="space-y-4" onSubmit={submit} noValidate>
        <div className="space-y-2">
          <Label htmlFor="new-password">{t("auth.password", { defaultValue: "Password" })}</Label>
          <PasswordField id="new-password" autoComplete="new-password" value={password} onChange={(event) => setPassword(event.target.value)} minLength={PASSWORD_MIN_LENGTH} required />
        </div>
        <div className="space-y-2">
          <Label htmlFor="confirm-password">{t("auth.confirmPassword", { defaultValue: "Confirm password" })}</Label>
          <PasswordField id="confirm-password" autoComplete="new-password" value={confirmation} onChange={(event) => setConfirmation(event.target.value)} required />
        </div>
        {!token ? <p role="alert" className="text-sm text-destructive">{t("auth.invalidLink", { defaultValue: "This link is invalid or has expired." })}</p> : null}
        {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
        <Button type="submit" className="w-full" disabled={!token || submitting}>{submitting ? t("common.saving") : t("auth.setPassword.continue", { defaultValue: "Continue" })}</Button>
      </form>
    </AuthShell>
  );
}
