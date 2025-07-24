import {
  addSecond,
  diffDays,
  diffHours,
  diffMinutes,
  diffMonths,
  diffSeconds,
  diffWeeks,
  diffYears,
  format,
  isBefore,
  offset,
  parse,
  removeOffset
} from "@formkit/tempo";


/**
 * getTime can be used for a v-model binding to a html input element
 * It converts a Date object to a string in the format "HH:mm"
 *
 * Usage:
 * const refTime = ref<Date>(new Date());
 * const timeModel = defineModel("time", {
 *   get() {
 *     return getTime(() => refTime.value);
 *   },
 *   set(value: string) {
 *     return setTime((date: Date) => refTime.value = date, value);
 *   }
 * });
 * <input v-model="timeModel" type="time">
 */
export function getTime(getValFn: () => Date | string): string | undefined {
  if (!getValFn()) {
    return undefined;
  }
  let date: Date;
  if (typeof getValFn() === "string") {
    date = parse(getValFn() as string);
  } else {
    date = getValFn() as Date;
  }

  if (isNaN(date.getTime())) {
    return undefined;
  }

  let hours = date.getHours();
  if (isNaN(hours)) {
    hours = 0;
  }

  let minutes = date.getMinutes();
  if (isNaN(minutes)) {
    minutes = 0;
  }
  return `${hours.toString().padStart(2, "0")}:${minutes.toString().padStart(2, "0")}`;
}

/**
 * setTime can be used for a v-model binding to a html input element
 * It converts a string in the format "HH:mm" to a Date object
 *
 * Usage:
 * const refTime = ref<Date>(new Date());
 * const timeModel = defineModel("time", {
 *   get() {
 *     return getTime(() => refTime.value);
 *   },
 *   set(value: string) {
 *     return setTime((date: Date) => refTime.value = date, value);
 *   }
 * });
 * <input v-model="timeModel" type="time">
 */
export function setTime(setValFn: (date: Date) => void, value: string): string {
  const split = value.split(":");
  if (split.length !== 2) {
    return value;
  }
  const [hours, minutes] = split.map((val) => {
    const num = Number(val);
    return isNaN(num) ? undefined : num;
  });
  if (hours === undefined || minutes === undefined || hours < 0 || hours > 23 || minutes < 0 || minutes > 59) {
    return value;
  }
  const date = new Date();
  date.setHours(hours, minutes, 0, 0);
  setValFn(date);
  return value;
}

/**
 * isInPast checks if a Date object is in the past.
 * @param date The date to check
 * @param ignoreTz If true, the timezone offset is ignored
 */
export function isInPast(date: Date, ignoreTz = false): boolean {
  const now = new Date();
  if (!ignoreTz) {
    const offsetToUTC = offset(now);
    date = removeOffset(date, offsetToUTC);
  }
  return isBefore(date, now);
}

/**
 * toRelativeTimeString converts a Date object to a human-readable string that is relative to the current time.
 * @param date The date to convert
 * @param ignoreTz If true, the timezone offset is ignored
 */
export function toRelativeTimeString(date: Date, ignoreTz = false): string {
  const now = new Date();
  if (!ignoreTz) {
    const offsetToUTC = offset(now);
    date = removeOffset(date, offsetToUTC);
  }

  if (isBefore(date, now)) {
    return toPastString(date, now);
  }
  return toFutureString(date, now);
}

/**
 * toFutureString converts a Date object to a human-readable string that is relative to the current time.
 * @param date The date to convert (must be in the future)
 * @param now The current date
 */
function toFutureString(date: Date, now: Date): string {
  const dSeconds = diffSeconds(date, now, "ceil");
  if (dSeconds < 60) {
    return `in less than a minute`;
  }

  const dMinutes = diffMinutes(date, now, "ceil");
  if (dMinutes < 60) {
    return `in ${dMinutes} minute${dMinutes !== 1 ? "s" : ""}`;
  }

  const dHours = diffHours(date, now, "ceil");
  if (dHours < 24) {
    return `in ${dHours} hour${dHours !== 1 ? "s" : ""}`;
  }

  const dDays = diffDays(date, now, "ceil");
  if (dDays < 7) {
    return `in ${dDays} day${dDays !== 1 ? "s" : ""}`;
  }

  const dWeeks = diffWeeks(date, now, "ceil");
  if (dWeeks < 4) {
    return `in ${dWeeks} week${dWeeks !== 1 ? "s" : ""}`;
  }

  const dMonths = diffMonths(date, now);
  if (dMonths === 0) {
    // If less than a month in the future, show in weeks
    return `in ${dWeeks} week${dWeeks !== 1 ? "s" : ""}`;
  }
  if (dMonths < 12) {
    return `in ${dMonths} month${dMonths !== 1 ? "s" : ""}`;
  }

  const dYears = diffYears(date, now);
  return `in ${dYears} year${dYears !== 1 ? "s" : ""}`;
}

/**
 * toPastString converts a Date object to a human-readable string that is relative to the current time.
 * @param date The date to convert (must be in the past)
 * @param now The current date
 */
function toPastString(date: Date, now: Date): string {
  const dSeconds = diffSeconds(now, date, "floor");
  if (dSeconds < 60) {
    return `less than a minute ago`;
  }

  const dMinutes = diffMinutes(now, date, "floor");
  if (dMinutes < 60) {
    return `${dMinutes} minute${dMinutes !== 1 ? "s" : ""} ago`;
  }

  const dHours = diffHours(now, date, "floor");
  if (dHours < 24) {
    return `${dHours} hour${dHours !== 1 ? "s" : ""} ago`;
  }

  const dDays = diffDays(now, date, "floor");
  if (dDays < 7) {
    return `${dDays} day${dDays !== 1 ? "s" : ""} ago`;
  }

  const dWeeks = diffWeeks(now, date, "floor");
  if (dWeeks < 4) {
    return `${dWeeks} week${dWeeks !== 1 ? "s" : ""} ago`;
  }

  const dMonths = diffMonths(now, date);
  if (dMonths === 0) {
    // If less than a month ago, show in weeks
    return `${dWeeks} week${dWeeks !== 1 ? "s" : ""} ago`;
  }
  if (dMonths < 12) {
    return `${dMonths} month${dMonths !== 1 ? "s" : ""} ago`;
  }

  const dYears = diffYears(now, date);
  return `${dYears} year${dYears !== 1 ? "s" : ""} ago`;
}

/**
 * toShortDateString converts a Date object to a human-readable string with a long date format.
 * @param date The date to convert
 */
export function toLongDateString(date: Date): string {
  const now = new Date();
  const offsetToUTC = offset(now);
  date = removeOffset(date, offsetToUTC);
  return format(date, { date: "long", time: "short" });
}

/**
 * toDurationString converts a number of seconds to a human-readable string.
 * @param seconds The number of seconds to convert
 * @returns A string in the format "Xs", "Xm Ys", "Xh Ym", or "Xd Yh"
 */
export function toDurationString(seconds: number): string {
  if (seconds <= 0) {
    return "-";
  }
  const date = addSecond(new Date(0), seconds);
  if (seconds < 60) {
    return `${seconds}s`;
  }
  if (seconds < 3600) {
    return `${date.getMinutes()}m ${date.getSeconds()}s`;
  }
  if (seconds < 86400) {
    return `${date.getHours()}h ${date.getMinutes()}m`;
  }
  return `${date.getDate()}d ${date.getHours()}h`;
}