export enum Location {
  Local = "local",
  Remote = "remote",
}

export function getLocation(locationStr: string): Location {
  return locationStr.startsWith("ssh://") || locationStr.includes("@") ? Location.Remote : Location.Local;
}

export function getBgColor(location: Location): string {
  return location === Location.Local ? "bg-secondary group-hover/repo:bg-secondary/70" : "bg-info group-hover/repo:bg-info/70";
}

export function getTextColor(location: Location): string {
  return location === Location.Local ? "text-secondary" : "text-info";
}

export function getTooltipColor(location: Location): string {
  return location === Location.Local ? "tooltip-secondary" : "tooltip-info";
}

export function getBadgeColor(location: Location): string {
  return location === Location.Local ? "badge-secondary" : "badge-info";
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
