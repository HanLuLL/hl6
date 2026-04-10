import { useTranslation } from "react-i18next";
import { toast } from "sonner";

import { cn } from "@/lib/utils";

interface CopyableEmailProps {
  email?: string | null;
  className?: string;
  maxLen?: number;
  truncate?: boolean;
  stopPropagation?: boolean;
}

function truncateEmail(value: string, maxLen: number): string {
  if (value.length <= maxLen) {
    return value;
  }
  return `${value.slice(0, maxLen)}...`;
}

export function CopyableEmail({
  email,
  className,
  maxLen = 24,
  truncate = true,
  stopPropagation = false,
}: CopyableEmailProps) {
  const { t } = useTranslation();

  if (!email) {
    return <span className={cn("text-muted-foreground", className)}>-</span>;
  }

  return (
    <button
      type="button"
      title={email}
      className={cn(
        "cursor-copy text-left hover:underline",
        truncate ? "max-w-full truncate" : "break-all",
        className
      )}
      onClick={(e) => {
        if (stopPropagation) {
          e.stopPropagation();
        }
        navigator.clipboard.writeText(email).then(() => {
          toast.success(t("common.copied"));
        }).catch(() => {
          // Ignore clipboard errors in unsupported or insecure contexts.
        });
      }}
    >
      {truncate ? truncateEmail(email, maxLen) : email}
    </button>
  );
}
