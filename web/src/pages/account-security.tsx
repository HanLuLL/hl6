import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import {
  BadgeCheck,
  Clock,
  Mail,
  ShieldCheck,
  AlertCircle,
  RefreshCw,
  CheckCircle2,
  XCircle,
} from "lucide-react";
import { api } from "@/lib/api";
import { useAuth } from "@/hooks/use-auth";
import { useDocumentTitle } from "@/hooks/use-document-title";
import {
  useRealnameStatus,
  useRealnameHistory,
  useSubmitRealname,
  useRetryRealname,
} from "@/hooks/use-realname";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// 身份证号简单校验：18位，最后一位可为 X
function isValidIDCard(value: string): boolean {
  return /^\d{17}[\dXx]$/.test(value.trim());
}

export default function AccountSecurityPage() {
  const { user } = useAuth();
  const { t } = useTranslation();
  useDocumentTitle(t("accountSecurity.title"));

  const { data: status, isLoading: statusLoading } = useRealnameStatus();
  const [page] = useState(1);
  const { data: historyData } = useRealnameHistory(page, 5);
  const submitMutation = useSubmitRealname();
  const retryMutation = useRetryRealname();

  // 支付方式列表（仅在需要付费时使用）
  const { data: paymentMethodsData } = useQuery({
    queryKey: ["payment-methods"],
    queryFn: async () => (await api.getPaymentMethods()).data,
    staleTime: 60_000,
    enabled: !!status?.latest_application || true,
  });

  const [realName, setRealName] = useState("");
  const [idCard, setIDCard] = useState("");
  const [verificationType, setVerificationType] = useState<"idcard" | "face">("idcard");
  const [gateway, setGateway] = useState<string>("");
  const [paymentMethod, setPaymentMethod] = useState<string>("");

  const availableMethods = paymentMethodsData?.methods ?? [];
  useEffect(() => {
    if (availableMethods.length > 0 && !gateway) {
      setGateway(availableMethods[0].gateway);
      setPaymentMethod(availableMethods[0].method);
    }
  }, [availableMethods, gateway]);

  const currentStatus = status?.status ?? "unverified";
  const needPay = submitMutation.data?.data?.need_pay;

  const handleSubmit = () => {
    if (!realName.trim() || !idCard.trim()) return;
    if (!isValidIDCard(idCard)) {
      return;
    }
    submitMutation.mutate({
      real_name: realName.trim(),
      id_card: idCard.trim().toUpperCase(),
      verification_type: verificationType,
      gateway: gateway || undefined,
      payment_method: paymentMethod || undefined,
    });
  };

  const statusBadge = (s: string) => {
    switch (s) {
      case "verified":
        return <Badge variant="default" className="bg-green-600 hover:bg-green-600"><CheckCircle2 className="h-3 w-3 mr-1" />{t("realname.status.verified")}</Badge>;
      case "pending":
      case "pending_payment":
      case "paid":
      case "verifying":
        return <Badge variant="secondary"><Clock className="h-3 w-3 mr-1" />{t("realname.status.pending")}</Badge>;
      case "rejected":
      case "failed":
        return <Badge variant="destructive"><XCircle className="h-3 w-3 mr-1" />{t("realname.status.rejected")}</Badge>;
      default:
        return <Badge variant="outline">{t("realname.status.unverified")}</Badge>;
    }
  };

  const isFormDisabled =
    currentStatus === "verified" ||
    currentStatus === "pending" ||
    submitMutation.isPending;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("accountSecurity.title")}</h1>
        <p className="text-muted-foreground">{t("accountSecurity.subtitle")}</p>
      </div>

      {/* 邮箱绑定信息 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Mail className="h-4 w-4" />
            {t("accountSecurity.email.title")}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center justify-between gap-4">
            <div className="min-w-0">
              <p className="text-sm font-medium truncate">{user?.email}</p>
              <p className="text-xs text-muted-foreground">{t("accountSecurity.email.current")}</p>
            </div>
            <BadgeCheck className="h-5 w-5 text-green-600 shrink-0" />
          </div>
          <p className="text-xs text-muted-foreground">{t("accountSecurity.email.hint")}</p>
        </CardContent>
      </Card>

      {/* 实名认证 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <ShieldCheck className="h-4 w-4" />
            {t("accountSecurity.realname.title")}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {statusLoading ? (
            <Skeleton className="h-20 w-full" />
          ) : (
            <>
              {/* 当前状态 */}
              <div className="flex items-center justify-between gap-4 rounded-md border p-3">
                <div className="min-w-0">
                  <p className="text-sm font-medium">{t("accountSecurity.realname.currentStatus")}</p>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    {currentStatus === "verified" && status?.realname_name
                      ? t("accountSecurity.realname.verifiedAs", { name: status.realname_name })
                      : t("accountSecurity.realname.statusHint")}
                  </p>
                  {status?.verified_at && (
                    <p className="text-xs text-muted-foreground">
                      {t("accountSecurity.realname.verifiedAt")}: {new Date(status.verified_at).toLocaleString()}
                    </p>
                  )}
                </div>
                <div className="shrink-0">{statusBadge(currentStatus)}</div>
              </div>

              {/* 被拒绝时显示原因 */}
              {currentStatus === "rejected" && status?.latest_application?.reject_reason && (
                <div className="flex items-start gap-2 rounded-md border border-destructive/50 bg-destructive/5 p-3 text-sm">
                  <AlertCircle className="h-4 w-4 mt-0.5 text-destructive shrink-0" />
                  <div>
                    <p className="font-medium text-destructive">{t("accountSecurity.realname.rejectReason")}</p>
                    <p className="text-muted-foreground mt-1">{status.latest_application.reject_reason}</p>
                  </div>
                </div>
              )}

              {/* 等待中：显示重试按钮 */}
              {(currentStatus === "pending" || currentStatus === "rejected") && (
                <div className="flex justify-end">
                  <Button variant="outline" size="sm" onClick={() => retryMutation.mutate()} disabled={retryMutation.isPending}>
                    <RefreshCw className="h-4 w-4" />
                    {retryMutation.isPending ? t("common.processing") : t("accountSecurity.realname.retry")}
                  </Button>
                </div>
              )}

              {/* 表单：仅在未认证/被拒绝时显示 */}
              {(currentStatus === "unverified" || currentStatus === "rejected") && (
                <div className="space-y-4 border-t pt-4">
                  <div className="space-y-2">
                    <Label>{t("accountSecurity.realname.realName")}</Label>
                    <Input
                      value={realName}
                      onChange={(e) => setRealName(e.target.value)}
                      placeholder={t("accountSecurity.realname.realNamePlaceholder")}
                      disabled={isFormDisabled}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label>{t("accountSecurity.realname.idCard")}</Label>
                    <Input
                      value={idCard}
                      onChange={(e) => setIDCard(e.target.value)}
                      placeholder={t("accountSecurity.realname.idCardPlaceholder")}
                      disabled={isFormDisabled}
                      maxLength={18}
                    />
                    {idCard && !isValidIDCard(idCard) && (
                      <p className="text-xs text-destructive">{t("accountSecurity.realname.idCardInvalid")}</p>
                    )}
                  </div>
                  <div className="space-y-2">
                    <Label>{t("accountSecurity.realname.verificationType")}</Label>
                    <Select value={verificationType} onValueChange={(v) => setVerificationType(v as "idcard" | "face")} disabled={isFormDisabled}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="idcard">{t("accountSecurity.realname.typeIDCard")}</SelectItem>
                        <SelectItem value="face">{t("accountSecurity.realname.typeFace")}</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  {/* 支付方式（仅在需要付费时显示） */}
                  {availableMethods.length > 0 && (
                    <div className="space-y-2">
                      <Label>{t("accountSecurity.realname.paymentMethod")}</Label>
                      <Select
                        value={`${gateway}:${paymentMethod}`}
                        onValueChange={(v) => {
                          const [g, m] = v.split(":");
                          setGateway(g);
                          setPaymentMethod(m);
                        }}
                        disabled={isFormDisabled}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {availableMethods.map((m) => (
                            <SelectItem key={`${m.gateway}:${m.method}`} value={`${m.gateway}:${m.method}`}>
                              {m.gateway === "epay" ? t("credits.epay") : t("credits.codepay")} · {m.method === "alipay" ? t("credits.alipay") : m.method === "wechat" ? t("credits.wechat") : "QQ"}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      <p className="text-xs text-muted-foreground">{t("accountSecurity.realname.paymentHint")}</p>
                    </div>
                  )}

                  <div className="flex items-center justify-between gap-4">
                    <p className="text-xs text-muted-foreground">{t("accountSecurity.realname.privacyNotice")}</p>
                    <Button onClick={handleSubmit} disabled={isFormDisabled || !realName.trim() || !isValidIDCard(idCard)}>
                      {submitMutation.isPending ? t("common.processing") : t("accountSecurity.realname.submit")}
                    </Button>
                  </div>

                  {needPay && (
                    <p className="text-xs text-muted-foreground">{t("accountSecurity.realname.payRedirectHint")}</p>
                  )}
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>

      {/* 历史申请 */}
      {historyData && historyData.data && historyData.data.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{t("accountSecurity.realname.history")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {historyData.data.map((app) => (
                <div key={app.id} className="flex items-center justify-between gap-4 rounded-md border p-3 text-sm">
                  <div className="min-w-0">
                    <p className="font-medium truncate">
                      {app.real_name_masked} · {app.id_card_masked}
                    </p>
                    <p className="text-xs text-muted-foreground mt-0.5">
                      {new Date(app.created_at).toLocaleString()}
                    </p>
                    {app.reject_reason && (
                      <p className="text-xs text-destructive mt-1 truncate">{app.reject_reason}</p>
                    )}
                  </div>
                  <div className="shrink-0">{statusBadge(app.status)}</div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
