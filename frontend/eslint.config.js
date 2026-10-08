import vue from 'eslint-plugin-vue'
import globals from 'globals'

export default [
  { ignores: ['dist/**', 'node_modules/**', '*.d.ts'] },
  ...vue.configs['flat/essential'],
  {
    files: ['src/**/*.js', 'src/**/*.vue'],
    languageOptions: {
      globals: { ...globals.browser, ElMessage: 'readonly', ElMessageBox: 'readonly' },
    },
    rules: { 'no-unreachable': 'error', 'no-duplicate-imports': 'error', 'no-undef': 'error' },
  },
  { rules: { 'vue/no-mutating-props': ['error', { shallowOnly: true }] } },
  { files: ['tests/**/*.js'], languageOptions: { globals: globals.node } },
]
