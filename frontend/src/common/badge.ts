import { diffDays, diffHours, offset, removeOffset } from "@formkit/tempo";


/**
 * Returns the style for a badge based on the difference between the current date and the given date.
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
    return "badge badge-warning dark:badge-outline h-full cursor-pointer";
  }
  const dDays = diffDays(now, date);
  if (dDays < 1) {
    return "badge badge-success dark:badge-outline h-full cursor-pointer";
  }
  if (dDays < 2) {
    return "badge text-white bg-blue-600 dark:badge-outline dark:bg-transparent dark:text-blue-500 h-full cursor-pointer";
  }
  if (dDays < 7) {
    return "badge text-white bg-blue-900 dark:badge-outline dark:bg-transparent dark:text-blue-400 h-full cursor-pointer";
  }
  if (dDays < 30) {
    return "badge text-white bg-gray-700 dark:badge-outline dark:bg-transparent dark:text-blue-200 h-full cursor-pointer";
  }
  if (dDays < 365) {
    return "badge text-white bg-gray-500 dark:badge-outline dark:bg-transparent dark:text-gray-400 h-full cursor-pointer";
  }

  return "badge text-white bg-gray-300 dark:badge-outline dark:bg-transparent dark:text-gray-500 h-full cursor-pointer";
}