import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api, getErrorMessage } from "@/lib/api";
import { useAuth } from "@/hooks/use-auth";
import { useDocumentTitle } from "@/hooks/use-document-title";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { User as UserIcon } from "lucide-react";

export default function ProfilePage() {
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  useDocumentTitle(t("profile.title"));

  const [name, setName] = useState(user?.name ?? "");
  const [avatarUrl, setAvatarUrl] = useState(user?.avatar_url ?? "");
  const [bio, setBio] = useState((user as any)?.bio ?? "");
  const [website, setWebsite] = useState((user as any)?.website ?? "");

  const mutation = useMutation({
    mutationFn: (data: { name?: string; avatar_url?: string; bio?: string; website?: string }) =>
      api.updateProfile(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["me"] });
      toast.success(t("profile.saved"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const handleSave = () => {
    const data: Record<string, string> = {};
    if (name !== (user?.name ?? "")) data.name = name;
    if (avatarUrl !== (user?.avatar_url ?? "")) data.avatar_url = avatarUrl;
    if (bio !== ((user as any)?.bio ?? "")) data.bio = bio;
    if (website !== ((user as any)?.website ?? "")) data.website = website;
    if (Object.keys(data).length === 0) return;
    mutation.mutate(data);
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("profile.title")}</h1>
        <p className="text-muted-foreground">{t("profile.subtitle")}</p>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center gap-4">
            <Avatar className="h-16 w-16">
              <AvatarImage src={avatarUrl} />
              <AvatarFallback className="text-lg">{name?.charAt(0)?.toUpperCase() || "U"}</AvatarFallback>
            </Avatar>
            <div>
              <CardTitle className="text-base">{user?.name}</CardTitle>
              <p className="text-sm text-muted-foreground">{user?.email}</p>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 lg:grid-cols-2">
            <div className="space-y-2">
              <Label>{t("profile.name")}</Label>
              <Input value={name} onChange={(e) => setName(e.target.value)} placeholder={t("profile.namePlaceholder")} />
            </div>
            <div className="space-y-2">
              <Label>{t("profile.avatarUrl")}</Label>
              <Input value={avatarUrl} onChange={(e) => setAvatarUrl(e.target.value)} placeholder="https://example.com/avatar.png" />
            </div>
          </div>
          <div className="space-y-2">
            <Label>{t("profile.website")}</Label>
            <Input value={website} onChange={(e) => setWebsite(e.target.value)} placeholder="https://example.com" />
          </div>
          <div className="space-y-2">
            <Label>{t("profile.bio")}</Label>
            <Textarea value={bio} onChange={(e) => setBio(e.target.value)} placeholder={t("profile.bioPlaceholder")} rows={3} />
          </div>
          <div className="flex justify-end">
            <Button onClick={handleSave} disabled={mutation.isPending}>
              {mutation.isPending ? t("common.saving") : t("common.save")}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
