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
  useAdminAppeals,
  useAdminReviewAppeal,
} from "@/hooks/use-ai-audit";
import type { UserAppeal } from "@/types/ai-audit";

function appealStatusBadge(status: string, t: (key: string) => string) {
  switch (status) {
    case "pending":
      return <Badge className="bg-yellow-600 hover:bg-yellow-700">{t("aiAudit.appealStatusPending")}</Badge>;
    case "approved":
      return <Badge className="bg-green-600 hover:bg-green-700">{t("aiAudit.appealStatusApproved")}</Badge>;
    case "rejected":
      return <Badge variant="destructive">{t("aiAudit.appealStatusRejected")}</Badge>;
    default:
      return <Badge variant="outline">{status}</Badge>;
  }
}

const PER_PAGE = 15;

export function AppealsTab() {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const params = new URLSearchParams({ page: String(page), page_size: String(PER_PAGE) });
  const { data, isLoading } = useAdminAppeals(params.toString());
  const reviewMutation = useAdminReviewAppeal();

  const [appealTarget, setAppealTarget] = useState<UserAppeal | null>(null);
  const [appealStatus, setAppealStatus] = useState<string>("approved");
  const [appealReply, setAppealReply] = useState("");

  const appeals = (data?.data ?? []) as UserAppeal[];
  const total = (data?.total ?? 0) as number;
  const totalPages = Math.ceil(total / PER_PAGE);

  const handleReview = () => {
    if (!appealTarget) return;
    reviewMutation.mutate(
      { id: appealTarget.id, status: appealStatus, reply: appealReply.trim() || undefined },
      { onSuccess: () => {
        setAppealTarget(null);
        setAppealStatus("approved");
        setAppealReply("");
      }},
    );
  };

  const openReview = (a: UserAppeal) => {
    setAppealTarget(a);
    setAppealStatus("approved");
    setAppealReply("");
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("aiAudit.colUserId")}</TableHead>
                  <TableHead>{t("aiAudit.colContent")}</TableHead>
                  <TableHead>{t("aiAudit.colAppealStatus")}</TableHead>
                  <TableHead>{t("aiAudit.colCreatedAt")}</TableHead>
                  <TableHead className="text-right">{t("aiAudit.colActions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {[...Array(5)].map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-40" /></TableCell>
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
                <TableHead>{t("aiAudit.colUserId")}</TableHead>
                <TableHead>{t("aiAudit.colContent")}</TableHead>
                <TableHead>{t("aiAudit.colAppealStatus")}</TableHead>
                <TableHead>{t("aiAudit.colReply")}</TableHead>
                <TableHead>{t("aiAudit.colCreatedAt")}</TableHead>
                <TableHead className="text-right">{t("aiAudit.colActions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {appeals.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                    {t("aiAudit.noAppeals")}
                  </TableCell>
                </TableRow>
              ) : (
                appeals.map((a) => (
                  <TableRow key={a.id}>
                    <TableCell className="text-muted-foreground">#{a.user_id}</TableCell>
                    <TableCell className="max-w-xs truncate">{a.content}</TableCell>
                    <TableCell>{appealStatusBadge(a.status, t)}</TableCell>
                    <TableCell className="max-w-xs truncate text-muted-foreground">{a.reply || "-"}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {new Date(a.created_at).toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button variant="ghost" size="sm" onClick={() => openReview(a)}>
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

      {/* Appeal Review Dialog */}
      <Dialog open={!!appealTarget} onOpenChange={(open) => !open && setAppealTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("aiAudit.appealReviewTitle")}</DialogTitle>
          </DialogHeader>
          {appealTarget && (
            <div className="space-y-4">
              <div className="rounded-md border p-3 space-y-2 text-sm">
                <div className="flex items-center gap-2">
                  <span className="text-muted-foreground">{t("aiAudit.colUserId")}:</span>
                  <span>#{appealTarget.user_id}</span>
                </div>
                <div className="flex gap-2">
                  <span className="text-muted-foreground shrink-0">{t("aiAudit.colContent")}:</span>
                  <span>{appealTarget.content}</span>
                </div>
                {appealTarget.review_id && (
                  <div className="flex items-center gap-2">
                    <span className="text-muted-foreground">{t("aiAudit.colReviewId")}:</span>
                    <span>#{appealTarget.review_id}</span>
                  </div>
                )}
              </div>
              <div className="space-y-2">
                <Label>{t("aiAudit.colAppealStatus")}</Label>
                <Select value={appealStatus} onValueChange={setAppealStatus}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="approved">{t("aiAudit.appealStatusApproved")}</SelectItem>
                    <SelectItem value="rejected">{t("aiAudit.appealStatusRejected")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>{t("aiAudit.colReply")}</Label>
                <Textarea
                  value={appealReply}
                  onChange={(e) => setAppealReply(e.target.value)}
                  placeholder={t("aiAudit.appealReplyPlaceholder")}
                  rows={3}
                />
              </div>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setAppealTarget(null)}>{t("common.cancel")}</Button>
            <Button onClick={handleReview} disabled={reviewMutation.isPending}>
              {reviewMutation.isPending ? t("common.saving") : t("common.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
