import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { ApiError, getErrorMessage } from "@/lib/api";

/**
 * 统一的错误提示 hook，返回一个函数，接受 unknown 错误并以顶部胶囊 toast 形式展示。
 *
 * 优先级：
 *   1. ApiError.messageKey 对应的 i18n 文本
 *   2. Error.message
 *   3. fallbackKey 翻译结果
 */
export function useErrorToast() {
  const { t } = useTranslation();
  return useCallback(
    (err: unknown, fallbackKey = "error.internalError") => {
      if (err instanceof ApiError) {
        const msg = getErrorMessage(err, t);
        toast.error(msg && msg !== err.messageKey ? msg : t(fallbackKey));
        return;
      }
      if (err instanceof Error) {
        toast.error(err.message || t(fallbackKey));
        return;
      }
      toast.error(t(fallbackKey));
    },
    [t]
  );
}
