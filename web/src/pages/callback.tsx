import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";

export default function CallbackPage() {
  const navigate = useNavigate();
  const { t } = useTranslation();

  useEffect(() => {
    // Backend handles callback and redirects to /dashboard.
    // This page is a fallback if user navigates here directly.
    navigate("/dashboard", { replace: true });
  }, [navigate]);

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto" />
        <p className="mt-4 text-muted-foreground">{t("common.signingIn")}</p>
      </div>
    </div>
  );
}
