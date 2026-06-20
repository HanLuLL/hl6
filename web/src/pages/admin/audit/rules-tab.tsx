import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useErrorToast } from "@/hooks/use-error-toast";
import { toast } from "sonner";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import { Plus, Pencil, Trash2 } from "lucide-react";
import type { AuditRule } from "@/types";
import { formatDate } from "@/lib/format-date";
import { RuleDialog } from "./rule-dialog";

function summarizeTargets(rule: AuditRule, t: (k: string) => string) {
  return rule.targets.map((tg) => t(`audit.targets.${tg}`)).join(", ");
}

function summarizeMatch(rule: AuditRule) {
  if (rule.match_type === "keyword") {
    return `${rule.keyword_logic}: ${rule.keywords.slice(0, 3).join(", ")}${rule.keywords.length > 3 ? "…" : ""}`;
  }
  return rule.pattern || "—";
}

export function RulesTab() {
  const { t } = useTranslation();
  const showError = useErrorToast();
  const queryClient = useQueryClient();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editRule, setEditRule] = useState<AuditRule | undefined>();

  const { data: rules, isLoading } = useQuery({
    queryKey: ["admin-audit-rules"],
    queryFn: async () => (await api.adminListAuditRules()).data,
    staleTime: 30_000,
  });

  const toggleMutation = useMutation({
    mutationFn: (id: number) => api.adminToggleAuditRule(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["admin-audit-rules"] }),
    onError: (err) => showError(err),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.adminDeleteAuditRule(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-audit-rules"] });
      toast.success(t("audit.rules.deleted"));
    },
    onError: (err) => showError(err),
  });

  const openCreate = () => { setEditRule(undefined); setDialogOpen(true); };
  const openEdit = (rule: AuditRule) => { setEditRule(rule); setDialogOpen(true); };

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button onClick={openCreate} size="sm">
          <Plus className="h-4 w-4 mr-1" />
          {t("audit.rules.create")}
        </Button>
      </div>

      <Card>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="p-6"><Skeleton className="h-40 w-full" /></div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("audit.rules.columns.name")}</TableHead>
                  <TableHead>{t("audit.rules.columns.targets")}</TableHead>
                  <TableHead>{t("audit.rules.columns.match")}</TableHead>
                  <TableHead>{t("audit.rules.columns.action")}</TableHead>
                  <TableHead>{t("audit.rules.columns.hits7d")}</TableHead>
                  <TableHead>{t("audit.rules.columns.lastHit")}</TableHead>
                  <TableHead>{t("audit.rules.columns.enabled")}</TableHead>
                  <TableHead className="text-right">{t("audit.columns.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(rules ?? []).length === 0 && (
                  <TableRow>
                    <TableCell colSpan={8} className="text-center text-muted-foreground py-8">
                      {t("audit.rules.empty")}
                    </TableCell>
                  </TableRow>
                )}
                {(rules ?? []).map((rule) => (
                  <TableRow key={rule.id}>
                    <TableCell className="font-medium">{rule.name}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{summarizeTargets(rule, t)}</TableCell>
                    <TableCell className="text-xs font-mono max-w-[200px] truncate">{summarizeMatch(rule)}</TableCell>
                    <TableCell>
                      <Badge variant={rule.action === "user" ? "destructive" : rule.action === "site" ? "default" : "secondary"}>
                        {t(`audit.actions.${rule.action}`)}
                      </Badge>
                    </TableCell>
                    <TableCell>{rule.hit_count_7d ?? 0}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {rule.last_hit_at ? (
                        <>
                          <span className="block">{formatDate(rule.last_hit_at)}</span>
                          <span className="text-[10px]">{rule.last_hit_fqdn}</span>
                        </>
                      ) : "—"}
                    </TableCell>
                    <TableCell>
                      <Switch
                        checked={rule.enabled}
                        onCheckedChange={() => toggleMutation.mutate(rule.id)}
                        disabled={toggleMutation.isPending}
                      />
                    </TableCell>
                    <TableCell className="text-right space-x-1">
                      <Button variant="ghost" size="icon" onClick={() => openEdit(rule)}>
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="text-destructive"
                        onClick={() => deleteMutation.mutate(rule.id)}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <RuleDialog open={dialogOpen} setOpen={setDialogOpen} rule={editRule} />
    </div>
  );
}
