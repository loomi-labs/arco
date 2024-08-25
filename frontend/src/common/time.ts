import { diffDays, format, parse } from "@formkit/tempo";


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
  const [hours, minutes] = split.map(Number);
  if (hours < 0 || hours > 23 || minutes < 0 || minutes > 59) {
    return value;
  }
  const date = new Date();
  date.setHours(hours, minutes, 0, 0);
  setValFn(date);
  return value;
}


/**
 * toHumanReadable converts a Date object to a human-readable string
 * @param date
 */
export function toHumanReadable(date: Date): string {
  const dd = diffDays(date, new Date());
  if (dd === 0) {
    return "Today";
  } else if (dd === 1) {
    return "Tomorrow";
  } else if (dd === -1) {
    return "Yesterday";
  } else if (dd > 1 && dd < 7) {
    return format(date, "dddd");
  }
  return format(date, "dddd, MMMM Do YYYY");
}
