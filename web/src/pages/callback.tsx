import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useHandleSignInCallback } from "@logto/react";

export default function CallbackPage() {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { isLoading } = useHandleSignInCallback(() => {
    navigate("/dashboard", { replace: true });
  });

  useEffect(() => {
    if (!isLoading) {
      navigate("/dashboard", { replace: true });
    }
  }, [isLoading, navigate]);

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto" />
        <p className="mt-4 text-muted-foreground">{t("common.signingIn")}</p>
      </div>
    </div>
  );
}
