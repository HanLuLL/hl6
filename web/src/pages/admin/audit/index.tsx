import { useTranslation } from "react-i18next";
import { useDocumentTitle } from "@/hooks/use-document-title";
import { useSearchParams } from "react-router-dom";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { AuditSummaryBar } from "./audit-summary-bar";
import { SubdomainsTab } from "./subdomains-tab";
import { RulesTab } from "./rules-tab";
import { HistoryTab } from "./history-tab";
import { AIModelsTab } from "./ai-models-tab";
import { AITemplatesTab } from "./ai-templates-tab";
import { AIReviewsTab } from "./ai-reviews-tab";
import { AppealsTab } from "./appeals-tab";

const TABS = [
  "domains",
  "rules",
  "history",
  "ai-models",
  "ai-templates",
  "ai-reviews",
  "appeals",
] as const;
type AuditTab = (typeof TABS)[number];

function parseTab(raw: string | null): AuditTab {
  // 兼容旧路由重定向：violations/sites → domains
  if (raw === "violations" || raw === "sites") return "domains";
  // 兼容旧 /admin/ai-audit 入口
  if (raw === "models") return "ai-models";
  if (raw === "templates") return "ai-templates";
  if (raw === "reviews") return "ai-reviews";
  if (raw && TABS.includes(raw as AuditTab)) return raw as AuditTab;
  return "domains";
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
        {/* 7 Tab 在窄屏（Android）会溢出，使用横向滚动；宽屏居中 */}
        <TabsList variant="line" className="max-w-full overflow-x-auto whitespace-nowrap">
          <TabsTrigger value="domains">{t("audit.tabs.domains")}</TabsTrigger>
          <TabsTrigger value="rules">{t("audit.tabs.rules")}</TabsTrigger>
          <TabsTrigger value="history">{t("audit.tabs.history")}</TabsTrigger>
          <TabsTrigger value="ai-models">{t("audit.tabs.aiModels")}</TabsTrigger>
          <TabsTrigger value="ai-templates">{t("audit.tabs.aiTemplates")}</TabsTrigger>
          <TabsTrigger value="ai-reviews">{t("audit.tabs.aiReviews")}</TabsTrigger>
          <TabsTrigger value="appeals">{t("audit.tabs.appeals")}</TabsTrigger>
        </TabsList>

        <TabsContent value="domains" className="mt-4 space-y-4">
          <SubdomainsTab />
        </TabsContent>
        <TabsContent value="rules" className="mt-4 space-y-4">
          <RulesTab />
        </TabsContent>
        <TabsContent value="history" className="mt-4 space-y-4">
          <HistoryTab />
        </TabsContent>
        <TabsContent value="ai-models" className="mt-4 space-y-4">
          <AIModelsTab />
        </TabsContent>
        <TabsContent value="ai-templates" className="mt-4 space-y-4">
          <AITemplatesTab />
        </TabsContent>
        <TabsContent value="ai-reviews" className="mt-4 space-y-4">
          <AIReviewsTab />
        </TabsContent>
        <TabsContent value="appeals" className="mt-4 space-y-4">
          <AppealsTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
