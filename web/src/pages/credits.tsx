import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { api, getErrorMessage } from "@/lib/api";
import { useCredits, useDailyCheckinStatus, useTransactions } from "@/hooks/use-credits";
import { useReferrals } from "@/hooks/use-referrals";
import { toast } from "sonner";
import { useDocumentTitle } from "@/hooks/use-document-title";

export default function CreditsPage() {
  const queryClient = useQueryClient();
  const { data: creditData, isLoading: creditLoading } = useCredits();
  const { data: checkinStatus } = useDailyCheckinStatus();
  const [page, setPage] = useState(1);
  const { data: txnData, isLoading: txnLoading } = useTransactions(page, 10);
  const [refPage, setRefPage] = useState(1);
  const { data: refData, isLoading: refLoading } = useReferrals(refPage, 10);
  const { t } = useTranslation();
  useDocumentTitle(t("credits.title"));
  const claimMutation = useMutation({
    mutationFn: api.claimDailyCheckin,
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: ["credits"] });
      queryClient.invalidateQueries({ queryKey: ["transactions"] });
      queryClient.invalidateQueries({ queryKey: ["me"] });
      queryClient.invalidateQueries({ queryKey: ["daily-checkin-status"] });
      toast.success(t("credits.dailyCheckinSuccess", { amount: res.data.granted }));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const typeBadge = (type: string) => {
    switch (type) {
      case "grant": return <Badge className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">{t("credits.grant")}</Badge>;
      case "deduct": return <Badge className="bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200">{t("credits.deduct")}</Badge>;
      case "refund": return <Badge className="bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200">{t("credits.refund")}</Badge>;
      default: return <Badge variant="outline">{type}</Badge>;
    }
  };

  const referralEnabled = refData?.referral_enabled ?? false;
  const referralCode = refData?.referral_code ?? "";
  const referralLink = referralCode ? `${window.location.origin}/?ref=${referralCode}` : "";

  const copyLink = () => {
    navigator.clipboard.writeText(referralLink);
    toast.success(t("common.copied"));
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("credits.title")}</h1>
        <p className="text-muted-foreground">{t("credits.subtitle")}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium text-muted-foreground">{t("credits.currentBalance")}</CardTitle>
        </CardHeader>
        <CardContent>
          {creditLoading ? (
            <Skeleton className="h-10 w-20" />
          ) : (
            <div className="text-4xl font-bold">{creditData?.balance ?? 0}</div>
          )}
          <p className="text-sm text-muted-foreground mt-1">{t("credits.creditsAvailable")}</p>
        </CardContent>
      </Card>

      {checkinStatus?.enabled && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">{t("credits.dailyCheckinTitle")}</CardTitle>
            <p className="text-sm text-muted-foreground">{t("credits.dailyCheckinDesc")}</p>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-sm text-muted-foreground">
              {t("credits.dailyCheckinReward", { amount: checkinStatus.reward })}
            </p>
            <Button
              onClick={() => claimMutation.mutate()}
              disabled={checkinStatus.claimed_today || claimMutation.isPending}
            >
              {checkinStatus.claimed_today
                ? t("credits.dailyCheckinClaimed")
                : claimMutation.isPending
                  ? t("credits.dailyCheckingIn")
                  : t("credits.dailyCheckinNow")}
            </Button>
          </CardContent>
        </Card>
      )}

      {referralEnabled && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">{t("referral.title")}</CardTitle>
            <p className="text-sm text-muted-foreground">{t("referral.subtitle")}</p>
          </CardHeader>
          <CardContent className="space-y-4">
            {refLoading ? (
              <Skeleton className="h-9 w-full" />
            ) : (
              <div className="flex items-center gap-2">
                <code className="flex-1 rounded-md border bg-muted px-3 py-2 text-sm truncate">
                  {referralLink}
                </code>
                <Button variant="outline" size="sm" onClick={copyLink}>
                  {t("referral.copy")}
                </Button>
              </div>
            )}

            {!refLoading && refData && refData.records.length > 0 && (
              <div className="space-y-3">
                <h4 className="text-sm font-medium text-muted-foreground">{t("referral.records")}</h4>
                {refData.records.map((r) => (
                  <div key={r.id} className="flex items-center justify-between py-2 border-b last:border-0">
                    <div>
                      <p className="text-sm font-medium">{r.invitee_name}</p>
                      <p className="text-xs text-muted-foreground">
                        {new Date(r.invitee_created_at).toLocaleDateString()}
                      </p>
                    </div>
                    <div className="text-right">
                      {r.inviter_credits > 0 && (
                        <p className="text-sm font-medium text-green-600">+{r.inviter_credits}</p>
                      )}
                      <p className="text-xs text-muted-foreground">
                        {new Date(r.created_at).toLocaleDateString()}
                      </p>
                    </div>
                  </div>
                ))}
                {refData.total > 10 && (
                  <div className="flex justify-center gap-2 pt-2">
                    <Button variant="outline" size="sm" disabled={refPage <= 1} onClick={() => setRefPage((p) => p - 1)}>
                      {t("common.previous")}
                    </Button>
                    <span className="flex items-center text-sm text-muted-foreground">
                      {t("common.pageOf", { page: refPage, total: Math.ceil(refData.total / 10) })}
                    </span>
                    <Button variant="outline" size="sm" disabled={refPage >= Math.ceil(refData.total / 10)} onClick={() => setRefPage((p) => p + 1)}>
                      {t("common.next")}
                    </Button>
                  </div>
                )}
              </div>
            )}

            {!refLoading && refData && refData.records.length === 0 && (
              <p className="text-sm text-muted-foreground">{t("referral.noRecords")}</p>
            )}
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">{t("credits.transactionHistory")}</CardTitle>
        </CardHeader>
        <CardContent>
          {txnLoading ? (
            <div className="space-y-3">
              {[...Array(5)].map((_, i) => (
                <div key={i} className="flex items-center justify-between py-2 border-b last:border-0">
                  <div className="flex items-center gap-3">
                    <Skeleton className="h-5 w-14 rounded-full" />
                    <div className="space-y-1">
                      <Skeleton className="h-4 w-40" />
                      <Skeleton className="h-3 w-28" />
                    </div>
                  </div>
                  <div className="text-right space-y-1">
                    <Skeleton className="h-4 w-10 ml-auto" />
                    <Skeleton className="h-3 w-16 ml-auto" />
                  </div>
                </div>
              ))}
            </div>
          ) : !txnData?.data || txnData.data.length === 0 ? (
            <p className="text-center text-muted-foreground py-8">{t("credits.noTransactions")}</p>
          ) : (
            <div className="space-y-3">
              {txnData.data.map((tx) => (
                <div key={tx.id} className="flex items-center justify-between py-2 border-b last:border-0">
                  <div className="flex items-center gap-3">
                    {typeBadge(tx.type)}
                    <div>
                      <p className="text-sm font-medium">{t(tx.description_key, tx.description_params ?? {})}</p>
                      <p className="text-xs text-muted-foreground">
                        {new Date(tx.created_at).toLocaleString()}
                      </p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className={`font-medium ${tx.amount > 0 ? "text-green-600" : "text-red-600"}`}>
                      {tx.amount > 0 ? "+" : ""}{tx.amount}
                    </p>
                    <p className="text-xs text-muted-foreground">{t("credits.balance", { balance: tx.balance_after })}</p>
                  </div>
                </div>
              ))}
              {txnData.total > 10 && (
                <div className="flex justify-center gap-2 pt-4">
                  <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
                    {t("common.previous")}
                  </Button>
                  <span className="flex items-center text-sm text-muted-foreground">
                    {t("common.pageOf", { page, total: Math.ceil(txnData.total / 10) })}
                  </span>
                  <Button variant="outline" size="sm" disabled={page >= Math.ceil(txnData.total / 10)} onClick={() => setPage((p) => p + 1)}>
                    {t("common.next")}
                  </Button>
                </div>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
