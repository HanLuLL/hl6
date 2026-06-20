import punycode from "punycode";

/** 将 ACE 主机名/FQDN 渲染为 Unicode 标签（DNS 展示形式）。 */
export function domainToUnicode(value: string): string {
  const trimmed = value.trim().replace(/\.+$/, "");
  if (!trimmed) {
    return value;
  }
  return trimmed
    .split(".")
    .map((label) => {
      if (!/^xn--/i.test(label)) {
        return label;
      }
      try {
        return punycode.toUnicode(label.toLowerCase());
      } catch {
        return label;
      }
    })
    .join(".");
}
