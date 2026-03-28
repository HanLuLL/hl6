import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useAuth } from "@/hooks/use-auth";
import { Button } from "@/components/ui/button";

export default function NotFoundPage() {
  const { isAuthenticated } = useAuth();
  const { t } = useTranslation();

  return (
    <div className="flex min-h-screen items-center justify-center px-4 py-10">
      <div className="w-full max-w-xl rounded-2xl border bg-card px-6 py-10 text-center shadow-sm sm:px-10">
        <p className="text-xs font-semibold tracking-[0.32em] text-muted-foreground">404</p>
        <h1 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">{t("notFound.title")}</h1>
        <p className="mt-4 text-muted-foreground">{t("notFound.description")}</p>
        <div className="mt-8">
          <Button asChild>
            <Link to={isAuthenticated ? "/dashboard" : "/"}>
              {isAuthenticated ? t("notFound.backDashboard") : t("notFound.backHome")}
            </Link>
          </Button>
        </div>
      </div>
    </div>
  );
}
