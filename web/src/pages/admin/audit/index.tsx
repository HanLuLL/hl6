import { useTranslation } from "react-i18next";
import { useDocumentTitle } from "@/hooks/use-document-title";
import { useSearchParams } from "react-router-dom";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { AuditSummaryBar } from "./audit-summary-bar";
import { ViolationsTab } from "./violations-tab";
import { SitesTab } from "./sites-tab";
import { RulesTab } from "./rules-tab";
import { HistoryTab } from "./history-tab";

const TABS = ["violations", "sites", "rules", "history"] as const;
type AuditTab = (typeof TABS)[number];

function parseTab(raw: string | null): AuditTab {
  if (raw && TABS.includes(raw as AuditTab)) return raw as AuditTab;
  return "violations";
}

export default function AdminAuditPage() {
  const { t } = useTranslation();
  useDocumentTitle(t("audit.title"));
  const [searchParams, setSearchParams] = useSearchParams();
  const currentTab = parseTab(searchParams.get("tab"));

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("audit.title")}</h1>
        <p className="text-muted-foreground">{t("audit.description")}</p>
      </div>
      <AuditSummaryBar />

      <Tabs
        value={currentTab}
        onValueChange={(value) => setSearchParams({ tab: value })}
      >
        <TabsList variant="line">
          <TabsTrigger value="violations">{t("audit.tabs.violations")}</TabsTrigger>
          <TabsTrigger value="sites">{t("audit.tabs.sites")}</TabsTrigger>
          <TabsTrigger value="rules">{t("audit.tabs.rules")}</TabsTrigger>
          <TabsTrigger value="history">{t("audit.tabs.history")}</TabsTrigger>
        </TabsList>

        <TabsContent value="violations" className="mt-4 space-y-4">
          <ViolationsTab />
        </TabsContent>
        <TabsContent value="sites" className="mt-4 space-y-4">
          <SitesTab />
        </TabsContent>
        <TabsContent value="rules" className="mt-4 space-y-4">
          <RulesTab />
        </TabsContent>
        <TabsContent value="history" className="mt-4 space-y-4">
          <HistoryTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
