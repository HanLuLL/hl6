import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import type { Notification } from "@/types";

interface TypeBadgeProps {
  type: Notification["type"];
  compact?: boolean;
}

export function TypeBadge({ type, compact }: TypeBadgeProps) {
  const { t } = useTranslation();
  const variants: Record<string, "destructive" | "default" | "secondary"> = {
    urgent: "destructive",
    pinned: "default",
    normal: "secondary",
  };
  return (
    <Badge
      variant={variants[type] || "secondary"}
      className={compact ? "text-[10px] px-1.5 py-0" : undefined}
    >
      {t(`notifications.type_${type}`)}
    </Badge>
  );
}
