import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import {
  AlertCircle,
  Download,
  Eye,
  Search,
  AlertTriangle,
} from "lucide-react";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useDocumentTitle } from "@/hooks/use-document-title";
import type { SystemLog } from "@/types";

function downloadBlob(blob: Blob, filename: string) {
  const href = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = href;
  anchor.download = filename;
  anchor.click();
  window.setTimeout(() => URL.revokeObjectURL(href), 0);
}

const logLevels = [
  { value: "", labelKey: "systemLogs.allLevels" },
  { value: "DEBUG", labelKey: "DEBUG" },
  { value: "INFO", labelKey: "INFO" },
  { value: "WARN", labelKey: "WARN" },
  { value: "ERROR", labelKey: "ERROR" },
];

function LevelBadge({ level }: { level: string }) {
  const variants: Record<string, string> = {
    DEBUG: "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300",
    INFO: "bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300",
    WARN: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300",
    ERROR: "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300",
  };
  return (
    <Badge className={variants[level] || variants.INFO}>
      {level}
    </Badge>
  );
}

function LogDetailDialog({
  log,
  open,
  onClose,
}: {
  log: SystemLog | null;
  open: boolean;
  onClose: () => void;
}) {
  const { t } = useTranslation();

  if (!log) return null;

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <LevelBadge level={log.level} />
            <span>{log.module}</span>
          </DialogTitle>
          <DialogDescription>
            {t("systemLogs.logId")}: {log.id} · {new Date(log.created_at).toLocaleString()}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div>
            <Label className="text-muted-foreground">{t("systemLogs.message")}</Label>
            <p className="mt-1 text-sm">{log.message}</p>
          </div>
          {log.fields && Object.keys(log.fields).length > 0 && (
            <div>
              <Label className="text-muted-foreground">{t("systemLogs.fields")}</Label>
              <pre className="mt-1 p-3 bg-muted rounded-md text-xs overflow-x-auto">
                {JSON.stringify(log.fields, null, 2)}
              </pre>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function StatsSkeleton() {
  return (
    <div className="grid gap-4 md:grid-cols-4">
      {[...Array(4)].map((_, i) => (
        <Card key={i}>
          <CardHeader className="pb-2">
            <Skeleton className="h-4 w-24" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-8 w-16" />
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

function TableSkeleton() {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead><Skeleton className="h-4 w-16" /></TableHead>
          <TableHead><Skeleton className="h-4 w-12" /></TableHead>
          <TableHead><Skeleton className="h-4 w-24" /></TableHead>
          <TableHead><Skeleton className="h-4 w-32" /></TableHead>
          <TableHead><Skeleton className="h-4 w-24" /></TableHead>
          <TableHead><Skeleton className="h-4 w-16" /></TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {[...Array(5)].map((_, i) => (
          <TableRow key={i}>
            <TableCell><Skeleton className="h-4 w-16" /></TableCell>
            <TableCell><Skeleton className="h-4 w-12" /></TableCell>
            <TableCell><Skeleton className="h-4 w-24" /></TableCell>
            <TableCell><Skeleton className="h-4 w-48" /></TableCell>
            <TableCell><Skeleton className="h-4 w-24" /></TableCell>
            <TableCell><Skeleton className="h-8 w-8" /></TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

export default function AdminSystemLogsPage() {
  const { t } = useTranslation();
  useDocumentTitle(t("systemLogs.title"));

  const [page, setPage] = useState(1);
  const [levelFilter, setLevelFilter] = useState("");
  const [moduleFilter, setModuleFilter] = useState("");
  const [search, setSearch] = useState("");
  const [fromDate, setFromDate] = useState("");
  const [toDate, setToDate] = useState("");
  const [selectedLog, setSelectedLog] = useState<SystemLog | null>(null);
  const [exportFormat, setExportFormat] = useState<"json" | "txt">("json");
  const perPage = 20;

  // Fetch stats
  const { data: stats, isLoading: statsLoading } = useQuery({
    queryKey: ["system-log-stats"],
    queryFn: async () => (await api.adminGetSystemLogStats()).data,
    staleTime: 30_000,
  });

  // Fetch modules
  const { data: modules } = useQuery({
    queryKey: ["system-log-modules"],
    queryFn: async () => (await api.adminGetSystemLogModules()).data,
    staleTime: 60_000,
  });

  // Fetch logs
  const { data, isLoading } = useQuery({
    queryKey: ["system-logs", page, levelFilter, moduleFilter, search, fromDate, toDate],
    queryFn: async () => {
      const params: Record<string, string | number | undefined> = {
        page,
        per_page: perPage,
      };
      if (levelFilter) params.level = levelFilter;
      if (moduleFilter) params.module = moduleFilter;
      if (search) params.search = search;
      if (fromDate) params.from = new Date(fromDate).toISOString();
      if (toDate) params.to = new Date(toDate).toISOString();
      return api.adminListSystemLogs(params);
    },
  });

  const handleExport = async () => {
    try {
      const params: Record<string, string | undefined> = { format: exportFormat };
      if (levelFilter) params.level = levelFilter;
      if (moduleFilter) params.module = moduleFilter;
      if (search) params.search = search;
      if (fromDate) params.from = new Date(fromDate).toISOString();
      if (toDate) params.to = new Date(toDate).toISOString();
      const { blob, filename } = await api.adminExportSystemLogs(params);
      downloadBlob(blob, filename);
    } catch (err) {
      // Error handled by toast
    }
  };

  const logs = (data?.data ?? []) as SystemLog[];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / perPage);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("systemLogs.title")}</h1>
        <p className="text-muted-foreground">{t("systemLogs.subtitle")}</p>
      </div>

      {/* Stats */}
      {statsLoading ? (
        <StatsSkeleton />
      ) : stats ? (
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                {t("systemLogs.totalLogs")}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.total}</div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                {t("systemLogs.todayLogs")}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.today}</div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-1">
                <AlertCircle className="h-4 w-4 text-red-500" />
                {t("systemLogs.errorLogs")}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-red-500">{stats.level_ERROR ?? 0}</div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-1">
                <AlertTriangle className="h-4 w-4 text-yellow-500" />
                {t("systemLogs.warnLogs")}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-yellow-500">{stats.level_WARN ?? 0}</div>
            </CardContent>
          </Card>
        </div>
      ) : null}

      {/* Filters */}
      <Card>
        <CardContent className="pt-6">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-6">
            <div className="space-y-2">
              <Label>{t("systemLogs.level")}</Label>
              <Select value={levelFilter} onValueChange={setLevelFilter}>
                <SelectTrigger>
                  <SelectValue placeholder={t("systemLogs.allLevels")} />
                </SelectTrigger>
                <SelectContent>
                  {logLevels.map((l) => (
                    <SelectItem key={l.value} value={l.value}>
                      {l.value || t("systemLogs.allLevels")}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>{t("systemLogs.module")}</Label>
              <Select value={moduleFilter} onValueChange={setModuleFilter}>
                <SelectTrigger>
                  <SelectValue placeholder={t("systemLogs.allModules")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">{t("systemLogs.allModules")}</SelectItem>
                  {(modules ?? []).map((m) => (
                    <SelectItem key={m} value={m}>
                      {m}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>{t("systemLogs.search")}</Label>
              <div className="relative">
                <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder={t("systemLogs.searchPlaceholder")}
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="pl-8"
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label>{t("systemLogs.from")}</Label>
              <Input
                type="date"
                value={fromDate}
                onChange={(e) => setFromDate(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("systemLogs.to")}</Label>
              <Input
                type="date"
                value={toDate}
                onChange={(e) => setToDate(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("systemLogs.export")}</Label>
              <div className="flex gap-2">
                <Select value={exportFormat} onValueChange={(v) => setExportFormat(v as "json" | "txt")}>
                  <SelectTrigger className="w-24">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="json">JSON</SelectItem>
                    <SelectItem value="txt">TXT</SelectItem>
                  </SelectContent>
                </Select>
                <Button variant="outline" size="icon" onClick={handleExport}>
                  <Download className="h-4 w-4" />
                </Button>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Table */}
      <Card>
        <CardContent className="p-0">
          {isLoading ? (
            <TableSkeleton />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("systemLogs.level")}</TableHead>
                  <TableHead>{t("systemLogs.module")}</TableHead>
                  <TableHead>{t("systemLogs.message")}</TableHead>
                  <TableHead>{t("systemLogs.time")}</TableHead>
                  <TableHead className="w-16">{t("systemLogs.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {logs.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                      {t("systemLogs.noLogs")}
                    </TableCell>
                  </TableRow>
                ) : (
                  logs.map((log) => (
                    <TableRow key={log.id}>
                      <TableCell>
                        <LevelBadge level={log.level} />
                      </TableCell>
                      <TableCell className="font-medium">{log.module}</TableCell>
                      <TableCell className="max-w-md truncate">{log.message}</TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {new Date(log.created_at).toLocaleString()}
                      </TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => setSelectedLog(log)}
                        >
                          <Eye className="h-4 w-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex justify-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
          >
            {t("common.previous")}
          </Button>
          <span className="flex items-center text-sm text-muted-foreground">
            {t("common.pageOf", { page, total: totalPages })}
          </span>
          <Button
            variant="outline"
            size="sm"
            disabled={page >= totalPages}
            onClick={() => setPage((p) => p + 1)}
          >
            {t("common.next")}
          </Button>
        </div>
      )}

      {/* Detail Dialog */}
      <LogDetailDialog
        log={selectedLog}
        open={selectedLog !== null}
        onClose={() => setSelectedLog(null)}
      />
    </div>
  );
}