import { CompressionMode } from "../../bindings/github.com/loomi-labs/arco/backend/ent/backupprofile/models";

export enum CompressionPreset {
  None = "none",
  Fast = "fast",
  Balanced = "balanced",
  Maximum = "maximum",
  Custom = "custom"
}

interface Algorithm {
  mode: CompressionMode;
  /** Short name used in card summary (must be unique across all algorithms) */
  summaryLabel: string;
  /** Label for simple-mode dropdown (only used when preset is set) */
  simpleLabel: string;
  /** Full label with algorithm suffix for expert dropdown */
  expertLabel: string;
  defaultLevel: number | null;
  /** Preset key for simple mode, null if expert-only */
  preset: CompressionPreset | null;
}

//                                                  summary     simple      expert               level  preset
const algorithms: Algorithm[] = [
  { mode: CompressionMode.CompressionModeLz4,  summaryLabel: "Fast",     simpleLabel: "Fast",     expertLabel: "Fast (lz4)",     defaultLevel: null, preset: CompressionPreset.Fast },
  { mode: CompressionMode.CompressionModeZstd, summaryLabel: "Balanced", simpleLabel: "Balanced", expertLabel: "Balanced (zstd)", defaultLevel: 3,    preset: CompressionPreset.Balanced },
  { mode: CompressionMode.CompressionModeZlib, summaryLabel: "Classic",  simpleLabel: "",         expertLabel: "Classic (zlib)",  defaultLevel: 6,    preset: null },
  { mode: CompressionMode.CompressionModeLzma, summaryLabel: "Maximum",  simpleLabel: "Maximum",  expertLabel: "Maximum (lzma)", defaultLevel: 6,    preset: CompressionPreset.Maximum },
];

function findByMode(mode: CompressionMode): Algorithm | undefined {
  return algorithms.find(a => a.mode === mode);
}

function findByPreset(preset: CompressionPreset): Algorithm | undefined {
  return algorithms.find(a => a.preset === preset);
}

/** Simple-mode preset options (for dropdown) */
export const simplePresets: { preset: CompressionPreset; label: string }[] = [
  { preset: CompressionPreset.None, label: "Off" },
  ...algorithms.filter(a => a.preset !== null).map(a => ({ preset: a.preset!, label: a.simpleLabel })),
];

/** Expert-mode algorithm options (for dropdown) */
export const expertAlgorithms: { mode: CompressionMode; label: string }[] =
  algorithms.map(a => ({ mode: a.mode, label: a.expertLabel }));

/** Get default level for a compression mode */
export function getDefaultLevel(mode: CompressionMode): number | null {
  return findByMode(mode)?.defaultLevel ?? null;
}

/** Map mode+level → preset */
export function getPreset(mode: CompressionMode | undefined, level: number | null): CompressionPreset {
  if (!mode || mode === CompressionMode.CompressionModeNone) return CompressionPreset.None;
  const algo = findByMode(mode);
  if (!algo?.preset) return CompressionPreset.Custom;
  if (algo.defaultLevel === null || level === null || level === algo.defaultLevel) return algo.preset;
  return CompressionPreset.Custom;
}

/** Map preset → mode+level */
export function fromPreset(preset: CompressionPreset): { mode: CompressionMode; level: number | null } {
  if (preset === CompressionPreset.None) return { mode: CompressionMode.CompressionModeNone, level: null };
  const algo = findByPreset(preset);
  if (!algo) return { mode: CompressionMode.CompressionModeLz4, level: null };
  return { mode: algo.mode, level: algo.defaultLevel };
}

/** Human-readable label for the collapse header summary */
export function getCompressionLabel(mode: CompressionMode | undefined, level: number | null, expert = false): string {
  if (!mode || mode === CompressionMode.CompressionModeNone) return "Off";
  const algo = findByMode(mode);
  if (!algo) return "Custom";
  if (!expert) return algo.summaryLabel;
  if (algo.defaultLevel !== null && level !== null && level !== algo.defaultLevel) {
    return `${algo.summaryLabel} (${mode} - level ${level})`;
  }
  return `${algo.summaryLabel} (${mode})`;
}
