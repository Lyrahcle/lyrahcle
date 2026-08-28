export default {
  content: ["./index.html", "./src/**/*.{js,jsx}"],
  darkMode: ["class"],
  theme: {
    extend: {
      colors: {
        lyra: {
          bg: "rgb(var(--lyra-bg) / <alpha-value>)",
          bgSoft: "rgb(var(--lyra-bg-soft) / <alpha-value>)",
          panel: "rgb(var(--lyra-panel) / <alpha-value>)",
          border: "rgb(var(--lyra-border) / <alpha-value>)",
          borderSoft: "rgb(var(--lyra-border-soft) / <alpha-value>)",
          gold: "rgb(var(--lyra-gold) / <alpha-value>)",
          goldBright: "rgb(var(--lyra-gold-bright) / <alpha-value>)",
          text: "rgb(var(--lyra-text) / <alpha-value>)",
          muted: "rgb(var(--lyra-muted) / <alpha-value>)",
          mutedDim: "rgb(var(--lyra-muted-dim) / <alpha-value>)",
          good: "rgb(var(--lyra-good) / <alpha-value>)",
          warn: "rgb(var(--lyra-warn) / <alpha-value>)",
          bad: "rgb(var(--lyra-bad) / <alpha-value>)",
        },
      },
      fontFamily: {
        display: ["'Space Grotesk'", "sans-serif"],
        body: ["Inter", "sans-serif"],
        mono: ["'JetBrains Mono'", "monospace"],
      },
      backgroundImage: {
        stars:
          "radial-gradient(1px 1px at 20px 30px, rgba(236,231,245,0.25), transparent), radial-gradient(1px 1px at 90px 80px, rgba(236,231,245,0.18), transparent), radial-gradient(1.5px 1.5px at 150px 40px, rgba(236,231,245,0.2), transparent), radial-gradient(1px 1px at 210px 120px, rgba(236,231,245,0.15), transparent)",
      },
      backgroundSize: {
        stars: "240px 160px",
      },
    },
  },
  plugins: [],
}
