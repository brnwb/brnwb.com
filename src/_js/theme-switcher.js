(() => {
  const storageKey = "theme";
  const allowedThemes = new Set(["system", "light", "dark"]);
  const root = document.documentElement;

  const normalizeTheme = (value) => (allowedThemes.has(value) ? value : "system");

  const getStoredTheme = () => {
    const stored = localStorage.getItem(storageKey);
    if (stored == null) {
      return "system";
    }

    const normalized = normalizeTheme(stored);
    if (normalized != stored) {
      // Invalid persisted values are reset back to the default behavior.
      localStorage.removeItem(storageKey);
    }

    return normalized;
  };

  const setStoredTheme = (value) => {
    const theme = normalizeTheme(value);
    if (theme === "system") {
      localStorage.removeItem(storageKey);
      return;
    }
    localStorage.setItem(storageKey, theme);
  };

  const getSystemTheme = () =>
    window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches
      ? "dark"
      : "light";

  const applyTheme = (storedTheme) => {
    root.dataset.theme = storedTheme === "system" ? getSystemTheme() : storedTheme;
  };

  // Apply theme as early as possible to avoid flash.
  applyTheme(getStoredTheme());

  document.addEventListener("DOMContentLoaded", () => {
    const selector = document.querySelector(".theme-selector");
    const select = document.querySelector("#theme-select");
    const currentTheme = getStoredTheme();

    if (selector) {
      selector.removeAttribute("hidden");
      document.querySelectorAll('input[name="theme"]').forEach((input) => {
        input.checked = input.value === currentTheme;
        input.addEventListener("change", () => {
          setStoredTheme(input.value);
          applyTheme(getStoredTheme());
        });
      });
    }

    if (select) {
      select.value = currentTheme;
      select.addEventListener("change", () => {
        setStoredTheme(select.value);
        const nextTheme = getStoredTheme();
        select.value = nextTheme;
        applyTheme(nextTheme);
      });
    }

    if (window.matchMedia) {
      const media = window.matchMedia("(prefers-color-scheme: dark)");
      media.addEventListener("change", () => {
        if (getStoredTheme() === "system") {
          applyTheme("system");
        }
      });
    }
  });
})();
