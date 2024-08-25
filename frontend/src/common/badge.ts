


export function getBadgeStyle(date: Date | undefined): string {
  if (!date) {
    return "";
  }
  return "badge badge-outline badge-primary";
}