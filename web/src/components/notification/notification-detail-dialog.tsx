import { useTranslation } from "react-i18next";
import DOMPurify from "dompurify";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { TypeBadge } from "./type-badge";
import { useMarkRead } from "@/hooks/use-notifications";
import type { Notification } from "@/types";

const ALLOWED_TAGS = ["p", "br", "strong", "em", "u", "s", "h2", "h3", "ul", "ol", "li", "a", "img", "span"];
const ALLOWED_ATTR = ["href", "src", "alt", "target", "style"];

DOMPurify.addHook("afterSanitizeAttributes", (node) => {
  if (node.tagName === "A") {
    node.setAttribute("rel", "noopener noreferrer");
  }
});

function sanitize(html: string) {
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS,
    ALLOWED_ATTR,
    ALLOW_DATA_ATTR: false,
  });
}

interface NotificationDetailDialogProps {
  notification: Notification | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function NotificationDetailDialog({ notification, open, onOpenChange }: NotificationDetailDialogProps) {
  const { t } = useTranslation();
  const markRead = useMarkRead();

  if (!notification) return null;

  const handleMarkRead = () => {
    markRead.mutate(notification.id, {
      onSuccess: () => onOpenChange(false),
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg max-h-[80vh] flex flex-col">
        <DialogHeader>
          <div className="flex items-center gap-2">
            <TypeBadge type={notification.type} />
            <DialogTitle className="text-lg">{notification.title}</DialogTitle>
          </div>
          <p className="text-xs text-muted-foreground">
            {new Date(notification.created_at).toLocaleString()}
          </p>
        </DialogHeader>
        <div
          className="flex-1 overflow-y-auto prose prose-sm dark:prose-invert max-w-none"
          dangerouslySetInnerHTML={{ __html: sanitize(notification.content) }}
        />
        {!notification.is_read && (
          <div className="flex justify-end pt-2 border-t">
            <Button
              size="sm"
              onClick={handleMarkRead}
              disabled={markRead.isPending}
            >
              {markRead.isPending ? t("common.loading") : t("notifications.markRead")}
            </Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
