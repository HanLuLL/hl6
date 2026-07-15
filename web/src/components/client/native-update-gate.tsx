import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { api } from "@/lib/api";
import { clientVersion, isNativeClient } from "@/lib/client-runtime";
import { openNativeExternalUrl } from "@/lib/native-client";
import type { ClientVersionConfig } from "@/types";

export function NativeUpdateGate() {
  const [update, setUpdate] = useState<ClientVersionConfig | null>(null);
  const [dismissed, setDismissed] = useState(false);
  const [checking, setChecking] = useState(isNativeClient && Boolean(clientVersion));

  useEffect(() => {
    if (!isNativeClient || !clientVersion) return;
    let mounted = true;

    void api.getClientVersion(clientVersion)
      .then((response) => {
        if (!mounted || !response.data.update_available) return;
        setUpdate(response.data);
      })
      .catch(() => {
        // A temporary version-check failure must not make a non-forced client unusable.
      })
      .finally(() => {
        if (mounted) setChecking(false);
      });

    return () => {
      mounted = false;
    };
  }, []);

  if (!checking && (!update || dismissed)) return null;

  const forceUpdate = checking || Boolean(update?.force_update);
  const canUpdate = Boolean(update?.update_url.startsWith("https://"));

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !forceUpdate) setDismissed(true);
      }}
    >
      <DialogContent
        showCloseButton={!forceUpdate}
        showHotkeyGuide={false}
        enableHotkeys={false}
        onEscapeKeyDown={(event) => {
          if (forceUpdate) event.preventDefault();
        }}
        onPointerDownOutside={(event) => {
          if (forceUpdate) event.preventDefault();
        }}
      >
        <DialogHeader>
          <DialogTitle>{checking ? "正在检查版本" : "发现新版本"}</DialogTitle>
          <DialogDescription>
            {checking
              ? "正在从服务端获取客户端更新策略。"
              : forceUpdate
                ? "当前版本必须更新后才能继续使用。"
                : "更新后可获得最新功能与修复。"}
          </DialogDescription>
        </DialogHeader>
        {update?.update_notice && <p className="whitespace-pre-wrap text-sm text-foreground">{update.update_notice}</p>}
        <DialogFooter>
          {!checking && !forceUpdate && <Button variant="outline" onClick={() => setDismissed(true)}>稍后更新</Button>}
          {!checking && <Button disabled={!canUpdate} onClick={() => update && void openNativeExternalUrl(update.update_url)}>立即更新</Button>}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
