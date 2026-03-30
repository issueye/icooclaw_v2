import { defineStore } from "pinia";
import { ref, watch } from "vue";

// 预定义的颜色主题
export const colorThemes = {
  blue: {
    name: "清新蓝",
    color: "#3b82f6",
    hover: "#2563eb",
    light: "rgba(59, 130, 246, 0.08)",
  },
  purple: {
    name: "优雅紫",
    color: "#8b5cf6",
    hover: "#7c3aed",
    light: "rgba(139, 92, 246, 0.08)",
  },
  green: {
    name: "自然绿",
    color: "#10b981",
    hover: "#059669",
    light: "rgba(16, 185, 129, 0.08)",
  },
  orange: {
    name: "活力橙",
    color: "#f97316",
    hover: "#ea580c",
    light: "rgba(249, 115, 22, 0.08)",
  },
  pink: {
    name: "浪漫粉",
    color: "#ec4899",
    hover: "#db2777",
    light: "rgba(236, 72, 153, 0.08)",
  },
  cyan: {
    name: "青碧色",
    color: "#06b6d4",
    hover: "#0891b2",
    light: "rgba(6, 182, 212, 0.08)",
  },
  red: {
    name: "热情红",
    color: "#ef4444",
    hover: "#dc2626",
    light: "rgba(239, 68, 68, 0.08)",
  },
  indigo: {
    name: "靛蓝色",
    color: "#6366f1",
    hover: "#4f46e5",
    light: "rgba(99, 102, 241, 0.08)",
  },
};

export const useThemeStore = defineStore("theme", () => {
  // 明暗主题
  const theme = ref(localStorage.getItem("theme") || "light");
  // 颜色主题
  const colorTheme = ref(localStorage.getItem("colorTheme") || "blue");

  const setTheme = (newTheme) => {
    theme.value = newTheme;
    localStorage.setItem("theme", newTheme);
    applyTheme(newTheme);
  };

  const toggleTheme = () => {
    const newTheme = theme.value === "dark" ? "light" : "dark";
    setTheme(newTheme);
  };

  const setColorTheme = (newColorTheme) => {
    colorTheme.value = newColorTheme;
    localStorage.setItem("colorTheme", newColorTheme);
    applyColorTheme(newColorTheme);
  };

  const applyTheme = (themeName) => {
    document.documentElement.setAttribute("data-theme", themeName);
  };

  const applyColorTheme = (colorThemeName) => {
    const colors = colorThemes[colorThemeName];
    if (colors) {
      document.documentElement.style.setProperty("--color-accent", colors.color);
      document.documentElement.style.setProperty("--color-accent-hover", colors.hover);
      document.documentElement.style.setProperty("--color-accent-light", colors.light);
    }
  };

  const initTheme = () => {
    applyTheme(theme.value);
    applyColorTheme(colorTheme.value);
  };

  // 获取当前颜色主题信息
  const getCurrentColorTheme = () => {
    return colorThemes[colorTheme.value] || colorThemes.blue;
  };

  // 获取所有颜色主题列表
  const getColorThemeList = () => {
    return Object.entries(colorThemes).map(([key, value]) => ({
      key,
      ...value,
    }));
  };

  return {
    theme,
    colorTheme,
    setTheme,
    toggleTheme,
    setColorTheme,
    initTheme,
    getCurrentColorTheme,
    getColorThemeList,
    colorThemes,
  };
});