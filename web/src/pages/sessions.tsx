import { useTranslation } from "react-i18next";
import { useSessions } from "@/hooks/use-sessions";
import { useDocumentTitle } from "@/hooks/use-document-title";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Monitor, Smartphone, MoreVertical, LogOut, Trash2 } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useState } from "react";

export default function SessionsPage() {
  const { t } = useTranslation();
  useDocumentTitle(t("sessions.title"));
  const { sessions, isLoading, terminateSession, isTerminating, terminateAll, isTerminatingAll } = useSessions();
  const [terminatingId, setTerminatingId] = useState<number | null>(null);

  const handleTerminate = (id: number) => {
    setTerminatingId(id);
    terminateSession(id);
  };

  const formatDateTime = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleString(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const getRelativeTime = (dateStr: string) => {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) return t("sessions.justNow");
    if (minutes < 60) return t("sessions.minutesAgo", { count: minutes });
    if (hours < 24) return t("sessions.hoursAgo", { count: hours });
    return t("sessions.daysAgo", { count: days });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("sessions.title")}</h1>
          <p className="text-muted-foreground">{t("sessions.subtitle")}</p>
        </div>
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button variant="destructive" disabled={isTerminatingAll}>
              <LogOut className="mr-2 h-4 w-4" />
              {t("sessions.logoutAll")}
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>{t("sessions.logoutAllConfirm")}</AlertDialogTitle>
              <AlertDialogDescription>{t("sessions.logoutAllDesc")}</AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
              <AlertDialogAction onClick={() => terminateAll()} disabled={isTerminatingAll}>
                {t("sessions.logoutAll")}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("sessions.activeDevices")}</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-4">
              {[1, 2, 3].map((i) => (
                <div key={i} className="flex items-center gap-4">
                  <Skeleton className="h-10 w-10 rounded-full" />
                  <div className="flex-1 space-y-2">
                    <Skeleton className="h-4 w-32" />
                    <Skeleton className="h-3 w-48" />
                  </div>
                </div>
              ))}
            </div>
          ) : sessions.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              {t("sessions.noActiveDevices")}
            </div>
          ) : (
            <div className="divide-y">
              {sessions.map((session) => (
                <div
                  key={session.id}
                  className="flex items-center justify-between py-4 first:pt-0 last:pb-0"
                >
                  <div className="flex items-center gap-4">
                    <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
                      {session.device_type === "native" ? (
                        <Smartphone className="h-5 w-5 text-muted-foreground" />
                      ) : (
                        <Monitor className="h-5 w-5 text-muted-foreground" />
                      )}
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium">{session.device_name}</span>
                        {session.is_current && (
                          <Badge variant="secondary">{t("sessions.currentDevice")}</Badge>
                        )}
                      </div>
                      <p className="text-sm text-muted-foreground">
                        {t("sessions.lastActive")}: {getRelativeTime(session.last_active_at)} · {t("sessions.expires")}: {formatDateTime(session.expires_at)}
                      </p>
                    </div>
                  </div>
                  {!session.is_current && (
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" disabled={isTerminating && terminatingId === session.id}>
                          <MoreVertical className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem
                          onClick={() => handleTerminate(session.id)}
                          className="text-destructive focus:text-destructive"
                        >
                          <Trash2 className="mr-2 h-4 w-4" />
                          {t("sessions.terminate")}
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  )}
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
