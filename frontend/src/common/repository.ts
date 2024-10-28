export enum Location {
  Local = "local",
  Remote = "remote",
}

export function getLocation(locationStr: string): Location {
  return locationStr.startsWith("ssh://") || locationStr.includes("@") ? Location.Remote : Location.Local;
}

export function getBgColor(location: Location): string {
  return location === Location.Local ? "bg-arco-purple-500 group-hover/repo:bg-arco-purple-500/70" : "bg-arco-purple-700 group-hover/repo:bg-arco-purple-700/70";
}

export function getTextColor(location: Location): string {
  return location === Location.Local ? "text-arco-purple-500" : "text-arco-purple-700";
}

export function getTextColorWithHover(location: Location): string {
  return `${getTextColor(location)} ${getTextColorOnlyHover(location)}`;
}

export function getTextColorOnlyHover(location: Location): string {
  return location === Location.Local ? "hover:text-arco-purple-500/70 group-hover:text-arco-purple-500/70" : "hover:text-arco-purple-700/70 group-hover:text-arco-purple-700/70";
}

export function getBorderColor(location: Location): string {
  return location === Location.Local ? "border-arco-purple-500 hover:border-arco-purple-500/70 group-hover:border-arco-purple-500/70" : "border-arco-purple-700 hover:border-arco-purple-700/70 group-hover:border-arco-purple-700/70";
}

export function getTooltipColor(location: Location): string {
  return location === Location.Local ? "tooltip-arco-purple-500" : "tooltip-arco-purple-700";
}

export function getBadge(location: Location): string {
  return location === Location.Local ? "badge bg-arco-purple-500 dark:badge-outline dark:text-arco-purple-500 h-full" : "badge text-white bg-arco-purple-700 dark:badge-outline dark:text-arco-purple-700 h-full";
}

export function toHumanReadableSize(size: number): string {
  if (size < 0) {
    return "Invalid size";
  }

  if (size === 0) {
    return "-";
  }

  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }
  return `${size.toFixed(2)} ${units[unitIndex]}`;
}
