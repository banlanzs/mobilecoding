// Jest mock: react-native-marked 依赖 marked/github-slugger 等 ESM 包，
// 在 jest 环境难以转换。测试不关心 Markdown 渲染细节，故 mock 为纯文本。
import React from 'react'
import { Text } from 'react-native'

module.exports = {
  __esModule: true,
  default: function MockMarkdown(props) {
    return React.createElement(Text, null, props.value)
  },
}
