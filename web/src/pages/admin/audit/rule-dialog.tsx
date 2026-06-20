import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useErrorToast } from "@/hooks/use-error-toast";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import type { AuditRule, AuditScenario } from "@/types";
import { RuleTestPanel } from "./rule-test-panel";
import { X } from "lucide-react";

const TARGETS = ["body", "title", "final_url", "status_code"] as const;
const ACTIONS = ["observe", "site", "user"] as const;

type DraftRule = {
  name: string;
  enabled: boolean;
  scenario_id: string;
  description: string;
  targets: string[];
  match_type: string;
  keywords: string[];
  keyword_logic: string;
  pattern: string;
  case_sensitive: boolean;
  action: string;
  scope_domain_ids: number[];
};

const emptyDraft = (): DraftRule => ({
  name: "",
  enabled: true,
  scenario_id: "",
  description: "",
  targets: ["body"],
  match_type: "keyword",
  keywords: [],
  keyword_logic: "any",
  pattern: "",
  case_sensitive: false,
  action: "site",
  scope_domain_ids: [],
});

export function RuleDialog({
  open,
  setOpen,
  rule,
}: {
  open: boolean;
  setOpen: (v: boolean) => void;
  rule?: AuditRule;
}) {
  const { t } = useTranslation();
  const showError = useErrorToast();
  const queryClient = useQueryClient();
  const [mode, setMode] = useState<"scenario" | "custom">("custom");
  const [draft, setDraft] = useState<DraftRule>(emptyDraft());
  const [keywordInput, setKeywordInput] = useState("");

  const { data: scenarios } = useQuery({
    queryKey: ["admin-audit-scenarios"],
    queryFn: async () => (await api.adminListAuditScenarios()).data,
    enabled: open,
  });

  const { data: domains } = useQuery({
    queryKey: ["admin-domains-full"],
    queryFn: async () => (await api.adminListDomainsFull()).data,
    enabled: open,
  });

  useEffect(() => {
    if (!open) return;
    if (rule) {
      setMode("custom");
      setDraft({
        name: rule.name,
        enabled: rule.enabled,
        scenario_id: rule.scenario_id ?? "",
        description: rule.description ?? "",
        targets: [...rule.targets],
        match_type: rule.match_type,
        keywords: [...rule.keywords],
        keyword_logic: rule.keyword_logic,
        pattern: rule.pattern,
        case_sensitive: rule.case_sensitive,
        action: rule.action,
        scope_domain_ids: [...rule.scope_domain_ids],
      });
    } else {
      setMode("scenario");
      setDraft(emptyDraft());
    }
    setKeywordInput("");
  }, [open, rule]);

  const applyScenario = (scenario: AuditScenario) => {
    setDraft((d) => ({
      ...d,
      scenario_id: scenario.id,
      targets: [...scenario.targets],
      match_type: scenario.match_type,
      keywords: scenario.keywords ? [...scenario.keywords] : [],
      pattern: scenario.pattern ?? "",
      keyword_logic: scenario.keyword_logic ?? "any",
    }));
  };

  const createMutation = useMutation({
    mutationFn: () => api.adminCreateAuditRule(draft),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-audit-rules"] });
      toast.success(t("audit.rules.created"));
      setOpen(false);
    },
    onError: (err) => showError(err),
  });

  const updateMutation = useMutation({
    mutationFn: () => api.adminUpdateAuditRule(rule!.id, draft),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-audit-rules"] });
      toast.success(t("audit.rules.updated"));
      setOpen(false);
    },
    onError: (err) => showError(err),
  });

  const isPending = createMutation.isPending || updateMutation.isPending;
  const isEdit = !!rule;
  const hasStatusCode = draft.targets.includes("status_code");

  const toggleTarget = (target: string, checked: boolean) => {
    if (target === "status_code" && checked) {
      setDraft((d) => ({ ...d, targets: ["status_code"], match_type: "status_eq" }));
      return;
    }
    setDraft((d) => {
      const next = checked ? [...d.targets.filter((x) => x !== "status_code"), target] : d.targets.filter((x) => x !== target);
      return { ...d, targets: next.length ? next : ["body"] };
    });
  };

  const addKeyword = () => {
    const kw = keywordInput.trim();
    if (!kw || draft.keywords.includes(kw)) return;
    setDraft((d) => ({ ...d, keywords: [...d.keywords, kw] }));
    setKeywordInput("");
  };

  const save = () => {
    if (isEdit) updateMutation.mutate();
    else createMutation.mutate();
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="flex max-h-[90vh] max-w-2xl flex-col gap-0 overflow-hidden p-0">
        <DialogHeader className="border-b px-6 py-4">
          <DialogTitle>{isEdit ? t("audit.rules.edit") : t("audit.rules.create")}</DialogTitle>
          <DialogDescription>{t("audit.rules.dialogDesc")}</DialogDescription>
        </DialogHeader>

        <div className="flex-1 space-y-4 overflow-y-auto px-6 py-5">
          {!isEdit && (
            <div className="flex gap-2">
              <Button type="button" variant={mode === "scenario" ? "default" : "outline"} size="sm" onClick={() => setMode("scenario")}>
                {t("audit.rules.fromScenario")}
              </Button>
              <Button type="button" variant={mode === "custom" ? "default" : "outline"} size="sm" onClick={() => setMode("custom")}>
                {t("audit.rules.custom")}
              </Button>
            </div>
          )}

          {mode === "scenario" && !isEdit && (
            <div className="grid gap-2 sm:grid-cols-2">
              {(scenarios ?? []).map((s) => (
                <button
                  key={s.id}
                  type="button"
                  className={`rounded-md border p-3 text-left text-sm hover:bg-muted ${draft.scenario_id === s.id ? "border-primary" : ""}`}
                  onClick={() => applyScenario(s)}
                >
                  <p className="font-medium">{t(s.name_key)}</p>
                  <p className="text-xs text-muted-foreground mt-1">{t(s.desc_key)}</p>
                </button>
              ))}
            </div>
          )}

          <div className="space-y-2">
            <Label>{t("audit.rules.fields.name")}</Label>
            <Input value={draft.name} onChange={(e) => setDraft((d) => ({ ...d, name: e.target.value }))} />
          </div>

          <div className="space-y-2">
            <Label>{t("audit.rules.fields.targets")}</Label>
            <div className="flex flex-wrap gap-3">
              {TARGETS.map((tg) => (
                <label key={tg} className="flex items-center gap-2 text-sm">
                  <Checkbox
                    checked={draft.targets.includes(tg)}
                    disabled={hasStatusCode && tg !== "status_code"}
                    onCheckedChange={(c) => toggleTarget(tg, !!c)}
                  />
                  {t(`audit.targets.${tg}`)}
                </label>
              ))}
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>{t("audit.rules.fields.matchType")}</Label>
              <Select
                value={draft.match_type}
                disabled={hasStatusCode}
                onValueChange={(v) => setDraft((d) => ({ ...d, match_type: v }))}
              >
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="keyword">{t("audit.matchType.keyword")}</SelectItem>
                  <SelectItem value="regex">{t("audit.matchType.regex")}</SelectItem>
                  <SelectItem value="status_eq">{t("audit.matchType.status_eq")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>{t("audit.rules.fields.action")}</Label>
              <Select value={draft.action} onValueChange={(v) => setDraft((d) => ({ ...d, action: v }))}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {ACTIONS.map((a) => (
                    <SelectItem key={a} value={a}>{t(`audit.actions.${a}`)}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {draft.match_type === "keyword" && (
            <div className="space-y-2">
              <Label>{t("audit.rules.fields.keywords")}</Label>
              <div className="flex gap-2">
                <Input
                  value={keywordInput}
                  placeholder={t("audit.rules.keywordPlaceholder")}
                  onChange={(e) => setKeywordInput(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && (e.preventDefault(), addKeyword())}
                />
                <Button type="button" variant="outline" onClick={addKeyword}>{t("common.add")}</Button>
              </div>
              <div className="flex flex-wrap gap-1">
                {draft.keywords.map((kw) => (
                  <Badge key={kw} variant="secondary" className="gap-1">
                    {kw}
                    <button type="button" onClick={() => setDraft((d) => ({ ...d, keywords: d.keywords.filter((k) => k !== kw) }))}>
                      <X className="h-3 w-3" />
                    </button>
                  </Badge>
                ))}
              </div>
              <Select value={draft.keyword_logic} onValueChange={(v) => setDraft((d) => ({ ...d, keyword_logic: v }))}>
                <SelectTrigger className="w-40"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="any">{t("audit.keywordLogic.any")}</SelectItem>
                  <SelectItem value="all">{t("audit.keywordLogic.all")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          )}

          {(draft.match_type === "regex" || draft.match_type === "status_eq") && (
            <div className="space-y-2">
              <Label>{draft.match_type === "status_eq" ? t("audit.rules.fields.statusCode") : t("audit.rules.fields.pattern")}</Label>
              <Input value={draft.pattern} onChange={(e) => setDraft((d) => ({ ...d, pattern: e.target.value }))} />
            </div>
          )}

          <div className="flex items-center gap-2">
            <Switch checked={draft.case_sensitive} onCheckedChange={(v) => setDraft((d) => ({ ...d, case_sensitive: v }))} />
            <Label>{t("audit.rules.fields.caseSensitive")}</Label>
          </div>

          <div className="space-y-2">
            <Label>{t("audit.rules.fields.scope")}</Label>
            <div className="flex flex-wrap gap-2 max-h-32 overflow-y-auto border rounded-md p-2">
              {(domains ?? []).map((dom) => {
                const checked = draft.scope_domain_ids.includes(dom.id);
                return (
                  <label key={dom.id} className="flex items-center gap-1.5 text-xs">
                    <Checkbox
                      checked={checked}
                      onCheckedChange={(c) => {
                        setDraft((d) => ({
                          ...d,
                          scope_domain_ids: c
                            ? [...d.scope_domain_ids, dom.id]
                            : d.scope_domain_ids.filter((id) => id !== dom.id),
                        }));
                      }}
                    />
                    {dom.name}
                  </label>
                );
              })}
            </div>
            <p className="text-xs text-muted-foreground">{t("audit.rules.scopeHint")}</p>
          </div>

          <RuleTestPanel draft={draft} />
        </div>

        <DialogFooter className="gap-2 border-t px-6 py-4 sm:gap-2">
          <Button variant="outline" onClick={() => setOpen(false)} disabled={isPending}>{t("common.cancel")}</Button>
          <Button onClick={save} disabled={isPending || !draft.name.trim()}>
            {isPending ? t("common.saving") : t("common.save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
