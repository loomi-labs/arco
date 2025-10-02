import { diffDays, diffHours, offset, removeOffset } from "@formkit/tempo";
import type { LocationUnion } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";
import { LocationType } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";

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
    return "badge badge-warning dark:border-warning dark:bg-transparent dark:text-warning dark:badge-warning truncate cursor-pointer";
  }
  const dDays = diffDays(now, date);
  if (dDays < 1) {
    return "badge badge-success dark:border-success dark:bg-transparent dark:text-success truncate cursor-pointer";
  }
  if (dDays < 2) {
    return "badge text-white bg-blue-600 dark:border-blue-500 dark:bg-transparent dark:text-blue-500 truncate cursor-pointer";
  }
  if (dDays < 7) {
    return "badge text-white bg-blue-900 dark:border-blue-400 dark:bg-transparent dark:text-blue-400 truncate cursor-pointer";
  }
  if (dDays < 30) {
    return "badge text-white bg-gray-700 dark:border-blue-200 dark:bg-transparent dark:text-blue-200 truncate cursor-pointer";
  }
  if (dDays < 365) {
    return "badge text-white bg-gray-500 dark:border-gray-400 dark:bg-transparent dark:text-gray-400 truncate cursor-pointer";
  }

  return "badge text-white bg-gray-300 dark:border-gray-500 dark:bg-transparent dark:text-gray-500 truncate cursor-pointer";
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
