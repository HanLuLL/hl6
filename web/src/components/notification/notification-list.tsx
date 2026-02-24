import { useRef, useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNotifications } from "@/hooks/use-notifications";
import { Skeleton } from "@/components/ui/skeleton";
import { TypeBadge } from "./type-badge";
import { NotificationDetailDialog } from "./notification-detail-dialog";
import type { Notification } from "@/types";
import type { TFunction } from "i18next";

function formatRelativeTime(dateStr: string, t: TFunction): string {
  const now = Date.now();
  const date = new Date(dateStr).getTime();
  const diff = now - date;
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return t("notifications.justNow");
  if (minutes < 60) return t("notifications.minutesAgo", { count: minutes });
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return t("notifications.hoursAgo", { count: hours });
  const days = Math.floor(hours / 24);
  if (days < 30) return t("notifications.daysAgo", { count: days });
  return new Date(dateStr).toLocaleDateString();
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
                  <TypeBadge type={notification.type} compact />
                  <span className={`text-sm truncate ${!notification.is_read ? "font-semibold" : ""}`}>
                    {notification.title}
                  </span>
                </div>
                <p className="text-xs text-muted-foreground">
                  {formatRelativeTime(notification.created_at, t)}
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
