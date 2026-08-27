export default {
  content: ["./index.html", "./src/**/*.{js,jsx}"],
  theme: {
    extend: {
      colors: {
        lyra: {
          bg: "#0a0a0c",
          bgSoft: "#101012",
          panel: "#141416",
          border: "#26262a",
          borderSoft: "#1c1c1f",
          gold: "#c9a56e",
          goldBright: "#f0d9a8",
          text: "#ececeb",
          muted: "#8c8c90",
          mutedDim: "#5c5c60",
          good: "#7dd3a0",
          warn: "#e8b563",
          bad: "#e0748c",
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
