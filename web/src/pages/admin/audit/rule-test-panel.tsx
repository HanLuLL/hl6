import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useErrorToast } from "@/hooks/use-error-toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import type { AuditFetchChannelDetail, AuditRuleTestResult } from "@/types";

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

function ChannelSummary({ label, ch }: { label: string; ch: AuditFetchChannelDetail }) {
  const statusLabel = ch.http_status_code > 0 ? `HTTP ${ch.http_status_code}` : ch.status;
  return (
    <div className="rounded-md bg-muted/40 px-2.5 py-2 text-xs text-muted-foreground">
      <span className="font-medium text-foreground">{label}</span>
      <span className="mx-1.5 text-border">|</span>
      {statusLabel}
      {ch.final_url && <span className="ml-1.5 truncate">{ch.final_url}</span>}
      {ch.error_message && <span className="ml-1.5 text-destructive/80">{ch.error_message}</span>}
      {ch.title_preview && (
        <p className="mt-1 truncate text-foreground/70">{ch.title_preview}</p>
      )}
    </div>
  );
}

export function RuleTestPanel({ draft, embedded }: { draft: DraftRule; embedded?: boolean }) {
  const { t } = useTranslation();
  const showError = useErrorToast();
  const [fqdn, setFqdn] = useState("");
  const [result, setResult] = useState<AuditRuleTestResult | null>(null);

  const testMutation = useMutation({
    mutationFn: () =>
      api.adminTestAuditRule({
        fqdn: fqdn.trim(),
        rule: {
          name: draft.name || "test",
          enabled: true,
          scenario_id: draft.scenario_id,
          description: draft.description,
          targets: draft.targets,
          match_type: draft.match_type,
          keywords: draft.keywords,
          keyword_logic: draft.keyword_logic,
          pattern: draft.pattern,
          case_sensitive: draft.case_sensitive,
          action: draft.action,
          scope_domain_ids: draft.scope_domain_ids,
          ban_notify_content: draft.ban_notify_content,
          exempt_enabled: draft.exempt_enabled,
          exempt_recheck_minutes: draft.exempt_recheck_minutes,
          exempt_notify_content: draft.exempt_notify_content,
        },
      }),
    onSuccess: (res) => setResult(res.data),
    onError: (err) => showError(err),
  });

  return (
    <div className={embedded ? "space-y-3" : "space-y-3 rounded-lg border border-border/60 p-3.5"}>
      {!embedded && <p className="text-sm font-medium">{t("audit.ruleTest.title")}</p>}
      <div className="flex gap-2">
        <Input
          placeholder={t("audit.ruleTest.fqdnPlaceholder")}
          value={fqdn}
          onChange={(e) => setFqdn(e.target.value)}
        />
        <Button
          type="button"
          variant="outline"
          className="shrink-0"
          disabled={!fqdn.trim() || testMutation.isPending}
          onClick={() => testMutation.mutate()}
        >
          {testMutation.isPending ? t("audit.ruleTest.running") : t("audit.ruleTest.run")}
        </Button>
      </div>

      {result && (
        <div className="space-y-2 text-sm">
          <ChannelSummary label="HTTPS" ch={result.fetch.https} />
          <ChannelSummary label="HTTP" ch={result.fetch.http} />
          {result.matched_rules.length === 0 ? (
            <p className="text-xs text-muted-foreground">{t("audit.ruleTest.noMatch")}</p>
          ) : (
            <ul className="space-y-1.5">
              {result.matched_rules.map((m, i) => (
                <li key={i} className="flex items-start gap-2 text-xs">
                  <Badge variant="outline" className="shrink-0">{t(`audit.actions.${m.action}`)}</Badge>
                  <span className="min-w-0 break-words">{m.rule_name}: {m.snippet}</span>
                </li>
              ))}
            </ul>
          )}
          <p className="text-xs font-medium">
            {result.would_exempt
              ? t("audit.ruleTest.wouldExempt")
              : result.would_delete_dns
                ? t("audit.ruleTest.wouldDeleteDns")
                : result.would_suspend
                  ? t("audit.ruleTest.wouldSuspend")
                  : t("audit.ruleTest.wouldNotSuspend")}
          </p>
          {result.would_send_ban_notify && (
            <p className="text-xs text-muted-foreground">{t("audit.ruleTest.wouldSendBanNotify")}</p>
          )}
          {result.would_send_exempt_notify && (
            <p className="text-xs text-muted-foreground">{t("audit.ruleTest.wouldSendExemptNotify")}</p>
          )}
        </div>
      )}
    </div>
  );
}
