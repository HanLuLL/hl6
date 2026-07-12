import { useTranslation } from "react-i18next";
import { useFriendLinks } from "@/hooks/use-friend-links";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useDocumentTitle } from "@/hooks/use-document-title";
import { ExternalLink, Link2 } from "lucide-react";

export default function FriendLinksPage() {
  const { data: links, isLoading } = useFriendLinks();
  const { t } = useTranslation();
  useDocumentTitle(t("friendLinks.title"));

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("friendLinks.title")}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t("friendLinks.subtitle")}</p>
      </div>

      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {[...Array(3)].map((_, i) => (
            <Card key={i}>
              <CardContent className="flex items-center gap-4 p-4">
                <Skeleton className="h-12 w-12 rounded-lg" />
                <div className="flex-1 space-y-2">
                  <Skeleton className="h-4 w-24" />
                  <Skeleton className="h-3 w-full" />
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : !links || links.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-center">
          <div className="rounded-xl bg-muted p-4 mb-4">
            <Link2 className="h-8 w-8 text-muted-foreground/50" />
          </div>
          <p className="text-muted-foreground text-sm">{t("friendLinks.noLinks")}</p>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {links.map((link) => (
            <a
              key={link.id}
              href={link.url}
              target="_blank"
              rel="noopener noreferrer"
              className="block transition-transform duration-150 hover:-translate-y-px"
            >
              <Card className="h-full overflow-hidden transition-colors hover:border-primary/40">
                <CardContent className="flex items-start gap-4 p-4">
                  {link.logo_url ? (
                    <img
                      src={link.logo_url}
                      alt={link.name}
                      className="h-12 w-12 rounded-lg object-cover"
                      onError={(e) => {
                        (e.target as HTMLImageElement).style.display = "none";
                      }}
                    />
                  ) : (
                    <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-muted">
                      <Link2 className="h-5 w-5 text-muted-foreground" />
                    </div>
                  )}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-1.5">
                      <h3 className="font-semibold truncate">{link.name}</h3>
                      <ExternalLink className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                    </div>
                    {link.description && (
                      <p className="mt-1 text-sm text-muted-foreground line-clamp-2">{link.description}</p>
                    )}
                  </div>
                </CardContent>
              </Card>
            </a>
          ))}
        </div>
      )}
    </div>
  );
}
