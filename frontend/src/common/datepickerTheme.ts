/**
 * Applies custom color theme overrides for vue-tailwind-datepicker.
 * Uses MutationObserver to re-inject overrides whenever new styles are added,
 * ensuring our colors always win regardless of when the datepicker loads.
 */

const OVERRIDE_CSS = `
  /* === 1. CSS VARIABLE OVERRIDES (purple colors) === */
  :root, [data-theme=light] {
    --color-vtd-primary-50: #faf5ff !important;
    --color-vtd-primary-100: #f3e8ff !important;
    --color-vtd-primary-200: #e9d5ff !important;
    --color-vtd-primary-300: #d8b4fe !important;
    --color-vtd-primary-400: #c084fc !important;
    --color-vtd-primary-500: #a855f7 !important;
    --color-vtd-primary-600: #9333ea !important;
    --color-vtd-primary-700: #7e22ce !important;
    --color-vtd-primary-800: #6b21a8 !important;
    --color-vtd-primary-900: #581c87 !important;
    --color-vtd-primary-950: #3b0764 !important;
    --color-vtd-secondary-50: #f9fafb !important;
    --color-vtd-secondary-100: #f3f4f6 !important;
    --color-vtd-secondary-200: #e5e7eb !important;
    --color-vtd-secondary-300: #d1d5db !important;
    --color-vtd-secondary-400: #9ca3af !important;
    --color-vtd-secondary-500: #6b7280 !important;
    --color-vtd-secondary-600: #4b5563 !important;
    --color-vtd-secondary-700: #374151 !important;
    --color-vtd-secondary-800: #1f2937 !important;
    --color-vtd-secondary-900: #111827 !important;
    --color-vtd-secondary-950: #030712 !important;
  }

  [data-theme=dark] {
    --color-vtd-primary-50: #faf5ff !important;
    --color-vtd-primary-100: #f3e8ff !important;
    --color-vtd-primary-200: #e9d5ff !important;
    --color-vtd-primary-300: #d8b4fe !important;
    --color-vtd-primary-400: #c084fc !important;
    --color-vtd-primary-500: #a855f7 !important;
    --color-vtd-primary-600: #9333ea !important;
    --color-vtd-primary-700: #7e22ce !important;
    --color-vtd-primary-800: #6b21a8 !important;
    --color-vtd-primary-900: #581c87 !important;
    --color-vtd-primary-950: #3b0764 !important;
    --color-vtd-secondary-50: #f9fafb !important;
    --color-vtd-secondary-100: #f3f4f6 !important;
    --color-vtd-secondary-200: #e5e7eb !important;
    --color-vtd-secondary-300: #d1d5db !important;
    --color-vtd-secondary-400: #9ca3af !important;
    --color-vtd-secondary-500: #6b7280 !important;
    --color-vtd-secondary-600: #4b5563 !important;
    --color-vtd-secondary-700: #374151 !important;
    --color-vtd-secondary-800: #1f2937 !important;
    --color-vtd-secondary-900: #111827 !important;
    --color-vtd-secondary-950: #030712 !important;
  }

  /* === 2. NEUTRALIZE @media (prefers-color-scheme: dark) === */
  /* When system is in dark mode but app is in light theme, reset dark: utilities to light values */
  @media (prefers-color-scheme: dark) {
    .dark\\:bg-vtd-secondary-700,
    .dark\\:bg-vtd-secondary-700\\/50 { background-color: #ffffff !important; }
    .dark\\:bg-vtd-secondary-800 { background-color: #f9fafb !important; }
    .dark\\:border-vtd-secondary-600 { border-color: #d1d5db !important; }
    .dark\\:border-vtd-secondary-700,
    .dark\\:border-vtd-secondary-700\\/\\[1\\] { border-color: #d1d5db !important; }
    .dark\\:text-vtd-primary-400 { color: #9333ea !important; }
    .dark\\:text-vtd-secondary-100 { color: #374151 !important; }
    .dark\\:text-vtd-secondary-200 { color: #4b5563 !important; }
    .dark\\:text-vtd-secondary-300 { color: #6b7280 !important; }
    .dark\\:text-vtd-secondary-400 { color: #6b7280 !important; }
    .dark\\:placeholder-vtd-secondary-500::placeholder { color: #9ca3af !important; }
    .dark\\:ring-offset-vtd-secondary-800 { --tw-ring-offset-color: #ffffff !important; }
    .dark\\:hover\\:bg-vtd-secondary-700:hover { background-color: #f3f4f6 !important; }
    .dark\\:hover\\:text-vtd-primary-300:hover { color: #7e22ce !important; }
    .dark\\:focus\\:bg-vtd-secondary-700:focus { background-color: #f3f4f6 !important; }
    .dark\\:focus\\:text-vtd-primary-300:focus { color: #9333ea !important; }
  }

  /* === 3. RE-APPLY dark: utilities ONLY for [data-theme=dark] === */
  /* This takes precedence over the media query neutralization above */
  [data-theme=dark] .dark\\:bg-vtd-secondary-700,
  [data-theme=dark] .dark\\:bg-vtd-secondary-700\\/50 { background-color: var(--color-vtd-secondary-700) !important; }
  [data-theme=dark] .dark\\:bg-vtd-secondary-800 { background-color: var(--color-vtd-secondary-800) !important; }
  [data-theme=dark] .dark\\:border-vtd-secondary-600 { border-color: var(--color-vtd-secondary-600) !important; }
  [data-theme=dark] .dark\\:border-vtd-secondary-700,
  [data-theme=dark] .dark\\:border-vtd-secondary-700\\/\\[1\\] { border-color: var(--color-vtd-secondary-700) !important; }
  [data-theme=dark] .dark\\:text-vtd-primary-400 { color: var(--color-vtd-primary-400) !important; }
  [data-theme=dark] .dark\\:text-vtd-secondary-100 { color: var(--color-vtd-secondary-100) !important; }
  [data-theme=dark] .dark\\:text-vtd-secondary-200 { color: var(--color-vtd-secondary-200) !important; }
  [data-theme=dark] .dark\\:text-vtd-secondary-300 { color: var(--color-vtd-secondary-300) !important; }
  [data-theme=dark] .dark\\:text-vtd-secondary-400 { color: var(--color-vtd-secondary-400) !important; }
  [data-theme=dark] .dark\\:placeholder-vtd-secondary-500::placeholder { color: var(--color-vtd-secondary-500) !important; }
  [data-theme=dark] .dark\\:ring-offset-vtd-secondary-800 { --tw-ring-offset-color: var(--color-vtd-secondary-800) !important; }
  [data-theme=dark] .dark\\:hover\\:bg-vtd-secondary-700:hover { background-color: var(--color-vtd-secondary-700) !important; }
  [data-theme=dark] .dark\\:hover\\:text-vtd-primary-300:hover { color: var(--color-vtd-primary-300) !important; }
  [data-theme=dark] .dark\\:focus\\:bg-vtd-secondary-700:focus { background-color: var(--color-vtd-secondary-700) !important; }
  [data-theme=dark] .dark\\:focus\\:text-vtd-primary-300:focus { color: var(--color-vtd-primary-300) !important; }
`;

let overrideStyleEl: HTMLStyleElement | null = null;

function injectOverrideStyles() {
  // Remove existing override if present
  if (overrideStyleEl?.parentNode) {
    overrideStyleEl.parentNode.removeChild(overrideStyleEl);
  }

  // Create and append new override (ensures it's last in <head>)
  overrideStyleEl = document.createElement("style");
  overrideStyleEl.setAttribute("data-arco-datepicker-override", "");
  overrideStyleEl.textContent = OVERRIDE_CSS;
  document.head.appendChild(overrideStyleEl);
}

export function applyDatepickerThemeOverrides() {
  if (typeof document === "undefined") return;

  // Initial injection
  injectOverrideStyles();

  // Watch for new style elements being added (datepicker injects CSS on component load)
  const observer = new MutationObserver((mutations) => {
    for (const mutation of mutations) {
      const addedNodes = Array.from(mutation.addedNodes);
      for (const node of addedNodes) {
        // If a new style element was added (not ours), re-inject our overrides
        if (
          node instanceof HTMLStyleElement &&
          !node.hasAttribute("data-arco-datepicker-override")
        ) {
          injectOverrideStyles();
          return; // Only need to re-inject once per batch
        }
      }
    }
  });

  observer.observe(document.head, { childList: true });
}
