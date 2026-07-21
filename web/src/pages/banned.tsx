import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Skeleton } from "@/components/ui/skeleton";
import { useBanInfo, useCreateAppeal, useMyAppeals } from "@/hooks/use-ai-audit";
import { useAuth } from "@/hooks/use-auth";
import { ShieldX, MessageSquare, LogOut, Send, AlertCircle, RefreshCw } from "lucide-react";

export default function BannedPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { signOut } = useAuth();
  const { data: banInfo, isLoading: banInfoLoading, isError: banInfoError, refetch: refetchBanInfo } = useBanInfo();
  const { data: appeals, isLoading: appealsLoading, isError: appealsError } = useMyAppeals();
  const createAppeal = useCreateAppeal();
  const [appealContent, setAppealContent] = useState("");
  const [showAppealForm, setShowAppealForm] = useState(false);

  // 如果用户未被封禁，跳转回控制台（必须在 useEffect 中调用 navigate，不能在渲染期调用）
  useEffect(() => {
    if (!banInfoLoading && banInfo && !banInfo.banned) {
      navigate("/dashboard", { replace: true });
    }
  }, [banInfoLoading, banInfo, navigate]);

  const hasPendingAppeal = appeals?.some((a) => a.status === "pending") ?? false;

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
          {/* 加载中：显示骨架屏 */}
          {banInfoLoading && (
            <div className="space-y-3">
              <Skeleton className="h-20 w-full rounded-lg" />
            </div>
          )}

          {/* 加载失败：显示错误占位 + 重试按钮，避免页面卡死空白 */}
          {!banInfoLoading && banInfoError && (
            <div className="space-y-4">
              <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-4 text-center">
                <AlertCircle className="mx-auto mb-2 h-8 w-8 text-destructive" />
                <p className="text-sm font-medium">{t("banned.loadFailed", { defaultValue: "Failed to load ban information." })}</p>
                <p className="mt-1 text-xs text-muted-foreground">{t("banned.loadFailedHint", { defaultValue: "Please check your network and try again." })}</p>
              </div>
              <Button variant="outline" className="w-full" onClick={() => refetchBanInfo()}>
                <RefreshCw className="mr-2 h-4 w-4" />
                {t("common.retry", { defaultValue: "Retry" })}
              </Button>
            </div>
          )}

          {/* 正常加载：显示封禁原因 + 申诉区 */}
          {!banInfoLoading && !banInfoError && banInfo && (
            <>
              {/* 封禁原因 */}
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
                    ? t("banned.expectedUnban", { date: new Date(banInfo.banned_until).toLocaleString() })
                    : t("banned.expectedUnbanManual")}
                </p>
              </div>

              {/* 申诉列表加载失败：显示提示但不阻塞提交 */}
              {appealsError && (
                <div className="rounded-lg border border-yellow-500/30 bg-yellow-500/5 p-3 text-xs text-muted-foreground">
                  {t("banned.appealsLoadFailed", { defaultValue: "Failed to load appeal history. You can still submit a new appeal." })}
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

              {/* 申诉列表加载中 */}
              {appealsLoading && !appeals && (
                <div className="space-y-2">
                  <Skeleton className="h-16 w-full" />
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
