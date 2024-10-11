import { diffDays, diffHours, offset, removeOffset } from "@formkit/tempo";


/**
 * Returns the style for a badge based on the difference between the current date and the given date.
 * @param date The date to compare with the current date.
 */
export function toDurationBadge(date: Date | undefined): string {
  if (!date) {
    return "";
  }

  const now = new Date();
  const offsetToUTC = offset(now);
  date = removeOffset(date, offsetToUTC);

  const dHours = diffHours(now, date);
  if (dHours < 1) {
    return "badge badge-outline text-warning h-full";
  }
  const dDays = diffDays(now, date);
  if (dDays < 1) {
    return "badge badge-outline text-success h-full";
  }
  if (dDays < 2) {
    return "badge badge-outline text-blue-600 dark:text-blue-500 h-full";
  }
  if (dDays < 7) {
    return "badge badge-outline text-blue-900 dark:text-blue-400 h-full";
  }
  if (dDays < 30) {
    return "badge badge-outline text-gray-600 dark:text-blue-200 h-full";
  }
  if (dDays < 365) {
    return "badge badge-outline text-gray-500 dark:text-gray-400 h-full";
  }

  return "badge badge-outline text-gray-400 dark:text-gray-500 h-full";
}