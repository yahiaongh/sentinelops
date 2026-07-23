import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./components/**/*.{js,ts,jsx,tsx,mdx}",
    "./app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        bg: "#0B0E14",
        surface: "#131720",
        border: "#232838",
        text: "#E4E7EC",
        muted: "#8B93A7",
        accent: "#4FD1C5",
        critical: "#F2545B",
        warning: "#F5A623",
        healthy: "#3DD68C",
      },
      fontFamily: {
        sans: ["var(--font-sans)", "sans-serif"],
        mono: ["var(--font-mono)", "monospace"],
      },
    },
  },
  plugins: [],
};

export default config;