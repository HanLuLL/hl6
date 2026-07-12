import { Code2, GitBranch, GitCommitHorizontal, type LucideIcon } from "lucide-react";
import type { JSX } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { useBranding } from "@/hooks/use-branding";
import DOMPurify from 'dompurify';

const OPEN_SOURCE_URL = "https://github.com/HanLuLL/hl6";

type FooterBadgeProps = {
  icon: LucideIcon;
  label: string;
  value: string;
};

type SiteFooterProps = {
  withBorder?: boolean;
  centered?: boolean;
  className?: string;
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

export function SiteFooter({ withBorder = true, centered = false, className }: SiteFooterProps): JSX.Element {
  const { t } = useTranslation();
  const branding = useBranding();
  const branch = __APP_GIT_BRANCH__ || "unknown";
  const commit = __APP_GIT_COMMIT__ || "unknown";
  const showBranch = branch.toLowerCase() !== "unknown";
  const showCommit = commit.toLowerCase() !== "unknown";
  const icp = (branding as any)?.footer_icp;
  const icpLink = (branding as any)?.footer_icp_link;
  const footerContent = (branding as any)?.footer_content;

  return (
    <footer
      className={cn(
        "py-3 text-xs text-muted-foreground",
        withBorder ? "border-t px-4 lg:px-6" : "px-0",
        className,
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
          <span>{t("footer.openSource")}</span>
        </a>
        {showBranch && <FooterBadge icon={GitBranch} label={t("footer.branch")} value={branch} />}
        {showCommit && <FooterBadge icon={GitCommitHorizontal} label={t("footer.commit")} value={commit} />}
        {icp && (
          icpLink ? (
            <a href={icpLink} target="_blank" rel="noreferrer" className="transition-colors hover:text-foreground">
              {icp}
            </a>
          ) : (
            <span>{icp}</span>
          )
        )}
      </div>
      {footerContent && (
        <div
          className="mt-2 text-xs text-muted-foreground prose prose-xs prose-neutral dark:prose-invert max-w-none [&>*:first-child]:mt-0 [&>*:last-child]:mb-0"
          dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(footerContent) }}   // 114514 DOMPurify.sanitize
        />
      )}
    </footer>
  );
}