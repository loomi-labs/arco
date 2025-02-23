export enum RepoType {
  Local = "local",
  Remote = "remote",
  ArcoCloud = "arco-cloud",
}

export function getRepoType(locationStr: string): RepoType {
  return locationStr.startsWith("/") ? RepoType.Local : RepoType.Remote;
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

