import { useCallback, useEffect, useRef, useState } from "react";
import { RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { api, getErrorMessage } from "@/lib/api";

interface CaptchaWidgetProps {
  /**
   * 父表单提交时调用：通过 ref 暴露当前验证码 ID 和用户输入的验证码。
   * 如果验证码未启用，返回 { captchaId: "", captchaCode: "" }。
   */
  onChange: (value: { captchaId: string; captchaCode: string }) => void;
  /** 验证码错误时由父组件触发清空（可选） */
  resetSignal?: number;
}

interface CaptchaData {
  captchaId: string;
  image: string;
  enabled: boolean;
}

/**
 * 自托管图形验证码 widget。
 *
 * - 进入页面时调用 GET /auth/captcha 获取验证码；
 * - 如果后端返回 enabled=false，本组件不渲染任何 UI；
 * - 用户点击「换一张」或验证码过期后可手动刷新；
 * - 父组件通过 onChange 拿到 captchaId + captchaCode，提交时一并传给后端。
 */
export function CaptchaWidget({ onChange, resetSignal }: CaptchaWidgetProps) {
  const { t } = useTranslation();
  const [data, setData] = useState<CaptchaData | null>(null);
  const [loading, setLoading] = useState(false);
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const abortRef = useRef<AbortController | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError("");
    if (abortRef.current) {
      abortRef.current.abort();
    }
    const controller = new AbortController();
    abortRef.current = controller;
    try {
      const res = await api.getCaptcha();
      if (controller.signal.aborted) return;
      const payload = res.data ?? { captcha_id: "", image: "", enabled: false, ttl_seconds: 0 };
      const next: CaptchaData = {
        captchaId: payload.captcha_id,
        image: payload.image,
        enabled: payload.enabled,
      };
      setData(next);
      setCode("");
      if (!next.enabled) {
        onChange({ captchaId: "", captchaCode: "" });
      } else {
        onChange({ captchaId: next.captchaId, captchaCode: "" });
      }
    } catch (err) {
      if (controller.signal.aborted) return;
      setError(getErrorMessage(err, t));
    } finally {
      if (!controller.signal.aborted) {
        setLoading(false);
      }
    }
  }, [onChange, t]);

  useEffect(() => {
    void refresh();
    return () => {
      if (abortRef.current) {
        abortRef.current.abort();
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 父组件触发重置（例如提交失败后清空验证码并刷新一张新的）
  useEffect(() => {
    if (resetSignal === undefined || resetSignal === 0) return;
    void refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [resetSignal]);

  if (!data || !data.enabled) {
    return null;
  }

  return (
    <div className="space-y-2">
      <Label htmlFor="captcha-code">{t("auth.captcha.label", { defaultValue: "Captcha" })}</Label>
      <Input
        id="captcha-code"
        type="text"
        inputMode="numeric"
        autoComplete="off"
        placeholder={t("auth.captcha.placeholder", { defaultValue: "Enter the code shown" })}
        value={code}
        onChange={(event) => {
          const next = event.target.value;
          setCode(next);
          onChange({ captchaId: data.captchaId, captchaCode: next });
        }}
        required
      />
      {/* 验证码图片：点击刷新，窄屏（Android 视口）与宽屏均可用 */}
      <button
        type="button"
        onClick={() => void refresh()}
        disabled={loading}
        className="relative h-10 w-full overflow-hidden rounded-md border border-input bg-background"
        aria-label={t("auth.captcha.refresh", { defaultValue: "Refresh" })}
        title={t("auth.captcha.refresh", { defaultValue: "Refresh" })}
      >
        {loading || !data.image ? (
          <span className="flex h-full w-full items-center justify-center text-xs text-muted-foreground">
            <RefreshCw className="mr-1 h-3 w-3 animate-spin" />
            {t("auth.captcha.loading", { defaultValue: "Loading..." })}
          </span>
        ) : (
          <img
            src={data.image}
            alt={t("auth.captcha.label", { defaultValue: "Captcha" })}
            className="h-10 w-full object-contain"
            draggable={false}
          />
        )}
      </button>
      {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
    </div>
  );
}
