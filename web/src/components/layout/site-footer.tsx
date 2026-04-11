import { Code2, GitBranch, GitCommitHorizontal, type LucideIcon } from "lucide-react";
import type { JSX } from "react";
import { cn } from "@/lib/utils";

const OPEN_SOURCE_URL = "https://git.houlang.cloud/houlangcloud/hl6";

type FooterBadgeProps = {
  icon: LucideIcon;
  label: string;
  value: string;
};

type SiteFooterProps = {
  withBorder?: boolean;
  centered?: boolean;
};

function FooterBadge({ icon: Icon, label, value }: FooterBadgeProps): JSX.Element {
  return (
    <span className="inline-flex items-center gap-1.5">
      <Icon className="h-3.5 w-3.5" />
      <span className="text-[11px] uppercase tracking-wide">{label}</span>
      <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-[11px] text-foreground/80">
        {value}
      </code>
    </span>
  );
}

export function SiteFooter({ withBorder = true, centered = false }: SiteFooterProps): JSX.Element {
  const branch = __APP_GIT_BRANCH__ || "unknown";
  const commit = __APP_GIT_COMMIT__ || "unknown";
  const showBranch = branch.toLowerCase() !== "unknown";
  const showCommit = commit.toLowerCase() !== "unknown";

  return (
    <footer
      className={cn(
        "px-4 py-3 text-xs text-muted-foreground lg:px-6",
        withBorder && "border-t",
      )}
    >
      <div
        className={cn(
          "flex flex-wrap items-center gap-x-4 gap-y-2",
          centered ? "justify-center" : "justify-start",
        )}
      >
        <a
          href={OPEN_SOURCE_URL}
          target="_blank"
          rel="noreferrer"
          className="inline-flex items-center gap-1.5 transition-colors hover:text-foreground"
        >
          <Code2 className="h-3.5 w-3.5" />
          <span>厚浪开源</span>
        </a>
        {showBranch && <FooterBadge icon={GitBranch} label="Branch" value={branch} />}
        {showCommit && <FooterBadge icon={GitCommitHorizontal} label="Commit" value={commit} />}
      </div>
    </footer>
  );
}
