import nextVitals from "eslint-config-next/core-web-vitals";

const config = [
  {
    ignores: ["**/._*", "**/.__*", "**/*..__*", "**/.DS_Store", "**/Thumbs.db"],
  },
  ...nextVitals,
  {
    rules: {
      "import/no-anonymous-default-export": "off",
      "react-hooks/set-state-in-effect": "off",
      "react-hooks/purity": "off",
    },
  },
];

export default config;
