export function buildPaginatedQuery(
  path: string,
  page: number,
  perPage: number,
  extra?: Record<string, string | undefined>,
): string {
  const params = new URLSearchParams({
    page: String(page),
    per_page: String(perPage),
  });
  if (extra) {
    for (const [k, v] of Object.entries(extra)) {
      if (v === undefined || v === "") continue;
      params.set(k, v);
    }
  }
  const q = params.toString();
  const sep = path.includes("?") ? "&" : "?";
  return `${path}${sep}${q}`;
}
