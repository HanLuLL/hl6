import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  CheckCircle2,
  XCircle,
  RefreshCw,
} from "lucide-react";
import { useDocumentTitle } from "@/hooks/use-document-title";
import {
  useAdminRealnameApplications,
  useAdminRealnameStats,
  useAdminReviewRealname,
  useAdminRetryRealname,
} from "@/hooks/use-realname";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { Paginator } from "@/components/ui/paginator";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import type { RealnameApplication } from "@/types";

const PER_PAGE = 10;

const statusOptions = [
  { value: "", labelKey: "realname.admin.statusAll" },
  { value: "pending_payment", labelKey: "realname.status.pendingPayment" },
  { value: "paid", labelKey: "realname.status.paid" },
  { value: "verifying", labelKey: "realname.status.verifying" },
  { value: "verified", labelKey: "realname.status.verified" },
  { value: "rejected", labelKey: "realname.status.rejected" },
  { value: "failed", labelKey: "realname.status.failed" },
];

const providerLabel = (p: string, t: (k: string) => string) => {
  if (p === "aliyun") return t("adminSettings.realname.providerAliyun");
  if (p === "juhe") return t("adminSettings.realname.providerJuhe");
  return t("adminSettings.realname.providerManual");
};

const statusBadge = (s: string, t: (k: string) => string) => {
  switch (s) {
    case "verified":
      return <Badge variant="default" className="bg-green-600 hover:bg-green-600"><CheckCircle2 className="h-3 w-3 mr-1" />{t("realname.status.verified")}</Badge>;
    case "pending_payment":
    case "paid":
    case "verifying":
      return <Badge variant="secondary">{t(`realname.status.${s === "pending_payment" ? "pendingPayment" : s === "paid" ? "paid" : "verifying"}`)}</Badge>;
    case "rejected":
    case "failed":
      return <Badge variant="destructive"><XCircle className="h-3 w-3 mr-1" />{t(`realname.status.${s}`)}</Badge>;
    default:
      return <Badge variant="outline">{s}</Badge>;
  }
};

export default function AdminRealnamePage() {
  const { t } = useTranslation();
  useDocumentTitle(t("realname.admin.title"));

  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState("");
  const [providerFilter, setProviderFilter] = useState("");
  const [userIdFilter, setUserIdFilter] = useState("");
  const [reviewOpen, setReviewOpen] = useState(false);
  const [reviewTarget, setReviewTarget] = useState<RealnameApplication | null>(null);
  const [reviewApproved, setReviewApproved] = useState(true);
  const [reviewReason, setReviewReason] = useState("");

  const { data: stats, isLoading: statsLoading } = useAdminRealnameStats();
  const { data: listData, isLoading: listLoading } = useAdminRealnameApplications({
    page,
    per_page: PER_PAGE,
    status: statusFilter || undefined,
    provider: providerFilter || undefined,
    user_id: userIdFilter ? parseInt(userIdFilter, 10) || undefined : undefined,
  });

  const reviewMutation = useAdminReviewRealname();
  const retryMutation = useAdminRetryRealname();

  const applications = listData?.data ?? [];
  const total = listData?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PER_PAGE));

  const openReview = (app: RealnameApplication, approved: boolean) => {
    setReviewTarget(app);
    setReviewApproved(approved);
    setReviewReason("");
    setReviewOpen(true);
  };

  const handleReviewSubmit = () => {
    if (!reviewTarget) return;
    reviewMutation.mutate(
      { id: reviewTarget.id, approved: reviewApproved, reason: reviewReason },
      {
        onSettled: () => {
          setReviewOpen(false);
          setReviewTarget(null);
        },
      },
    );
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("realname.admin.title")}</h1>
        <p className="text-muted-foreground">{t("realname.admin.subtitle")}</p>
      </div>

      {/* 统计卡片 */}
      {statsLoading ? (
        <Skeleton className="h-24 w-full" />
      ) : (
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardHeader className="pb-2"><CardTitle className="text-sm font-medium text-muted-foreground">{t("realname.admin.statsVerified")}</CardTitle></CardHeader>
            <CardContent><p className="text-2xl font-bold">{stats?.verified_users ?? 0}</p></CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2"><CardTitle className="text-sm font-medium text-muted-foreground">{t("realname.status.verifying")}</CardTitle></CardHeader>
            <CardContent><p className="text-2xl font-bold">{stats?.status_counts?.verifying ?? 0}</p></CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2"><CardTitle className="text-sm font-medium text-muted-foreground">{t("realname.status.pendingPayment")}</CardTitle></CardHeader>
            <CardContent><p className="text-2xl font-bold">{(stats?.status_counts?.pending_payment ?? 0) + (stats?.status_counts?.paid ?? 0)}</p></CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2"><CardTitle className="text-sm font-medium text-muted-foreground">{t("realname.status.rejected")}</CardTitle></CardHeader>
            <CardContent><p className="text-2xl font-bold">{(stats?.status_counts?.rejected ?? 0) + (stats?.status_counts?.failed ?? 0)}</p></CardContent>
          </Card>
        </div>
      )}

      {/* 筛选 */}
      <Card>
        <CardHeader><CardTitle className="text-base">{t("realname.admin.filters")}</CardTitle></CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-3">
            <div className="space-y-2">
              <Label>{t("realname.admin.filterStatus")}</Label>
              <Select value={statusFilter} onValueChange={(v) => { setStatusFilter(v); setPage(1); }}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {statusOptions.map((opt) => (
                    <SelectItem key={opt.value || "all"} value={opt.value || "all"}>
                      {t(opt.labelKey)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>{t("realname.admin.filterProvider")}</Label>
              <Select
                value={providerFilter}
                onValueChange={(v) => { setProviderFilter(v === "all" ? "" : v); setPage(1); }}
              >
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">{t("realname.admin.statusAll")}</SelectItem>
                  <SelectItem value="manual">{t("adminSettings.realname.providerManual")}</SelectItem>
                  <SelectItem value="aliyun">{t("adminSettings.realname.providerAliyun")}</SelectItem>
                  <SelectItem value="juhe">{t("adminSettings.realname.providerJuhe")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>{t("realname.admin.filterUserId")}</Label>
              <Input
                value={userIdFilter}
                onChange={(e) => setUserIdFilter(e.target.value)}
                placeholder={t("realname.admin.filterUserIdPlaceholder")}
                type="number"
                min="1"
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 申请列表 */}
      <Card>
        <CardHeader><CardTitle className="text-base">{t("realname.admin.applications")}</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          {listLoading ? (
            <Skeleton className="h-40 w-full" />
          ) : applications.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-8">{t("realname.admin.empty")}</p>
          ) : (
            applications.map((app) => (
              <div key={app.id} className="rounded-md border p-4 space-y-3">
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0 space-y-1">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="font-medium">#{app.id}</span>
                      {statusBadge(app.status, t)}
                      <Badge variant="outline">{providerLabel(app.provider, t)}</Badge>
                      {app.verification_type === "face" && (
                        <Badge variant="outline">{t("accountSecurity.realname.typeFace")}</Badge>
                      )}
                    </div>
                    <p className="text-sm text-muted-foreground">
                      {app.real_name_masked} · {app.id_card_masked}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {app.user_email ?? `UID ${app.user_id}`} · {new Date(app.created_at).toLocaleString()}
                    </p>
                    {app.reject_reason && (
                      <p className="text-xs text-destructive mt-1">{app.reject_reason}</p>
                    )}
                    {app.verified_at && (
                      <p className="text-xs text-green-600 mt-1">
                        {t("realname.admin.verifiedAt")}: {new Date(app.verified_at).toLocaleString()}
                      </p>
                    )}
                  </div>
                  <div className="flex flex-col gap-2 shrink-0">
                    {/* 仅 manual 模式下且处于 verifying 状态可审核 */}
                    {app.provider === "manual" && app.status === "verifying" && (
                      <>
                        <Button size="sm" variant="default" className="bg-green-600 hover:bg-green-600" onClick={() => openReview(app, true)}>
                          <CheckCircle2 className="h-4 w-4" />
                          {t("realname.admin.approve")}
                        </Button>
                        <Button size="sm" variant="destructive" onClick={() => openReview(app, false)}>
                          <XCircle className="h-4 w-4" />
                          {t("realname.admin.reject")}
                        </Button>
                      </>
                    )}
                    {/* 失败状态可重试 */}
                    {(app.status === "failed" || app.status === "rejected") && (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => retryMutation.mutate(app.id)}
                        disabled={retryMutation.isPending}
                      >
                        <RefreshCw className="h-4 w-4" />
                        {t("realname.admin.retry")}
                      </Button>
                    )}
                  </div>
                </div>
              </div>
            ))
          )}
          <div className="flex justify-end pt-2">
            <Paginator page={page} totalPages={totalPages} onPageChange={setPage} />
          </div>
        </CardContent>
      </Card>

      {/* 审核对话框 */}
      <Dialog open={reviewOpen} onOpenChange={setReviewOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {reviewApproved ? t("realname.admin.approveTitle") : t("realname.admin.rejectTitle")}
            </DialogTitle>
            <DialogDescription>
              {reviewTarget && (
                <span>
                  #{reviewTarget.id} · {reviewTarget.real_name_masked} · {reviewTarget.id_card_masked}
                </span>
              )}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label>{t("realname.admin.reviewReason")}</Label>
            <Textarea
              value={reviewReason}
              onChange={(e) => setReviewReason(e.target.value)}
              rows={3}
              placeholder={reviewApproved ? t("realname.admin.reviewReasonApprovePlaceholder") : t("realname.admin.reviewReasonRejectPlaceholder")}
            />
            {!reviewApproved && (
              <p className="text-xs text-muted-foreground">{t("realname.admin.reviewReasonRejectHint")}</p>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setReviewOpen(false)} disabled={reviewMutation.isPending}>
              {t("common.cancel")}
            </Button>
            <Button
              variant={reviewApproved ? "default" : "destructive"}
              onClick={handleReviewSubmit}
              disabled={reviewMutation.isPending}
            >
              {reviewMutation.isPending ? t("common.processing") : reviewApproved ? t("realname.admin.confirmApprove") : t("realname.admin.confirmReject")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
