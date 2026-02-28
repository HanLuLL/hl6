import { useEffect } from "react"
import {
  CircleCheckIcon,
  InfoIcon,
  Loader2Icon,
  OctagonXIcon,
  TriangleAlertIcon,
} from "lucide-react"
import { useTheme } from "next-themes"
import { useTranslation } from "react-i18next"
import { Toaster as Sonner, type ToasterProps } from "sonner"

const Toaster = ({ ...props }: ToasterProps) => {
  const { theme = "system" } = useTheme()
  const { t } = useTranslation()

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      const toastEl = (e.target as HTMLElement).closest(
        "[data-sonner-toast]"
      ) as HTMLElement | null
      if (!toastEl) return

      const contentEl = toastEl.querySelector("[data-content]") as HTMLElement | null
      if (!contentEl) return

      const text = contentEl.textContent
      if (!text || toastEl.dataset.copied === "true") return
      toastEl.dataset.copied = "true"

      navigator.clipboard.writeText(text)

      contentEl.style.overflow = "hidden"
      contentEl.style.transition = "none"
      contentEl.style.height = `${contentEl.offsetHeight}px`

      const inner = contentEl.querySelector("[data-title]") as HTMLElement | null
      const target = inner || contentEl
      const originalContent = target.textContent || ""

      // 1. 提示文本向上滚出
      target.style.transition = "transform 0.2s ease-in, opacity 0.2s ease-in"
      target.style.transform = "translateY(-100%)"
      target.style.opacity = "0"

      setTimeout(() => {
        // 2. "已复制"从下方进入
        if (inner) inner.textContent = t("common.copied")
        else contentEl.textContent = t("common.copied")
        target.style.transition = "none"
        target.style.transform = "translateY(100%)"
        target.style.opacity = "0"
        requestAnimationFrame(() => {
          target.style.transition = "transform 0.2s ease-out, opacity 0.2s ease-out"
          target.style.transform = "translateY(0)"
          target.style.opacity = "1"

          // 3. 停留 500ms 后，"已复制"向上滚出
          setTimeout(() => {
            target.style.transition = "transform 0.2s ease-in, opacity 0.2s ease-in"
            target.style.transform = "translateY(-100%)"
            target.style.opacity = "0"

            setTimeout(() => {
              // 4. 提示文本从下方进入
              if (inner) inner.textContent = originalContent
              else contentEl.textContent = originalContent
              target.style.transition = "none"
              target.style.transform = "translateY(100%)"
              target.style.opacity = "0"
              requestAnimationFrame(() => {
                target.style.transition = "transform 0.2s ease-out, opacity 0.2s ease-out"
                target.style.transform = "translateY(0)"
                target.style.opacity = "1"
                delete toastEl.dataset.copied
                contentEl.style.overflow = ""
                contentEl.style.height = ""
              })
            }, 200)
          }, 500)
        })
      }, 200)
    }

    document.addEventListener("click", handler, true)
    return () => document.removeEventListener("click", handler, true)
  }, [t])

  return (
    <Sonner
      theme={theme as ToasterProps["theme"]}
      position="top-center"
      richColors
      className="toaster group"
      icons={{
        success: <CircleCheckIcon className="size-4" />,
        info: <InfoIcon className="size-4" />,
        warning: <TriangleAlertIcon className="size-4" />,
        error: <OctagonXIcon className="size-4" />,
        loading: <Loader2Icon className="size-4 animate-spin" />,
      }}
      toastOptions={{
        style: {
          paddingInline: "20px",
          transition: "transform 0.45s cubic-bezier(0.34, 1.56, 0.64, 1), opacity 0.3s ease, height 0.4s ease, box-shadow 0.2s ease",
          cursor: "pointer",
        },
      }}
      style={
        {
          "--normal-bg": "var(--popover)",
          "--normal-text": "var(--popover-foreground)",
          "--normal-border": "var(--border)",
          "--border-radius": "9999px",
        } as React.CSSProperties
      }
      {...props}
    />
  )
}

export { Toaster }
