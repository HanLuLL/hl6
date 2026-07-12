import { clsx, type ClassValue } from "clsx"
import DOMPurify from "dompurify"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

const SANITIZE_ALLOWED_TAGS = ["p", "br", "strong", "em", "u", "s", "h2", "h3", "h4", "ul", "ol", "li", "a", "img", "span", "div", "blockquote", "code", "pre", "hr", "table", "thead", "tbody", "tr", "th", "td"]
const SANITIZE_ALLOWED_ATTR = ["href", "src", "alt", "target", "rel", "style", "class"]

DOMPurify.addHook("afterSanitizeAttributes", (node) => {
  if (node.tagName === "A") {
    node.setAttribute("rel", "noopener noreferrer")
  }
})

export function sanitizeHTML(html: string): string {
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS: SANITIZE_ALLOWED_TAGS,
    ALLOWED_ATTR: SANITIZE_ALLOWED_ATTR,
    ALLOW_DATA_ATTR: false,
  })
}
