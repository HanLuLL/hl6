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
import { Textarea } from "@/components/ui/textarea";
import type { AuditRule, AuditScenario } from "@/types";
import { RuleTestPanel } from "./rule-test-panel";
import { X } from "lucide-react";

const TARGETS = ["body", "title", "final_url", "status_code"] as const;
const ACTIONS = ["observe", "delete_dns", "site", "user"] as const;

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
  ban_notify_content: string;
  exempt_enabled: boolean;
  exempt_recheck_minutes: number;
  exempt_notify_content: string;
};

const NOTIFY_MAX = 1024;
const EXEMPT_RECHECK_MIN = 5;
const EXEMPT_RECHECK_MAX = 10080;

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
  ban_notify_content: "",
  exempt_enabled: false,
  exempt_recheck_minutes: 60,
  exempt_notify_content: "",
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
        ban_notify_content: rule.ban_notify_content ?? "",
        exempt_enabled: rule.exempt_enabled ?? false,
        exempt_recheck_minutes: rule.exempt_recheck_minutes ?? 60,
        exempt_notify_content: rule.exempt_notify_content ?? "",
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
      targets: scenario.match_type === "unreachable" ? [] : [...scenario.targets],
      match_type: scenario.match_type,
      keywords: scenario.keywords ? [...scenario.keywords] : [],
      pattern: scenario.pattern ?? "",
      keyword_logic: scenario.keyword_logic ?? "any",
      action: scenario.id === "unreachable" ? "delete_dns" : d.action,
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
  const isUnreachable = draft.match_type === "unreachable";
  const hasStatusCode = draft.targets.includes("status_code");
  const showBanNotify = draft.action !== "observe";
  const banNotifyChars = draft.ban_notify_content.length;
  const exemptNotifyChars = draft.exempt_notify_content.length;

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
            {isUnreachable ? (
              <p className="text-xs text-muted-foreground">{t("audit.matchType.unreachableHint")}</p>
            ) : (
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
            )}
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>{t("audit.rules.fields.matchType")}</Label>
              <Select
                value={draft.match_type}
                disabled={hasStatusCode}
                onValueChange={(v) => {
                  if (v === "unreachable") {
                    setDraft((d) => ({ ...d, match_type: v, targets: [], keywords: [], pattern: "" }));
                  } else {
                    setDraft((d) => ({ ...d, match_type: v, targets: d.targets.length ? d.targets : ["body"] }));
                  }
                }}
              >
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="keyword">{t("audit.matchType.keyword")}</SelectItem>
                  <SelectItem value="regex">{t("audit.matchType.regex")}</SelectItem>
                  <SelectItem value="status_eq">{t("audit.matchType.status_eq")}</SelectItem>
                  <SelectItem value="unreachable">{t("audit.matchType.unreachable")}</SelectItem>
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

          {draft.match_type === "keyword" && !isUnreachable && (
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

          {(draft.match_type === "regex" || draft.match_type === "status_eq") && !isUnreachable && (
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

          {showBanNotify && (
            <div className="space-y-2 rounded-md border p-4">
              <Label>{t("audit.rules.banNotify.title")}</Label>
              <p className="text-xs text-muted-foreground">{t("audit.rules.banNotify.hint")}</p>
              <Textarea
                value={draft.ban_notify_content}
                rows={4}
                maxLength={NOTIFY_MAX}
                placeholder={t("audit.rules.banNotify.placeholder")}
                onChange={(e) => setDraft((d) => ({ ...d, ban_notify_content: e.target.value }))}
              />
              <p className="text-xs text-muted-foreground">
                {t("audit.rules.banNotify.vars")}
              </p>
              <p className="text-xs text-muted-foreground text-right">
                {banNotifyChars}/{NOTIFY_MAX}
              </p>
            </div>
          )}

          <div className="space-y-3 rounded-md border p-4">
            <div className="flex items-center justify-between gap-2">
              <div>
                <Label>{t("audit.rules.exempt.title")}</Label>
                <p className="text-xs text-muted-foreground mt-1">{t("audit.rules.exempt.hint")}</p>
              </div>
              <Switch
                checked={draft.exempt_enabled}
                onCheckedChange={(v) => setDraft((d) => ({ ...d, exempt_enabled: v }))}
              />
            </div>
            {draft.exempt_enabled && (
              <div className="space-y-3">
                <div className="space-y-2">
                  <Label>{t("audit.rules.exempt.recheckMinutes")}</Label>
                  <Input
                    type="number"
                    min={EXEMPT_RECHECK_MIN}
                    max={EXEMPT_RECHECK_MAX}
                    value={draft.exempt_recheck_minutes}
                    onChange={(e) => {
                      const n = parseInt(e.target.value, 10);
                      setDraft((d) => ({
                        ...d,
                        exempt_recheck_minutes: Number.isFinite(n) ? n : EXEMPT_RECHECK_MIN,
                      }));
                    }}
                  />
                  <p className="text-xs text-muted-foreground">
                    {t("audit.rules.exempt.recheckRange", { min: EXEMPT_RECHECK_MIN, max: EXEMPT_RECHECK_MAX })}
                  </p>
                </div>
                <div className="space-y-2">
                  <Label>{t("audit.rules.exempt.notifyTitle")}</Label>
                  <p className="text-xs text-muted-foreground">{t("audit.rules.exempt.notifyHint")}</p>
                  <Textarea
                    value={draft.exempt_notify_content}
                    rows={4}
                    maxLength={NOTIFY_MAX}
                    placeholder={t("audit.rules.exempt.notifyPlaceholder")}
                    onChange={(e) => setDraft((d) => ({ ...d, exempt_notify_content: e.target.value }))}
                  />
                  <p className="text-xs text-muted-foreground">
                    {t("audit.rules.exempt.vars")}
                  </p>
                  <p className="text-xs text-muted-foreground text-right">
                    {exemptNotifyChars}/{NOTIFY_MAX}
                  </p>
                </div>
              </div>
            )}
          </div>

          <RuleTestPanel draft={draft} />
        </div>

        <DialogFooter className="gap-2 border-t px-6 py-4 sm:gap-2">
          <Button variant="outline" onClick={() => setOpen(false)} disabled={isPending}>{t("common.cancel")}</Button>
          <Button onClick={save} disabled={isPending || !draft.name.trim() || (draft.exempt_enabled && (draft.exempt_recheck_minutes < EXEMPT_RECHECK_MIN || draft.exempt_recheck_minutes > EXEMPT_RECHECK_MAX))}>
            {isPending ? t("common.saving") : t("common.save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
