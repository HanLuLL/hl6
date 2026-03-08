import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api, getErrorMessage } from "@/lib/api";
import { cacheBranding } from "@/hooks/use-branding";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";

export function BrandContent() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [name, setName] = useState("");

  const { data: branding, isLoading } = useQuery({
    queryKey: ["branding-admin"],
    queryFn: async () => {
      const res = await api.getBranding();
      return res.data;
    },
    staleTime: 30_000,
  });

  useEffect(() => {
    if (branding) {
      setName(branding.name);
    }
  }, [branding]);

  const updateNameMutation = useMutation({
    mutationFn: api.adminUpdateBranding,
    onSuccess: (res) => {
      cacheBranding(res.data);
      queryClient.setQueryData(["branding-admin"], res.data);
      toast.success(t("adminBrand.nameSaved"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const uploadLogoMutation = useMutation({
    mutationFn: api.adminUploadBrandingLogo,
    onSuccess: (res) => {
      cacheBranding(res.data);
      queryClient.setQueryData(["branding-admin"], res.data);
      toast.success(t("adminBrand.logoUploaded"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const canSaveName = name.trim().length > 0 && !updateNameMutation.isPending;

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold">{t("adminBrand.title")}</h2>
        <p className="text-sm text-muted-foreground">{t("adminBrand.subtitle")}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("adminBrand.nameLabel")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {isLoading ? (
            <Skeleton className="h-9 w-full max-w-sm" />
          ) : (
            <>
              <Input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t("adminBrand.namePlaceholder")}
                className="max-w-sm"
              />
              <Button
                onClick={() => updateNameMutation.mutate({ name: name.trim() })}
                disabled={!canSaveName}
              >
                {updateNameMutation.isPending ? t("common.saving") : t("adminBrand.saveName")}
              </Button>
            </>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("adminBrand.logoLabel")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-center gap-6">
            <div className="space-y-2">
              <Label>{t("adminBrand.currentLogo")}</Label>
              <div className="h-14 w-14 rounded-md border bg-muted/20 p-2">
                {branding?.logo_url ? (
                  <img src={branding.logo_url} alt={branding.name} className="h-full w-full object-contain" />
                ) : (
                  <div className="flex h-full w-full items-center justify-center text-xs text-muted-foreground">-</div>
                )}
              </div>
            </div>
            <div className="space-y-2">
              <Label>{t("adminBrand.currentFavicon")}</Label>
              <div className="h-10 w-10 rounded-md border bg-muted/20 p-2">
                {branding?.favicon_url ? (
                  <img src={branding.favicon_url} alt="favicon" className="h-full w-full object-contain" />
                ) : (
                  <div className="flex h-full w-full items-center justify-center text-xs text-muted-foreground">-</div>
                )}
              </div>
            </div>
          </div>

          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            className="hidden"
            onChange={(e) => {
              const file = e.target.files?.[0];
              if (!file) {
                return;
              }
              uploadLogoMutation.mutate(file);
              e.target.value = "";
            }}
          />

          <Button
            onClick={() => fileInputRef.current?.click()}
            disabled={uploadLogoMutation.isPending}
          >
            {uploadLogoMutation.isPending ? t("common.saving") : t("adminBrand.uploadLogo")}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
