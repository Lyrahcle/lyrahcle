export default {
  content: ["./index.html", "./src/**/*.{js,jsx}"],
  theme: {
    extend: {
      colors: {
        lyra: {
          bg: "#0f0c1a",
          bgSoft: "#141024",
          panel: "#17132a",
          border: "#2b2447",
          borderSoft: "#221c3d",
          gold: "#c9a56e",
          goldBright: "#f0d9a8",
          text: "#ece7f5",
          muted: "#877fa0",
          mutedDim: "#5f597a",
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
