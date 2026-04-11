import { useTranslation } from "react-i18next";
import { useUnreadStatus } from "@/hooks/use-notifications";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Button } from "@/components/ui/button";
import { NotificationList } from "./notification-list";

export function NotificationBell() {
  const { t } = useTranslation();
  const { data: hasUnread } = useUnreadStatus();

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="ghost" size="icon" className="relative h-8 w-8" aria-label="Notifications">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            className={hasUnread ? "text-yellow-500 animate-wobble" : ""}
          >
            <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
            <path d="M13.73 21a2 2 0 0 1-3.46 0" />
          </svg>
          {hasUnread && (
            <span className="absolute top-0.5 right-0.5 h-2 w-2 rounded-full bg-yellow-500" />
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-80 p-0" align="end">
        <div className="flex items-center justify-between px-3 py-2 border-b">
          <h4 className="text-sm font-semibold">{t("notifications.title")}</h4>
        </div>
        <NotificationList />
      </PopoverContent>
    </Popover>
  );
}
