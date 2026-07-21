import { useEffect, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { AppWindow, ExternalLink, Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { AuthShell } from "@/components/auth/auth-shell";
import { isNativeClient } from "@/lib/client-runtime";

/**
 * 邮件按钮跳转的中转页。
 *
 * 后端把邮件按钮统一指向 https://domain.lxii.cc/redirect?token=xxx&purpose=xxx&app_link=linyu://...
 * 这样可以解决 QQ 邮箱等邮件客户端不直接支持 linyu:// 自定义 scheme 的问题。
 *
 * 本页逻辑：
 * 1. 解析 token / purpose / app_link 参数，得到 web 跳转目标和 native deep link
 * 2. 在 native 客户端中直接 navigate 到 web 目标（deep link 不会被再次触发）
 * 3. 其他环境：自动尝试唤起 native APP，同时显示「打开 APP」和「在网页中继续」两个按钮
 */
export default function RedirectPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const token = searchParams.get("token") ?? "";
  const purpose = searchParams.get("purpose") ?? "";
  const appLink = searchParams.get("app_link") ?? "";
  const [launching, setLaunching] = useState(false);

  // 根据 purpose 推导 web 目标
  const webTarget = (() => {
    if (!token) return "";
    if (purpose === "password_reset") {
      return `/reset-password?token=${encodeURIComponent(token)}`;
    }
    // registration_verify / account_activation / 其他默认都进入设置密码页
    return `/set-password?token=${encodeURIComponent(token)}`;
  })();

  // native 客户端中：deep link 已被原生层拦截，直接走 web 路由
  useEffect(() => {
    if (!isNativeClient) return;
    if (!webTarget) {
      navigate("/login", { replace: true });
      return;
    }
    navigate(webTarget, { replace: true });
  }, [isNativeClient, webTarget, navigate]);

  // 移动浏览器：进入页面时自动尝试唤起 native APP
  // 通过 location.href 触发，失败时用户可点击「打开 APP」按钮手动重试
  useEffect(() => {
    if (isNativeClient || !appLink) return;
    const fallbackTimer = window.setTimeout(() => setLaunching(false), 1500);
    try {
      setLaunching(true);
      window.location.href = appLink;
    } catch {
      setLaunching(false);
    }
    return () => window.clearTimeout(fallbackTimer);
  }, [isNativeClient, appLink]);

  // 桌面浏览器：没有 app_link 时直接跳转 web 目标
  useEffect(() => {
    if (isNativeClient) return;
    if (appLink) return; // 移动端流程
    if (!webTarget) return;
    navigate(webTarget, { replace: true });
  }, [isNativeClient, appLink, webTarget, navigate]);

  const handleOpenApp = () => {
    if (!appLink) return;
    setLaunching(true);
    try {
      window.location.href = appLink;
    } catch {
      setLaunching(false);
    }
  };

  const handleContinueWeb = () => {
    if (!webTarget) {
      navigate("/login", { replace: true });
      return;
    }
    navigate(webTarget, { replace: true });
  };

  // 参数缺失：显示错误并提供返回登录入口
  if (!webTarget) {
    return (
      <AuthShell
        title={t("auth.invalidLink", { defaultValue: "This link is invalid or has expired." })}
        description=""
      >
        <div className="space-y-4">
          <Button asChild className="w-full">
            <Link to="/login">{t("auth.verify.back", { defaultValue: "Back to sign in" })}</Link>
          </Button>
        </div>
      </AuthShell>
    );
  }

  // native 客户端：等 useEffect 跳转即可，渲染加载占位
  if (isNativeClient) {
    return (
      <AuthShell title={t("auth.tokenVerifying", { defaultValue: "Verifying link..." })} description="">
        <div className="flex items-center justify-center py-6 text-sm text-muted-foreground">
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          {t("auth.tokenVerifying", { defaultValue: "Verifying link..." })}
        </div>
      </AuthShell>
    );
  }

  return (
    <AuthShell
      title={t("redirect.title", { defaultValue: "Opening the app..." })}
      description={t("redirect.description", { defaultValue: "Choose how to continue: open the native app or proceed in the web browser." })}
      footer={<Link to="/login" className="font-medium text-primary hover:underline">{t("auth.verify.back", { defaultValue: "Back to sign in" })}</Link>}
    >
      <div className="space-y-4">
        {appLink ? (
          <Button type="button" className="w-full" onClick={handleOpenApp} disabled={launching}>
            {launching ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                {t("redirect.launching", { defaultValue: "Launching..." })}
              </>
            ) : (
              <>
                <AppWindow className="mr-2 h-4 w-4" />
                {t("redirect.openApp", { defaultValue: "Open in app" })}
              </>
            )}
          </Button>
        ) : null}
        <Button type="button" variant="outline" className="w-full" onClick={handleContinueWeb}>
          <ExternalLink className="mr-2 h-4 w-4" />
          {t("redirect.continueWeb", { defaultValue: "Continue in browser" })}
        </Button>
      </div>
    </AuthShell>
  );
}
