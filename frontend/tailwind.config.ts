import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}", "./lib/**/*.ts"],
  theme: {
    extend: {
      colors: {
        ink: "#000000",
        coal: "#131417",
        smoke: "#1C1E23",
        bone: "#FAFAF7",
        paper: "#EFEDE6",
        blood: "#E3120B",
        bruise: "#7A0A06",
        steel: "#9CA3AF",
        ash: "#6B7280",
      },
      fontFamily: {
        display: ["var(--font-display)", "Impact", "sans-serif"],
        body: ["var(--font-body)", "system-ui", "sans-serif"],
      },
      boxShadow: {
        hard: "4px 4px 0 0 #000000",
        "hard-lg": "8px 8px 0 0 #000000",
        "hard-sm": "3px 3px 0 0 #000000",
        blood: "4px 4px 0 0 #E3120B",
        "blood-lg": "10px 10px 0 0 #E3120B",
      },
      borderWidth: {
        3: "3px",
      },
      outlineWidth: {
        3: "3px",
      },
    },
  },
  plugins: [],
};

export default config;
