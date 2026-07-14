import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { useBanInfo, useCreateAppeal, useMyAppeals } from "@/hooks/use-ai-audit";
import { useAuth } from "@/hooks/use-auth";
import { ShieldX, MessageSquare, LogOut, Send } from "lucide-react";

export default function BannedPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { signOut } = useAuth();
  const { data: banInfo, isLoading } = useBanInfo();
  const { data: appeals } = useMyAppeals();
  const createAppeal = useCreateAppeal();
  const [appealContent, setAppealContent] = useState("");
  const [showAppealForm, setShowAppealForm] = useState(false);

  // 如果用户未被封禁，跳转回控制台
  if (!isLoading && banInfo && !banInfo.banned) {
    navigate("/dashboard", { replace: true });
    return null;
  }

  const hasPendingAppeal = appeals?.some((a) => a.status === "pending");

  const handleSubmitAppeal = () => {
    if (!appealContent.trim()) return;
    createAppeal.mutate(appealContent.trim(), {
      onSuccess: () => {
        setAppealContent("");
        setShowAppealForm(false);
      },
    });
  };

  const handleLogout = async () => {
    await signOut();
    navigate("/", { replace: true });
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <Card className="w-full max-w-lg">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 rounded-full bg-destructive/10 p-4">
            <ShieldX className="h-10 w-10 text-destructive" />
          </div>
          <CardTitle className="text-2xl">{t("banned.title")}</CardTitle>
          <p className="text-sm text-muted-foreground">{t("banned.subtitle")}</p>
        </CardHeader>
        <CardContent className="space-y-6">
          {isLoading ? null : (
            <>
              {/* 封禁原因 */}
              {banInfo && (
                <div className="rounded-lg border bg-muted/50 p-4">
                  <p className="text-sm font-medium text-muted-foreground mb-1">{t("banned.reason")}</p>
                  <p className="text-sm">{banInfo.reason || "-"}</p>
                  {banInfo.banned_at && (
                    <p className="text-xs text-muted-foreground mt-2">
                      {t("banned.bannedAt", { date: new Date(banInfo.banned_at).toLocaleString() })}
                    </p>
                  )}
                  <p className="text-xs text-muted-foreground mt-1">
                    {banInfo.banned_until
                      ? t("banned.expectedUnban", { date: new Date(banInfo.banned_until).toLocaleString(), defaultValue: `Expected unban time: ${new Date(banInfo.banned_until).toLocaleString()}` })
                      : t("banned.expectedUnbanManual", { defaultValue: "Expected unban time: manual review required." })}
                  </p>
                </div>
              )}

              {/* 申诉列表 */}
              {appeals && appeals.length > 0 && (
                <div className="space-y-2">
                  <p className="text-sm font-medium">{t("banned.appealHistory")}</p>
                  {appeals.map((appeal) => (
                    <div key={appeal.id} className="rounded-lg border p-3 text-sm">
                      <div className="flex items-center justify-between mb-1">
                        <span className="text-muted-foreground">
                          {new Date(appeal.created_at).toLocaleString()}
                        </span>
                        <span
                          className={`text-xs font-medium px-2 py-0.5 rounded-full ${
                            appeal.status === "pending"
                              ? "bg-yellow-100 text-yellow-800"
                              : appeal.status === "approved"
                              ? "bg-green-100 text-green-800"
                              : "bg-red-100 text-red-800"
                          }`}
                        >
                          {t(`banned.appealStatus.${appeal.status}`)}
                        </span>
                      </div>
                      <p className="text-muted-foreground">{appeal.content}</p>
                      {appeal.reply && (
                        <div className="mt-2 rounded bg-muted p-2 text-xs">
                          <p className="font-medium">{t("banned.adminReply")}：</p>
                          <p>{appeal.reply}</p>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}

              {/* 申诉表单 */}
              {!hasPendingAppeal && !showAppealForm && (
                <Button
                  variant="outline"
                  className="w-full"
                  onClick={() => setShowAppealForm(true)}
                >
                  <MessageSquare className="mr-2 h-4 w-4" />
                  {t("banned.submitAppeal")}
                </Button>
              )}

              {showAppealForm && (
                <div className="space-y-3">
                  <Textarea
                    value={appealContent}
                    onChange={(e) => setAppealContent(e.target.value)}
                    placeholder={t("banned.appealPlaceholder")}
                    rows={4}
                  />
                  <div className="flex gap-2">
                    <Button
                      onClick={handleSubmitAppeal}
                      disabled={createAppeal.isPending || !appealContent.trim()}
                      className="flex-1"
                    >
                      <Send className="mr-2 h-4 w-4" />
                      {createAppeal.isPending ? t("common.saving") : t("banned.submit")}
                    </Button>
                    <Button
                      variant="outline"
                      onClick={() => setShowAppealForm(false)}
                    >
                      {t("common.cancel")}
                    </Button>
                  </div>
                </div>
              )}

              {hasPendingAppeal && (
                <p className="text-center text-sm text-muted-foreground">
                  {t("banned.appealPending")}
                </p>
              )}

              {/* 安全登出 */}
              <Button
                variant="destructive"
                className="w-full"
                onClick={handleLogout}
              >
                <LogOut className="mr-2 h-4 w-4" />
                {t("banned.safeLogout")}
              </Button>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
