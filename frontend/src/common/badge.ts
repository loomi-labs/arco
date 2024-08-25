


export function getBadgeStyle(date: Date | undefined): string {
  if (!date) {
    return "";
  }
  // TODO: fix this
  return "badge badge-outline badge-success";
}