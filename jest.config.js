export default {
  testEnvironment: "node",
  testMatch: ["**/tests/**/*.test.js"],
  collectCoverageFrom: ["src/**/*.js"],
  coverageThreshold: {
    global: {
      statements: 40,
      branches: 50,
      functions: 40,
      lines: 40,
    },
  },
};
