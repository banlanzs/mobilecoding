module.exports = {
  preset: 'react-native',
  roots: ['<rootDir>/src'],
  transformIgnorePatterns: [
    'node_modules/(?!(@react-native|react-native|@react-navigation)/)'
  ],
  moduleNameMapper: {
    '^react-native-marked$': '<rootDir>/src/__mocks__/react-native-marked.js'
  },
  setupFilesAfterEnv: ['@testing-library/jest-native/extend-expect']
}
