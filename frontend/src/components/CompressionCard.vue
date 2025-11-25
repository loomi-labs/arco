<script setup lang='ts'>
import { computed, useId, useTemplateRef } from "vue";
import { InformationCircleIcon } from "@heroicons/vue/24/outline";
import { CompressionMode } from "../../bindings/github.com/loomi-labs/arco/backend/ent/backupprofile";
import { useExpertMode } from "../common/expertMode";
import CompressionInfoModal from "./CompressionInfoModal.vue";

/************
 * Types
 ************/

enum CompressionPreset {
  None = "none",
  Fast = "fast",
  Balanced = "balanced",
  Maximum = "maximum",
  Custom = "custom"
}

interface Props {
  compressionMode: CompressionMode;
  compressionLevel: number | null;
}

/************
 * Props & Emits
 ************/

const props = defineProps<Props>();

const emit = defineEmits<{
  "update:compression": [{ mode: CompressionMode; level: number | null }];
}>();

/************
 * Variables
 ************/

const compressionInfoModalKey = useId();
const compressionInfoModal = useTemplateRef<InstanceType<typeof CompressionInfoModal>>(compressionInfoModalKey);
const { expertMode } = useExpertMode();

/************
 * Computed
 ************/

const compressionPreset = computed({
  get: (): CompressionPreset => {
    const mode = props.compressionMode;
    const level = props.compressionLevel;

    if (mode === CompressionMode.CompressionModeNone) return CompressionPreset.None;
    if (mode === CompressionMode.CompressionModeLz4) return CompressionPreset.Fast;
    if (mode === CompressionMode.CompressionModeZstd && level === 3) return CompressionPreset.Balanced;
    if (mode === CompressionMode.CompressionModeLzma && level === 6) return CompressionPreset.Maximum;
    return CompressionPreset.Custom; // Custom settings that don't match a preset
  },
  set: (preset: CompressionPreset) => {
    switch (preset) {
      case CompressionPreset.None:
        emit("update:compression", { mode: CompressionMode.CompressionModeNone, level: null });
        break;
      case CompressionPreset.Fast:
        emit("update:compression", { mode: CompressionMode.CompressionModeLz4, level: null });
        break;
      case CompressionPreset.Balanced:
        emit("update:compression", { mode: CompressionMode.CompressionModeZstd, level: 3 });
        break;
      case CompressionPreset.Maximum:
        emit("update:compression", { mode: CompressionMode.CompressionModeLzma, level: 6 });
        break;
      case CompressionPreset.Custom:
        // No-op: keep current custom settings unchanged
        break;
      default:
        // Fallback to Fast preset
        emit("update:compression", { mode: CompressionMode.CompressionModeLz4, level: null });
        break;
    }
  }
});

const compressionLevelRange = computed(() => {
  const mode = props.compressionMode;
  if (mode === CompressionMode.CompressionModeZstd) {
    return { min: 1, max: 22, default: 3 };
  } else if (mode === CompressionMode.CompressionModeZlib) {
    return { min: 0, max: 9, default: 6 };
  } else if (mode === CompressionMode.CompressionModeLzma) {
    return { min: 0, max: 6, default: 6 };
  } else {
    return { min: 0, max: 0, default: 0 };
  }
});

const showCompressionLevelSlider = computed(() => {
  const mode = props.compressionMode;
  return mode === CompressionMode.CompressionModeZstd ||
         mode === CompressionMode.CompressionModeZlib ||
         mode === CompressionMode.CompressionModeLzma;
});

const isWarningLevel = computed(() => {
  const mode = props.compressionMode;
  const level = props.compressionLevel;

  if (level === null || level === undefined) return false;

  // ZSTD levels 16-22: Very high memory usage
  return mode === CompressionMode.CompressionModeZstd && level >= 16;
});

const customPresetLabel = computed(() => {
  const mode = props.compressionMode;
  const level = props.compressionLevel;

  // Extract the mode name (e.g., "zstd" from "CompressionModeZstd")
  const modeName = mode?.replace('CompressionMode', '').toLowerCase() || 'unknown';

  if (level !== null && level !== undefined) {
    return `Custom (${modeName}, level ${level})`;
  }
  return `Custom (${modeName})`;
});

const algorithmExplanation = computed<{ name: string; description: string } | null>(() => {
  if (!expertMode.value) {
    // Simple mode: explain presets with algorithm names
    switch (compressionPreset.value) {
      case CompressionPreset.None:
        return { name: 'None', description: 'No compression applied, fastest backup speed but largest backup size.' };
      case CompressionPreset.Fast:
        return { name: 'Fast - lz4', description: 'Very fast compression with low compression ratio, ideal for quick backups.' };
      case CompressionPreset.Balanced:
        return { name: 'Balanced - zstd', description: 'Modern algorithm with good balance of speed and compression ratio.' };
      case CompressionPreset.Maximum:
        return { name: 'Maximum - lzma', description: 'Highest compression ratio but slowest, best for long-term archives.' };
      case CompressionPreset.Custom: {
        // Extract algorithm name for custom preset
        const modeName = props.compressionMode?.replace('CompressionMode', '').toLowerCase() || 'unknown';
        return { name: `Custom - ${modeName}`, description: 'User-defined compression settings with custom algorithm and level.' };
      }
      default:
        return null;
    }
  } else {
    // Expert mode: explain algorithms with level-specific descriptions
    const level = props.compressionLevel;

    switch (props.compressionMode) {
      case CompressionMode.CompressionModeNone:
        return { name: 'none', description: 'No compression applied, fastest backup speed but largest backup size.' };

      case CompressionMode.CompressionModeLz4:
        return { name: 'lz4', description: 'Very fast compression with low compression ratio, ideal for quick backups.' };

      case CompressionMode.CompressionModeZstd: {
        if (level === null || level === undefined) {
          return { name: 'zstd', description: 'Modern algorithm offering configurable balance between speed and compression.' };
        } else if (level >= 1 && level <= 3) {
          return { name: 'zstd', description: 'Very fast compression similar to lz4, ideal for frequent backups.' };
        } else if (level >= 4 && level <= 9) {
          return { name: 'zstd', description: 'Best balance of speed and compression for most use cases.' };
        } else if (level >= 10 && level <= 15) {
          return { name: 'zstd', description: 'High compression with increased memory usage and slower speed.' };
        } else if (level >= 16 && level <= 22) {
          return { name: 'zstd', description: 'Maximum compression with very high memory usage and much slower compression (but fast decompression). Use only if disk space is critical!' };
        }
        return { name: 'zstd', description: 'Modern algorithm offering configurable balance between speed and compression.' };
      }

      case CompressionMode.CompressionModeZlib: {
        if (level === null || level === undefined) {
          return { name: 'zlib', description: 'Balanced algorithm with wide compatibility and good compression.' };
        } else if (level >= 0 && level <= 3) {
          return { name: 'zlib', description: 'Faster compression with moderate ratio, suitable for frequent backups.' };
        } else if (level >= 4 && level <= 6) {
          return { name: 'zlib', description: 'Balanced compression, good for general use.' };
        } else if (level >= 7 && level <= 9) {
          return { name: 'zlib', description: 'Slower compression with diminishing returns compared to lower levels.' };
        }
        return { name: 'zlib', description: 'Balanced algorithm with wide compatibility and good compression.' };
      }

      case CompressionMode.CompressionModeLzma: {
        if (level === null || level === undefined) {
          return { name: 'lzma', description: 'Highest compression ratio with slower processing time, best for long-term archives.' };
        } else if (level >= 0 && level <= 3) {
          return { name: 'lzma', description: 'Better compression than zstd but slower processing time.' };
        } else if (level >= 4 && level <= 6) {
          return { name: 'lzma', description: 'Standard archival compression with high ratio, slow but effective.' };
        }
        return { name: 'lzma', description: 'Highest compression ratio with slower processing time, best for long-term archives.' };
      }

      case CompressionMode.$zero:
      default:
        return null;
    }
  }
});

/************
 * Functions
 ************/

function onCompressionModeChange(mode: CompressionMode) {
  // Set default level for modes that support it
  let level: number | null = null;
  if (mode === CompressionMode.CompressionModeZstd) {
    level = 3;
  } else if (mode === CompressionMode.CompressionModeZlib) {
    level = 6;
  } else if (mode === CompressionMode.CompressionModeLzma) {
    level = 6;
  }

  emit("update:compression", { mode, level });
}

function onCompressionLevelChange(level: number) {
  emit("update:compression", { mode: props.compressionMode, level });
}

function toggleCompressionInfoModal() {
  compressionInfoModal.value?.showModal();
}

</script>

<template>
  <div class='ac-card p-10'>
    <div class='flex items-center justify-between mb-4'>
      <h3 class='text-xl font-semibold'>Compression</h3>
      <button @click='toggleCompressionInfoModal' class='btn btn-circle btn-ghost btn-xs'>
        <InformationCircleIcon class='size-6' />
      </button>
    </div>

    <!-- Simple Mode -->
    <div v-if='!expertMode' class='form-control'>
      <label class='label'>
        <span class='label-text font-medium'>Compression</span>
      </label>
      <select
        class='select select-bordered w-full'
        :value='compressionPreset'
        @change='(e) => compressionPreset = (e.target as HTMLSelectElement).value as CompressionPreset'>
        <option :value='CompressionPreset.None'>None</option>
        <option :value='CompressionPreset.Fast'>Fast</option>
        <option :value='CompressionPreset.Balanced'>Balanced</option>
        <option :value='CompressionPreset.Maximum'>Maximum</option>
        <option v-if='compressionPreset === CompressionPreset.Custom' :value='CompressionPreset.Custom'>{{ customPresetLabel }}</option>
      </select>
    </div>

    <!-- Expert Mode -->
    <div v-else class='space-y-4'>
      <div class='form-control'>
        <label class='label'>
          <span class='label-text font-medium'>Algorithm</span>
        </label>
        <select
          class='select select-bordered w-full'
          :value='props.compressionMode || CompressionMode.CompressionModeLz4'
          @change='(e) => onCompressionModeChange((e.target as HTMLSelectElement).value as CompressionMode)'>
          <option :value='CompressionMode.CompressionModeNone'>None</option>
          <option :value='CompressionMode.CompressionModeLz4'>Fast - lz4</option>
          <option :value='CompressionMode.CompressionModeZstd'>Modern - zstd</option>
          <option :value='CompressionMode.CompressionModeZlib'>Balanced - zlib</option>
          <option :value='CompressionMode.CompressionModeLzma'>Maximum - lzma</option>
        </select>
      </div>

      <!-- Compression Level Slider -->
      <div v-if='showCompressionLevelSlider' class='flex items-start gap-4'>
        <span class='label-text font-medium whitespace-nowrap'>Compression Level</span>
        <div class='flex-1'>
          <div class='flex items-center gap-2 mb-1'>
            <input
              type='range'
              :min='compressionLevelRange.min'
              :max='compressionLevelRange.max'
              :value='props.compressionLevel ?? compressionLevelRange.default'
              @input='(e) => onCompressionLevelChange(parseInt((e.target as HTMLInputElement).value))'
              class='range range-secondary flex-1' />
            <span class='label-text-alt'>{{ props.compressionLevel ?? compressionLevelRange.default }}</span>
          </div>
          <div class='flex justify-between text-xs text-base-content/70'>
            <span>Faster ({{ compressionLevelRange.min }})</span>
            <span>Smaller ({{ compressionLevelRange.max }})</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Algorithm Explanation Info Box -->
    <div v-if='algorithmExplanation' role='alert' :class="['alert alert-soft mt-6', isWarningLevel ? 'alert-warning' : 'alert-info']">
      <InformationCircleIcon class='size-5 shrink-0' />
      <div>
        <div><strong>{{ algorithmExplanation.name }}</strong>: {{ algorithmExplanation.description }}</div>
        <div class='text-sm mt-1'>Only affects future backups and can be changed anytime.</div>
      </div>
    </div>

    <!-- Compression Info Modal -->
    <CompressionInfoModal :ref='compressionInfoModalKey' />
  </div>
</template>
