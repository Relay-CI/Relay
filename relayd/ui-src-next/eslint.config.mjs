import nextVitals from "eslint-config-next/core-web-vitals";

const config = [...nextVitals, {
  rules: {
    "import/no-anonymous-default-export": "off",
    "react-hooks/set-state-in-effect": "off",
    "react-hooks/purity": "off",
  },
}];

export default config;
