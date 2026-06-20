import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useErrorToast } from "@/hooks/use-error-toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import type { AuditRuleTestResult } from "@/types";

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

export function RuleTestPanel({ draft }: { draft: DraftRule }) {
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
        },
      }),
    onSuccess: (res) => setResult(res.data),
    onError: (err) => showError(err),
  });

  return (
    <div className="rounded-md border p-4 space-y-3">
      <p className="text-sm font-medium">{t("audit.ruleTest.title")}</p>
      <div className="flex gap-2">
        <Input
          placeholder={t("audit.ruleTest.fqdnPlaceholder")}
          value={fqdn}
          onChange={(e) => setFqdn(e.target.value)}
        />
        <Button
          type="button"
          variant="outline"
          disabled={!fqdn.trim() || testMutation.isPending}
          onClick={() => testMutation.mutate()}
        >
          {testMutation.isPending ? t("audit.ruleTest.running") : t("audit.ruleTest.run")}
        </Button>
      </div>

      {result && (
        <div className="text-sm space-y-2">
          <div className="text-xs text-muted-foreground">
            HTTP {result.fetch.http_status_code} · {result.fetch.final_url || "—"}
          </div>
          {result.fetch.title_preview && (
            <p className="text-xs truncate">{result.fetch.title_preview}</p>
          )}
          {result.matched_rules.length === 0 ? (
            <p className="text-muted-foreground">{t("audit.ruleTest.noMatch")}</p>
          ) : (
            <ul className="space-y-1">
              {result.matched_rules.map((m, i) => (
                <li key={i} className="flex items-start gap-2">
                  <Badge variant="outline">{t(`audit.actions.${m.action}`)}</Badge>
                  <span>{m.rule_name}: {m.snippet}</span>
                </li>
              ))}
            </ul>
          )}
          <p className="text-xs">
            {result.would_suspend
              ? t("audit.ruleTest.wouldSuspend")
              : t("audit.ruleTest.wouldNotSuspend")}
          </p>
        </div>
      )}
    </div>
  );
}
