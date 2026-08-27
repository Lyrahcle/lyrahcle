export default {
  content: ["./index.html", "./src/**/*.{js,jsx}"],
  theme: {
    extend: {
      colors: {
        lyra: {
          bg: "#241a33",
          panel: "#2e2140",
          gold: "#c9a878",
          goldLight: "#e3cba8",
          text: "#f1ece2",
          muted: "#a89bb8",
        },
      },
      fontFamily: {
        display: ["Georgia", "serif"],
      },
    },
  },
  plugins: [],
}
