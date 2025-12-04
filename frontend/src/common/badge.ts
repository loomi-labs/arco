import { diffDays, diffHours, offset, removeOffset } from "@formkit/tempo";
import type { LocationUnion } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";
import { LocationType } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";

/**
 * Returns the style for a badge based on the difference between the current date and the given date.
 * Uses theme-aware colors that change between light/dark mode via CSS custom properties.
 * @param date The date to compare with the current date.
 */
export function toCreationTimeBadge(date: Date | undefined): string {
  if (!date) {
    return "";
  }

  const now = new Date();
  const offsetToUTC = offset(now);
  date = removeOffset(date, offsetToUTC);

  const dHours = diffHours(now, date);
  if (dHours < 1) {
    return "badge text-badge-fresh-text bg-badge-fresh-bg border border-badge-fresh-border truncate cursor-pointer";
  }
  const dDays = diffDays(now, date);
  if (dDays < 1) {
    return "badge text-badge-recent-text bg-badge-recent-bg border border-badge-recent-border truncate cursor-pointer";
  }
  if (dDays < 2) {
    return "badge text-badge-days-text bg-badge-days-bg border border-badge-days-border truncate cursor-pointer";
  }
  if (dDays < 7) {
    return "badge text-badge-week-text bg-badge-week-bg border border-badge-week-border truncate cursor-pointer";
  }
  if (dDays < 30) {
    return "badge text-badge-month-text bg-badge-month-bg border border-badge-month-border truncate cursor-pointer";
  }
  if (dDays < 365) {
    return "badge text-badge-year-text bg-badge-year-bg border border-badge-year-border truncate cursor-pointer";
  }
  return "badge text-badge-old-text bg-badge-old-bg border border-badge-old-border truncate cursor-pointer";
}

/**
 * Returns the style for a badge based on the given repository type.
 * @param type The repository type.
 */
export function toRepoTypeBadge(type: LocationUnion): string {
  switch (type.type) {
    case LocationType.LocationTypeLocal:
      return "badge border-arco-purple-500 text-arco-purple-500 bg-transparent truncate";
    case LocationType.LocationTypeRemote:
      return "badge border-arco-purple-500 text-arco-purple-500 bg-transparent truncate";
    case LocationType.LocationTypeArcoCloud:
      return "badge border-arco-purple-500 text-arco-purple-500 bg-transparent truncate";
    case LocationType.$zero:
    default:
      return "badge border-arco-purple-500 text-arco-purple-500 bg-transparent truncate";
  }
}
