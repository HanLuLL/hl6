import { useEffect, useState, type ReactNode } from "react";
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
import { Separator } from "@/components/ui/separator";
import {
  Collapsible, CollapsibleContent, CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { cn } from "@/lib/utils";
import type { AuditRule, AuditScenario } from "@/types";
import { RuleTestPanel } from "./rule-test-panel";
import { ChevronDown, X } from "lucide-react";

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

function FormSection({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children: ReactNode;
}) {
  return (
    <section className="space-y-3">
      <div>
        <h3 className="text-sm font-medium">{title}</h3>
        {description && (
          <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>
        )}
      </div>
      <div className="space-y-3">{children}</div>
    </section>
  );
}

function FieldRow({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return (
    <div className="space-y-1.5">
      <Label className="text-[13px]">{label}</Label>
      {children}
      {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
    </div>
  );
}

function TemplateVars({ vars }: { vars: string }) {
  return (
    <p className="text-[11px] leading-relaxed text-muted-foreground">
      <span className="text-foreground/70">{vars.split(":")[0]}:</span>
      {vars.includes(":") && (
        <span className="ml-1 font-mono">{vars.split(":").slice(1).join(":").trim()}</span>
      )}
    </p>
  );
}

function OptionalBlock({
  title,
  summary,
  configured,
  open,
  onOpenChange,
  children,
}: {
  title: string;
  summary?: string;
  configured?: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: ReactNode;
}) {
  const { t } = useTranslation();

  return (
    <Collapsible open={open} onOpenChange={onOpenChange}>
      <div className="rounded-lg border border-border/60">
        <CollapsibleTrigger className="flex w-full items-center gap-3 px-3.5 py-2.5 text-left hover:bg-muted/40">
          <ChevronDown
            className={cn(
              "h-4 w-4 shrink-0 text-muted-foreground transition-transform duration-200",
              open && "rotate-180",
            )}
          />
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium">{title}</span>
              {configured && !open && (
                <Badge variant="secondary" className="h-5 px-1.5 text-[10px] font-normal">
                  {t("audit.rules.configured")}
                </Badge>
              )}
            </div>
            {summary && !open && (
              <p className="mt-0.5 truncate text-xs text-muted-foreground">{summary}</p>
            )}
          </div>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <div className="space-y-3 border-t px-3.5 pb-3.5 pt-3">{children}</div>
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}

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
  const [banNotifyOpen, setBanNotifyOpen] = useState(false);
  const [testOpen, setTestOpen] = useState(false);

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
      setBanNotifyOpen(!!rule.ban_notify_content?.trim());
      setTestOpen(false);
    } else {
      setMode("scenario");
      setDraft(emptyDraft());
      setBanNotifyOpen(false);
      setTestOpen(false);
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
  const showCaseSensitive = !isUnreachable && draft.match_type !== "status_eq";
  const banNotifyChars = draft.ban_notify_content.length;
  const exemptNotifyChars = draft.exempt_notify_content.length;
  const scopeCount = draft.scope_domain_ids.length;

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

        <div className="flex-1 space-y-6 overflow-y-auto px-6 py-5">
          {!isEdit && (
            <div className="inline-flex rounded-lg bg-muted p-1">
              <Button
                type="button"
                variant={mode === "scenario" ? "default" : "ghost"}
                size="sm"
                className="h-8 px-3"
                onClick={() => setMode("scenario")}
              >
                {t("audit.rules.fromScenario")}
              </Button>
              <Button
                type="button"
                variant={mode === "custom" ? "default" : "ghost"}
                size="sm"
                className="h-8 px-3"
                onClick={() => setMode("custom")}
              >
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
                  className={cn(
                    "rounded-lg border p-3 text-left text-sm transition-colors hover:bg-muted/50",
                    draft.scenario_id === s.id && "border-primary bg-primary/5 ring-1 ring-primary/20",
                  )}
                  onClick={() => applyScenario(s)}
                >
                  <p className="font-medium">{t(s.name_key)}</p>
                  <p className="mt-1 text-xs text-muted-foreground line-clamp-2">{t(s.desc_key)}</p>
                </button>
              ))}
            </div>
          )}

          <FormSection title={t("audit.rules.sections.basics")}>
            <FieldRow label={t("audit.rules.fields.name")}>
              <Input
                value={draft.name}
                onChange={(e) => setDraft((d) => ({ ...d, name: e.target.value }))}
              />
            </FieldRow>
          </FormSection>

          <Separator />

          <FormSection title={t("audit.rules.sections.match")}>
            <FieldRow label={t("audit.rules.fields.targets")}>
              {isUnreachable ? (
                <p className="rounded-md bg-muted/50 px-3 py-2 text-xs leading-relaxed text-muted-foreground">
                  {t("audit.matchType.unreachableHint")}
                </p>
              ) : (
                <div className="flex flex-wrap gap-x-4 gap-y-2">
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
            </FieldRow>

            <div className="grid gap-3 sm:grid-cols-2">
              <FieldRow label={t("audit.rules.fields.matchType")}>
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
              </FieldRow>
              <FieldRow label={t("audit.rules.fields.action")}>
                <Select value={draft.action} onValueChange={(v) => setDraft((d) => ({ ...d, action: v }))}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {ACTIONS.map((a) => (
                      <SelectItem key={a} value={a}>{t(`audit.actions.${a}`)}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FieldRow>
            </div>

            {draft.match_type === "keyword" && !isUnreachable && (
              <FieldRow label={t("audit.rules.fields.keywords")}>
                <div className="flex gap-2">
                  <Input
                    value={keywordInput}
                    placeholder={t("audit.rules.keywordPlaceholder")}
                    onChange={(e) => setKeywordInput(e.target.value)}
                    onKeyDown={(e) => e.key === "Enter" && (e.preventDefault(), addKeyword())}
                  />
                  <Button type="button" variant="outline" className="shrink-0" onClick={addKeyword}>
                    {t("common.add")}
                  </Button>
                </div>
                {draft.keywords.length > 0 && (
                  <div className="flex flex-wrap gap-1.5 pt-0.5">
                    {draft.keywords.map((kw) => (
                      <Badge key={kw} variant="secondary" className="gap-1 pr-1">
                        {kw}
                        <button
                          type="button"
                          className="rounded-sm p-0.5 hover:bg-background/60"
                          onClick={() => setDraft((d) => ({ ...d, keywords: d.keywords.filter((k) => k !== kw) }))}
                        >
                          <X className="h-3 w-3" />
                        </button>
                      </Badge>
                    ))}
                  </div>
                )}
                <div className="flex flex-wrap items-center justify-between gap-3 pt-1">
                  <Select value={draft.keyword_logic} onValueChange={(v) => setDraft((d) => ({ ...d, keyword_logic: v }))}>
                    <SelectTrigger className="h-9 w-full sm:w-48"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="any">{t("audit.keywordLogic.any")}</SelectItem>
                      <SelectItem value="all">{t("audit.keywordLogic.all")}</SelectItem>
                    </SelectContent>
                  </Select>
                  {showCaseSensitive && (
                    <label className="flex items-center gap-2 text-sm">
                      <Switch
                        checked={draft.case_sensitive}
                        onCheckedChange={(v) => setDraft((d) => ({ ...d, case_sensitive: v }))}
                      />
                      {t("audit.rules.fields.caseSensitive")}
                    </label>
                  )}
                </div>
              </FieldRow>
            )}

            {(draft.match_type === "regex" || draft.match_type === "status_eq") && !isUnreachable && (
              <FieldRow
                label={draft.match_type === "status_eq" ? t("audit.rules.fields.statusCode") : t("audit.rules.fields.pattern")}
              >
                <Input value={draft.pattern} onChange={(e) => setDraft((d) => ({ ...d, pattern: e.target.value }))} />
                {showCaseSensitive && (
                  <label className="flex items-center gap-2 pt-1 text-sm">
                    <Switch
                      checked={draft.case_sensitive}
                      onCheckedChange={(v) => setDraft((d) => ({ ...d, case_sensitive: v }))}
                    />
                    {t("audit.rules.fields.caseSensitive")}
                  </label>
                )}
              </FieldRow>
            )}
          </FormSection>

          <Separator />

          <FormSection
            title={t("audit.rules.fields.scope")}
            description={t("audit.rules.scopeHint")}
          >
            {scopeCount > 0 && (
              <p className="text-xs text-muted-foreground">
                {t("audit.rules.scopeSelected", { count: scopeCount })}
              </p>
            )}
            <div className="max-h-28 overflow-y-auto rounded-md bg-muted/30 p-2.5">
              <div className="flex flex-wrap gap-x-3 gap-y-2">
                {(domains ?? []).map((dom) => {
                  const checked = draft.scope_domain_ids.includes(dom.id);
                  return (
                    <label key={dom.id} className="flex items-center gap-1.5 text-sm">
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
            </div>
          </FormSection>

          <Separator />

          <div className="space-y-3">
            {showBanNotify && (
              <OptionalBlock
                title={t("audit.rules.banNotify.title")}
                summary={t("audit.rules.banNotify.hint")}
                configured={!!draft.ban_notify_content.trim()}
                open={banNotifyOpen}
                onOpenChange={setBanNotifyOpen}
              >
                <div className="flex items-center justify-between gap-2">
                  <p className="text-xs text-muted-foreground">{t("audit.rules.banNotify.hint")}</p>
                  <span className="shrink-0 text-xs tabular-nums text-muted-foreground">
                    {banNotifyChars}/{NOTIFY_MAX}
                  </span>
                </div>
                <Textarea
                  value={draft.ban_notify_content}
                  rows={3}
                  maxLength={NOTIFY_MAX}
                  placeholder={t("audit.rules.banNotify.placeholder")}
                  onChange={(e) => setDraft((d) => ({ ...d, ban_notify_content: e.target.value }))}
                />
                <TemplateVars vars={t("audit.rules.banNotify.vars")} />
              </OptionalBlock>
            )}

            <div className="rounded-lg border border-border/60 px-3.5 py-3">
              <div className="flex items-start justify-between gap-4">
                <div className="min-w-0 space-y-0.5">
                  <p className="text-sm font-medium">{t("audit.rules.exempt.title")}</p>
                  <p className="text-xs text-muted-foreground">{t("audit.rules.exempt.hint")}</p>
                </div>
                <Switch
                  className="shrink-0"
                  checked={draft.exempt_enabled}
                  onCheckedChange={(v) => setDraft((d) => ({ ...d, exempt_enabled: v }))}
                />
              </div>
              {draft.exempt_enabled && (
                <div className="mt-4 space-y-3 border-t pt-3">
                  <FieldRow
                    label={t("audit.rules.exempt.recheckMinutes")}
                    hint={t("audit.rules.exempt.recheckRange", { min: EXEMPT_RECHECK_MIN, max: EXEMPT_RECHECK_MAX })}
                  >
                    <Input
                      type="number"
                      className="max-w-[140px]"
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
                  </FieldRow>
                  <div className="space-y-1.5">
                    <div className="flex items-center justify-between gap-2">
                      <Label className="text-[13px]">{t("audit.rules.exempt.notifyTitle")}</Label>
                      <span className="text-xs tabular-nums text-muted-foreground">
                        {exemptNotifyChars}/{NOTIFY_MAX}
                      </span>
                    </div>
                    <Textarea
                      value={draft.exempt_notify_content}
                      rows={3}
                      maxLength={NOTIFY_MAX}
                      placeholder={t("audit.rules.exempt.notifyPlaceholder")}
                      onChange={(e) => setDraft((d) => ({ ...d, exempt_notify_content: e.target.value }))}
                    />
                    <TemplateVars vars={t("audit.rules.exempt.vars")} />
                  </div>
                </div>
              )}
            </div>

            <OptionalBlock
              title={t("audit.ruleTest.title")}
              open={testOpen}
              onOpenChange={setTestOpen}
            >
              <RuleTestPanel draft={draft} embedded />
            </OptionalBlock>
          </div>
        </div>

        <DialogFooter className="gap-2 border-t px-6 py-4 sm:gap-2">
          <Button variant="outline" onClick={() => setOpen(false)} disabled={isPending}>{t("common.cancel")}</Button>
          <Button
            onClick={save}
            disabled={
              isPending
              || !draft.name.trim()
              || (draft.exempt_enabled && (draft.exempt_recheck_minutes < EXEMPT_RECHECK_MIN || draft.exempt_recheck_minutes > EXEMPT_RECHECK_MAX))
            }
          >
            {isPending ? t("common.saving") : t("common.save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
