import { useTranslation } from "react-i18next";
import { useDocumentTitle } from "@/hooks/use-document-title";
import { useSearchParams } from "react-router-dom";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { AuditTab } from "./audit-tab";
import { EmailTab } from "./email-tab";
import { SystemTab } from "./system-tab";

const TABS = ["audit", "email", "system"] as const;
type LogsTab = (typeof TABS)[number];

function parseTab(raw: string | null): LogsTab {
  if (raw && TABS.includes(raw as LogsTab)) return raw as LogsTab;
  return "audit";
}

export default function AdminLogsPage() {
  const { t } = useTranslation();
  useDocumentTitle(t("logsCenter.title"));
  const [searchParams, setSearchParams] = useSearchParams();
  const currentTab = parseTab(searchParams.get("tab"));

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("logsCenter.title")}</h1>
        <p className="text-muted-foreground">{t("logsCenter.description")}</p>
      </div>

      <Tabs
        value={currentTab}
        onValueChange={(value) => setSearchParams({ tab: value })}
      >
        {/* 窄屏横向滚动，宽屏居中 */}
        <TabsList variant="line" className="max-w-full overflow-x-auto whitespace-nowrap">
          <TabsTrigger value="audit">{t("logsCenter.tabs.audit")}</TabsTrigger>
          <TabsTrigger value="email">{t("logsCenter.tabs.email")}</TabsTrigger>
          <TabsTrigger value="system">{t("logsCenter.tabs.system")}</TabsTrigger>
        </TabsList>

        <TabsContent value="audit" className="mt-4 space-y-4">
          <AuditTab />
        </TabsContent>
        <TabsContent value="email" className="mt-4 space-y-4">
          <EmailTab />
        </TabsContent>
        <TabsContent value="system" className="mt-4 space-y-4">
          <SystemTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
