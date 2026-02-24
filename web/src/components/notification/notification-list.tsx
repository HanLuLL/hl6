import { useRef, useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNotifications } from "@/hooks/use-notifications";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { NotificationDetailDialog } from "./notification-detail-dialog";
import type { Notification } from "@/types";

function formatRelativeTime(dateStr: string): string {
  const now = Date.now();
  const date = new Date(dateStr).getTime();
  const diff = now - date;
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d`;
  return new Date(dateStr).toLocaleDateString();
}

function TypeBadge({ type }: { type: Notification["type"] }) {
  const { t } = useTranslation();
  const variants: Record<string, "destructive" | "default" | "secondary"> = {
    urgent: "destructive",
    pinned: "default",
    normal: "secondary",
  };
  return (
    <Badge variant={variants[type] || "secondary"} className="text-[10px] px-1.5 py-0">
      {t(`notifications.type_${type}`)}
    </Badge>
  );
}

export function NotificationList() {
  const { t } = useTranslation();
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } = useNotifications();
  const [selectedNotification, setSelectedNotification] = useState<Notification | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);

  const observer = useRef<IntersectionObserver | null>(null);
  const lastItemRef = useCallback(
    (node: HTMLDivElement | null) => {
      if (isFetchingNextPage) return;
      if (observer.current) observer.current.disconnect();
      observer.current = new IntersectionObserver((entries) => {
        if (entries[0].isIntersecting && hasNextPage) {
          fetchNextPage();
        }
      });
      if (node) observer.current.observe(node);
    },
    [isFetchingNextPage, fetchNextPage, hasNextPage]
  );

  const notifications = data?.pages.flatMap((page) => page.data).filter(Boolean) ?? [];

  if (isLoading) {
    return (
      <div className="space-y-2 p-2">
        {[...Array(3)].map((_, i) => (
          <Skeleton key={i} className="h-14 w-full" />
        ))}
      </div>
    );
  }

  if (notifications.length === 0) {
    return (
      <div className="flex items-center justify-center p-6 text-sm text-muted-foreground">
        {t("notifications.empty")}
      </div>
    );
  }

  return (
    <>
      <div className="max-h-80 overflow-y-auto">
        {notifications.map((notification, index) => {
          const isLast = index === notifications.length - 1;
          return (
            <div
              key={notification.id}
              ref={isLast ? lastItemRef : undefined}
              onClick={() => {
                setSelectedNotification(notification);
                setDialogOpen(true);
              }}
              className={`flex items-start gap-2 px-3 py-2.5 cursor-pointer hover:bg-accent transition-colors ${
                !notification.is_read ? "bg-accent/50 border-l-2 border-primary" : ""
              }`}
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-1.5 mb-0.5">
                  <TypeBadge type={notification.type} />
                  <span className={`text-sm truncate ${!notification.is_read ? "font-semibold" : ""}`}>
                    {notification.title}
                  </span>
                </div>
                <p className="text-xs text-muted-foreground">
                  {formatRelativeTime(notification.created_at)}
                </p>
              </div>
              {!notification.is_read && (
                <div className="mt-1.5 h-2 w-2 rounded-full bg-primary shrink-0" />
              )}
            </div>
          );
        })}
        {isFetchingNextPage && (
          <div className="p-2">
            <Skeleton className="h-10 w-full" />
          </div>
        )}
      </div>
      <NotificationDetailDialog
        notification={selectedNotification}
        open={dialogOpen}
        onOpenChange={setDialogOpen}
      />
    </>
  );
}
