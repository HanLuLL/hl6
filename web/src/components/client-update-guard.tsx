import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";

const CLIENT_VERSION = import.meta.env.VITE_CLIENT_BUILD_VERSION?.trim();

function compareVersions(current: string, latest: string) {
  const currentParts = current.split(/[.+-]/).map((part) => Number.parseInt(part, 10) || 0);
  const latestParts = latest.split(/[.+-]/).map((part) => Number.parseInt(part, 10) || 0);
  const length = Math.max(currentParts.length, latestParts.length);
  for (let index = 0; index < length; index += 1) {
    if ((currentParts[index] ?? 0) !== (latestParts[index] ?? 0)) {
      return (currentParts[index] ?? 0) - (latestParts[index] ?? 0);
    }
  }
  return 0;
}

export function ClientUpdateGuard() {
  const [update, setUpdate] = useState<{ latestVersion: string; notice: string; url: string; forced: boolean } | null>(null);

  useEffect(() => {
    if (!CLIENT_VERSION) return;
    void api.getClientVersion().then((res) => {
      const config = res.data;
      if (config.latest_version && compareVersions(CLIENT_VERSION, config.latest_version) < 0) {
        setUpdate({
          latestVersion: config.latest_version,
          notice: config.update_notice,
          url: config.update_url,
          forced: config.force_update,
        });
      }
    });
  }, []);

  if (!update) return null;

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center bg-background/95 p-4">
      <div className="w-full max-w-md space-y-4 rounded-lg border bg-card p-6 shadow-lg">
        <div>
          <h1 className="text-lg font-semibold">客户端有可用更新</h1>
          <p className="mt-1 text-sm text-muted-foreground">最新版本：{update.latestVersion}</p>
        </div>
        {update.notice && <p className="whitespace-pre-wrap text-sm">{update.notice}</p>}
        <div className="flex justify-end gap-2">
          {!update.forced && <Button variant="outline" onClick={() => setUpdate(null)}>稍后更新</Button>}
          {update.url && <Button onClick={() => { window.location.href = update.url; }}>立即更新</Button>}
        </div>
      </div>
    </div>
  );
}
