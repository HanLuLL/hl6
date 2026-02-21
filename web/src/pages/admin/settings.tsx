import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { toast } from "sonner";

export default function AdminSettingsPage() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  const { data: config, isLoading } = useQuery({
    queryKey: ["admin-config"],
    queryFn: async () => {
      const res = await api.adminGetConfig();
      return res.data;
    },
  });

  const [bonusCredits, setBonusCredits] = useState("0");

  useEffect(() => {
    if (config) {
      setBonusCredits(config.registration_bonus_credits ?? "0");
    }
  }, [config]);

  const updateMutation = useMutation({
    mutationFn: api.adminUpdateConfig,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-config"] });
      toast.success(t("adminSettings.saved"));
    },
    onError: (err) => toast.error(err.message),
  });

  if (isLoading) {
    return <div className="flex items-center justify-center py-12"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" /></div>;
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("adminSettings.title")}</h1>
        <p className="text-muted-foreground">{t("adminSettings.subtitle")}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("adminSettings.registrationBonus")}</CardTitle>
          <p className="text-sm text-muted-foreground">{t("adminSettings.registrationBonusDesc")}</p>
        </CardHeader>
        <CardContent>
          <div className="flex items-end gap-4">
            <div className="space-y-2 flex-1 max-w-xs">
              <Label>{t("adminSettings.registrationBonus")}</Label>
              <Input
                type="number"
                min="0"
                value={bonusCredits}
                onChange={(e) => setBonusCredits(e.target.value)}
              />
            </div>
            <Button
              onClick={() => updateMutation.mutate({ registration_bonus_credits: bonusCredits })}
              disabled={updateMutation.isPending}
            >
              {updateMutation.isPending ? t("common.saving") : t("common.save")}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
