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
 * Returns the tooltip class for a tooltip based on the difference between the current date and the given date.
 * Uses the same color scheme as toCreationTimeBadge.
 * @param date The date to compare with the current date.
 */
export function toCreationTimeTooltip(date: Date | undefined): string {
  if (!date) {
    return "tooltip";
  }

  const now = new Date();
  const offsetToUTC = offset(now);
  date = removeOffset(date, offsetToUTC);

  const dHours = diffHours(now, date);
  if (dHours < 1) {
    return "tooltip tooltip-badge-fresh";
  }
  const dDays = diffDays(now, date);
  if (dDays < 1) {
    return "tooltip tooltip-badge-recent";
  }
  if (dDays < 2) {
    return "tooltip tooltip-badge-days";
  }
  if (dDays < 7) {
    return "tooltip tooltip-badge-week";
  }
  if (dDays < 30) {
    return "tooltip tooltip-badge-month";
  }
  if (dDays < 365) {
    return "tooltip tooltip-badge-year";
  }
  return "tooltip tooltip-badge-old";
}

/**
 * Returns just the text color class for an icon based on the difference between the current date and the given date.
 * Uses the same color scheme as toCreationTimeBadge.
 * @param date The date to compare with the current date.
 */
export function toCreationTimeIconColor(date: Date | undefined): string {
  if (!date) {
    return "";
  }

  const now = new Date();
  const offsetToUTC = offset(now);
  date = removeOffset(date, offsetToUTC);

  const dHours = diffHours(now, date);
  if (dHours < 1) {
    return "text-badge-fresh-border";
  }
  const dDays = diffDays(now, date);
  if (dDays < 1) {
    return "text-badge-recent-border";
  }
  if (dDays < 2) {
    return "text-badge-days-border";
  }
  if (dDays < 7) {
    return "text-badge-week-border";
  }
  if (dDays < 30) {
    return "text-badge-month-border";
  }
  if (dDays < 365) {
    return "text-badge-year-border";
  }
  return "text-badge-old-border";
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
