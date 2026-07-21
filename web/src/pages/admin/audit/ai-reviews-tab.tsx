import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  useAdminAIReviews,
  useAdminReviewAIReview,
} from "@/hooks/use-ai-audit";
import type { AuditAIReview } from "@/types/ai-audit";

function judgmentBadge(judgment: string, t: (key: string) => string) {
  switch (judgment) {
    case "clean":
      return <Badge className="bg-green-600 hover:bg-green-700">{t("aiAudit.judgmentClean")}</Badge>;
    case "violation":
      return <Badge variant="destructive">{t("aiAudit.judgmentViolation")}</Badge>;
    case "error":
      return <Badge variant="secondary">{t("aiAudit.judgmentError")}</Badge>;
    default:
      return <Badge variant="outline">{judgment}</Badge>;
  }
}

function reviewStatusBadge(status: string, t: (key: string) => string) {
  switch (status) {
    case "pending":
      return <Badge className="bg-yellow-600 hover:bg-yellow-700">{t("aiAudit.reviewStatusPending")}</Badge>;
    case "confirmed":
      return <Badge variant="destructive">{t("aiAudit.reviewStatusConfirmed")}</Badge>;
    case "overturned":
      return <Badge className="bg-green-600 hover:bg-green-700">{t("aiAudit.reviewStatusOverturned")}</Badge>;
    case "dismissed":
      return <Badge variant="secondary">{t("aiAudit.reviewStatusDismissed")}</Badge>;
    default:
      return <Badge variant="outline">{status}</Badge>;
  }
}

const PER_PAGE = 15;

export function AIReviewsTab() {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const params = new URLSearchParams({ page: String(page), page_size: String(PER_PAGE) });
  const { data, isLoading } = useAdminAIReviews(params.toString());
  const reviewMutation = useAdminReviewAIReview();

  const [reviewTarget, setReviewTarget] = useState<AuditAIReview | null>(null);
  const [reviewStatus, setReviewStatus] = useState<string>("confirmed");
  const [reviewNote, setReviewNote] = useState("");

  const reviews = (data?.data ?? []) as AuditAIReview[];
  const total = (data?.total ?? 0) as number;
  const totalPages = Math.ceil(total / PER_PAGE);

  const handleReview = () => {
    if (!reviewTarget) return;
    reviewMutation.mutate(
      { id: reviewTarget.id, status: reviewStatus, note: reviewNote.trim() || undefined },
      { onSuccess: () => {
        setReviewTarget(null);
        setReviewStatus("confirmed");
        setReviewNote("");
      }},
    );
  };

  const openReview = (r: AuditAIReview) => {
    setReviewTarget(r);
    setReviewStatus("confirmed");
    setReviewNote("");
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("aiAudit.colFqdn")}</TableHead>
                  <TableHead>{t("aiAudit.colJudgment")}</TableHead>
                  <TableHead>{t("aiAudit.colConfidence")}</TableHead>
                  <TableHead>{t("aiAudit.colReviewStatus")}</TableHead>
                  <TableHead>{t("aiAudit.colCreatedAt")}</TableHead>
                  <TableHead className="text-right">{t("aiAudit.colActions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {[...Array(5)].map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="h-4 w-28" /></TableCell>
                    <TableCell><Skeleton className="h-5 w-16 rounded-full" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-12" /></TableCell>
                    <TableCell><Skeleton className="h-5 w-16 rounded-full" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-28" /></TableCell>
                    <TableCell className="text-right"><Skeleton className="h-8 w-20 ml-auto" /></TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("aiAudit.colFqdn")}</TableHead>
                <TableHead>{t("aiAudit.colJudgment")}</TableHead>
                <TableHead>{t("aiAudit.colConfidence")}</TableHead>
                <TableHead>{t("aiAudit.colViolationTypes")}</TableHead>
                <TableHead>{t("aiAudit.colReviewStatus")}</TableHead>
                <TableHead>{t("aiAudit.colCreatedAt")}</TableHead>
                <TableHead className="text-right">{t("aiAudit.colActions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {reviews.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center text-muted-foreground py-8">
                    {t("aiAudit.noReviews")}
                  </TableCell>
                </TableRow>
              ) : (
                reviews.map((r) => (
                  <TableRow key={r.id}>
                    <TableCell className="font-medium font-mono text-xs">{r.fqdn}</TableCell>
                    <TableCell>{judgmentBadge(r.ai_judgment, t)}</TableCell>
                    <TableCell className="text-muted-foreground">{(r.ai_confidence * 100).toFixed(1)}%</TableCell>
                    <TableCell className="text-xs text-muted-foreground max-w-[200px] truncate">
                      {r.violation_types?.join(", ") || "-"}
                    </TableCell>
                    <TableCell>{reviewStatusBadge(r.admin_review_status, t)}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {new Date(r.created_at).toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button variant="ghost" size="sm" onClick={() => openReview(r)}>
                        {t("aiAudit.review")}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {totalPages > 1 && (
        <div className="flex justify-center gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
            {t("common.previous")}
          </Button>
          <span className="flex items-center text-sm text-muted-foreground">
            {t("common.pageOf", { page, total: totalPages })}
          </span>
          <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>
            {t("common.next")}
          </Button>
        </div>
      )}

      {/* Admin Review Dialog */}
      <Dialog open={!!reviewTarget} onOpenChange={(open) => !open && setReviewTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("aiAudit.adminReviewTitle")}</DialogTitle>
          </DialogHeader>
          {reviewTarget && (
            <div className="space-y-4">
              <div className="rounded-md border p-3 space-y-2 text-sm">
                <div className="flex items-center gap-2">
                  <span className="text-muted-foreground">{t("aiAudit.colFqdn")}:</span>
                  <span className="font-mono">{reviewTarget.fqdn}</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-muted-foreground">{t("aiAudit.colJudgment")}:</span>
                  {judgmentBadge(reviewTarget.ai_judgment, t)}
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-muted-foreground">{t("aiAudit.colConfidence")}:</span>
                  <span>{(reviewTarget.ai_confidence * 100).toFixed(1)}%</span>
                </div>
                {reviewTarget.ai_suggested_action && (
                  <div className="flex gap-2">
                    <span className="text-muted-foreground shrink-0">{t("aiAudit.colSuggestedAction")}:</span>
                    <span>{reviewTarget.ai_suggested_action}</span>
                  </div>
                )}
                {reviewTarget.ai_response && (
                  <div className="flex gap-2">
                    <span className="text-muted-foreground shrink-0">{t("aiAudit.colAiResponse")}:</span>
                    <span className="text-xs break-all">{reviewTarget.ai_response}</span>
                  </div>
                )}
              </div>
              <div className="space-y-2">
                <Label>{t("aiAudit.colReviewStatus")}</Label>
                <Select value={reviewStatus} onValueChange={setReviewStatus}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="confirmed">{t("aiAudit.reviewStatusConfirmed")}</SelectItem>
                    <SelectItem value="overturned">{t("aiAudit.reviewStatusOverturned")}</SelectItem>
                    <SelectItem value="dismissed">{t("aiAudit.reviewStatusDismissed")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>{t("aiAudit.colAdminNote")}</Label>
                <Textarea
                  value={reviewNote}
                  onChange={(e) => setReviewNote(e.target.value)}
                  placeholder={t("aiAudit.adminNotePlaceholder")}
                  rows={3}
                />
              </div>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setReviewTarget(null)}>{t("common.cancel")}</Button>
            <Button onClick={handleReview} disabled={reviewMutation.isPending}>
              {reviewMutation.isPending ? t("common.saving") : t("common.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
