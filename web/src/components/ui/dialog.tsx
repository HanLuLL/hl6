"use client"

import * as React from "react"
import { XIcon } from "lucide-react"
import { Dialog as DialogPrimitive } from "radix-ui"
import { useTranslation } from "react-i18next"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"

const HOTKEY_GUIDE_STORAGE_KEY = "hl6-dialog-hotkey-guide-seen-v1"

function isElementDisabled(element: HTMLElement): boolean {
  if (
    element instanceof HTMLButtonElement ||
    element instanceof HTMLInputElement ||
    element instanceof HTMLTextAreaElement ||
    element instanceof HTMLSelectElement
  ) {
    return element.disabled
  }
  return element.getAttribute("aria-disabled") === "true"
}

function isElementVisible(element: HTMLElement): boolean {
  if (element.hidden || element.getAttribute("aria-hidden") === "true") return false
  const style = window.getComputedStyle(element)
  return style.display !== "none" && style.visibility !== "hidden"
}

function isRequiredElement(element: HTMLElement): boolean {
  if (element.dataset.hotkeyRequired === "true") return true
  if (
    element instanceof HTMLInputElement ||
    element instanceof HTMLTextAreaElement ||
    element instanceof HTMLSelectElement
  ) {
    return element.required
  }
  return element.getAttribute("aria-required") === "true"
}

function isFieldFilled(element: HTMLElement): boolean {
  const filledState = element.dataset.hotkeyFilled
  if (filledState === "true") return true
  if (filledState === "false") return false

  if (element instanceof HTMLInputElement) {
    if (element.type === "checkbox" || element.type === "radio") return element.checked
    return element.value.trim().length > 0
  }

  if (element instanceof HTMLTextAreaElement || element instanceof HTMLSelectElement) {
    return element.value.trim().length > 0
  }

  if (element.isContentEditable || element.getAttribute("contenteditable") === "true") {
    return (element.textContent ?? "").trim().length > 0
  }

  if (element.getAttribute("role") === "combobox") {
    const value = element.getAttribute("data-value") ?? element.getAttribute("aria-valuetext") ?? ""
    if (value.trim().length > 0) return true
    return (element.textContent ?? "").trim().length > 0
  }

  return false
}

function focusRequiredElement(element: HTMLElement) {
  const focusSelector = element.dataset.hotkeyFocusSelector
  if (focusSelector) {
    const target = document.querySelector<HTMLElement>(focusSelector)
    if (target) {
      target.focus()
      return
    }
  }

  element.focus()
  if (element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement) {
    element.select()
  }
}

function findFirstMissingRequiredField(container: HTMLElement): HTMLElement | null {
  const candidates = Array.from(
    container.querySelectorAll<HTMLElement>(
      "[data-hotkey-required='true'], input[required], textarea[required], select[required], [aria-required='true']"
    )
  )

  for (const candidate of candidates) {
    if (!isElementVisible(candidate) || isElementDisabled(candidate) || candidate.closest("[aria-hidden='true']")) {
      continue
    }
    if (!isRequiredElement(candidate)) continue
    if (!isFieldFilled(candidate)) return candidate
  }

  return null
}

function resolvePrimaryAction(container: HTMLElement): HTMLElement | null {
  const explicitPrimary = container.querySelector<HTMLElement>("[data-dialog-primary='true']")
  if (explicitPrimary && !isElementDisabled(explicitPrimary) && isElementVisible(explicitPrimary)) {
    return explicitPrimary
  }

  const footerButtons = Array.from(
    container.querySelectorAll<HTMLElement>(
      "[data-slot='dialog-footer'] button:not([data-slot='dialog-close']):not([data-dialog-ignore-primary='true'])"
    )
  ).filter((button) => !isElementDisabled(button) && isElementVisible(button))
  if (footerButtons.length > 0) return footerButtons[footerButtons.length - 1]

  const submitButton = container.querySelector<HTMLElement>(
    "button[type='submit']:not([data-slot='dialog-close']):not([data-dialog-ignore-primary='true'])"
  )
  if (submitButton && !isElementDisabled(submitButton) && isElementVisible(submitButton)) {
    return submitButton
  }

  const allButtons = Array.from(
    container.querySelectorAll<HTMLElement>(
      "button:not([data-slot='dialog-close']):not([data-dialog-ignore-primary='true'])"
    )
  ).filter((button) => !isElementDisabled(button) && isElementVisible(button))
  if (allButtons.length > 0) return allButtons[allButtons.length - 1]

  const closeButton = container.querySelector<HTMLElement>("button[data-slot='dialog-close']")
  if (closeButton && !isElementDisabled(closeButton) && isElementVisible(closeButton)) {
    return closeButton
  }

  return null
}

function Dialog({
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Root>) {
  return <DialogPrimitive.Root data-slot="dialog" {...props} />
}

function DialogTrigger({
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Trigger>) {
  return <DialogPrimitive.Trigger data-slot="dialog-trigger" {...props} />
}

function DialogPortal({
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Portal>) {
  return <DialogPrimitive.Portal data-slot="dialog-portal" {...props} />
}

function DialogClose({
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Close>) {
  return <DialogPrimitive.Close data-slot="dialog-close" {...props} />
}

function DialogOverlay({
  className,
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Overlay>) {
  return (
    <DialogPrimitive.Overlay
      data-slot="dialog-overlay"
      className={cn(
        "data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 fixed inset-0 z-50 bg-black/50",
        className
      )}
      {...props}
    />
  )
}

function DialogContent({
  className,
  children,
  showCloseButton = true,
  showHotkeyGuide = true,
  enableHotkeys = true,
  onKeyDownCapture,
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Content> & {
  showCloseButton?: boolean
  showHotkeyGuide?: boolean
  enableHotkeys?: boolean
}) {
  const { t } = useTranslation()
  const [guideDismissed, setGuideDismissed] = React.useState(() => {
    if (!showHotkeyGuide || typeof window === "undefined") return true
    return window.localStorage.getItem(HOTKEY_GUIDE_STORAGE_KEY) === "1"
  })

  React.useEffect(() => {
    if (!showHotkeyGuide) {
      setGuideDismissed(true)
      return
    }
    setGuideDismissed(window.localStorage.getItem(HOTKEY_GUIDE_STORAGE_KEY) === "1")
  }, [showHotkeyGuide])

  const handleDismissGuide = () => {
    setGuideDismissed(true)
    window.localStorage.setItem(HOTKEY_GUIDE_STORAGE_KEY, "1")
  }

  const handleKeyDownCapture = (event: React.KeyboardEvent<HTMLDivElement>) => {
    onKeyDownCapture?.(event)
    if (event.defaultPrevented || !enableHotkeys) return
    if (!(event.metaKey || event.ctrlKey) || event.key !== "Enter") return
    if (event.isComposing || event.nativeEvent.isComposing || event.repeat) return

    const content = event.currentTarget
    const firstMissingRequired = findFirstMissingRequiredField(content)
    if (firstMissingRequired) {
      event.preventDefault()
      focusRequiredElement(firstMissingRequired)
      return
    }

    const primaryAction = resolvePrimaryAction(content)
    if (!primaryAction) return

    event.preventDefault()
    primaryAction.click()
  }

  return (
    <DialogPortal data-slot="dialog-portal">
      <DialogOverlay />
      <DialogPrimitive.Content
        data-slot="dialog-content"
        onKeyDownCapture={handleKeyDownCapture}
        className={cn(
          "bg-background data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 fixed top-[50%] left-[50%] z-50 grid w-full max-w-[calc(100%-2rem)] translate-x-[-50%] translate-y-[-50%] gap-4 rounded-lg border p-6 shadow-lg duration-200 outline-none sm:max-w-lg",
          className
        )}
        {...props}
      >
        {showHotkeyGuide && !guideDismissed && (
          <div className="flex items-center justify-between gap-3 rounded-md border bg-muted/40 px-3 py-2">
            <p className="text-xs text-muted-foreground">
              {t("common.hotkeyGuide")}
            </p>
            <Button
              type="button"
              size="xs"
              variant="outline"
              onClick={handleDismissGuide}
              className="shrink-0"
            >
              {t("common.gotIt")}
            </Button>
          </div>
        )}
        {children}
        {showCloseButton && (
          <DialogPrimitive.Close
            data-slot="dialog-close"
            className="ring-offset-background focus:ring-ring data-[state=open]:bg-accent data-[state=open]:text-muted-foreground absolute top-4 right-4 rounded-xs opacity-70 transition-opacity hover:opacity-100 focus:ring-2 focus:ring-offset-2 focus:outline-hidden disabled:pointer-events-none [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4"
          >
            <XIcon />
            <span className="sr-only">Close</span>
          </DialogPrimitive.Close>
        )}
      </DialogPrimitive.Content>
    </DialogPortal>
  )
}

function DialogHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="dialog-header"
      className={cn("flex flex-col gap-2 text-center sm:text-left", className)}
      {...props}
    />
  )
}

function DialogFooter({
  className,
  showCloseButton = false,
  children,
  ...props
}: React.ComponentProps<"div"> & {
  showCloseButton?: boolean
}) {
  return (
    <div
      data-slot="dialog-footer"
      className={cn(
        "flex flex-col-reverse gap-2 sm:flex-row sm:justify-end",
        className
      )}
      {...props}
    >
      {children}
      {showCloseButton && (
        <DialogPrimitive.Close asChild>
          <Button variant="outline">Close</Button>
        </DialogPrimitive.Close>
      )}
    </div>
  )
}

function DialogTitle({
  className,
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Title>) {
  return (
    <DialogPrimitive.Title
      data-slot="dialog-title"
      className={cn("text-lg leading-none font-semibold", className)}
      {...props}
    />
  )
}

function DialogDescription({
  className,
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Description>) {
  return (
    <DialogPrimitive.Description
      data-slot="dialog-description"
      className={cn("text-muted-foreground text-sm", className)}
      {...props}
    />
  )
}

export {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogOverlay,
  DialogPortal,
  DialogTitle,
  DialogTrigger,
}
