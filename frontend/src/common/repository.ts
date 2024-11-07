export enum Location {
  Local = "local",
  Remote = "remote",
  ArcoCloud = "arco-cloud",
}

// Required so that we can dynamically generate the CSS classes
// Otherwise, Tailwind CSS will remove the unused classes
// noinspection JSUnusedGlobalSymbols
const _tailwindCssPlaceholder = "" +
  "text-local-repo" +
  "text-remote-repo" +
  "text-arco-cloud-repo" +
  "bg-local-repo" +
  "bg-remote-repo" +
  "bg-arco-cloud-repo" +
  "border-local-repo" +
  "border-remote-repo" +
  "border-arco-cloud-repo" +
  "tooltip-local-repo" +
  "tooltip-remote-repo" +
  "tooltip-arco-cloud-repo" +
  "badge-local-repo" +
  "badge-remote-repo" +
  "badge-arco-cloud-repo" +
  "hover:text-local-repo" +
  "hover:text-remote-repo" +
  "hover:text-arco-cloud-repo" +
  "hover:bg-local-repo" +
  "hover:bg-remote-repo" +
  "hover:bg-arco-cloud-repo" +
  "hover:border-local-repo" +
  "hover:border-remote-repo" +
  "hover:border-arco-cloud-repo" +
  "group-hover:text-local-repo" +
  "group-hover:text-remote-repo" +
  "group-hover:text-arco-cloud-repo" +
  "group-hover:bg-local-repo" +
  "group-hover:bg-remote-repo" +
  "group-hover:bg-arco-cloud-repo" +
  "group-hover:border-local-repo" +
  "group-hover:border-remote-repo" +
  "group-hover:border-arco-cloud-repo";

export function getLocation(locationStr: string): Location {
  return locationStr.startsWith("/") ? Location.Local : Location.Remote;
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

export enum RepoType {
  Local = "local",
  Remote = "remote",
  ArcoCloud = "arco-cloud",
}
