import React from 'react'
import { View, Text, ScrollView, Platform } from 'react-native'
import Markdown from 'react-native-marked'
import type { MarkedStyles } from 'react-native-marked'

const markedStyles: MarkedStyles = {
  text: { color: '#000', fontSize: 15, lineHeight: 22 },
  paragraph: { marginBottom: 8 },
  strong: { fontWeight: '700', color: '#000' },
  em: { fontStyle: 'italic' },
  h1: { fontSize: 20, fontWeight: '700', color: '#000', marginTop: 8, marginBottom: 6 },
  h2: { fontSize: 18, fontWeight: '700', color: '#000', marginTop: 6, marginBottom: 4 },
  h3: { fontSize: 16, fontWeight: '600', color: '#000', marginTop: 4, marginBottom: 4 },
  h4: { fontSize: 15, fontWeight: '600', color: '#000' },
  h5: { fontSize: 14, fontWeight: '600', color: '#000' },
  h6: { fontSize: 13, fontWeight: '600', color: '#666' },
  // 行内代码
  codespan: {
    fontFamily: Platform.OS === 'ios' ? 'Menlo' : 'monospace',
    fontSize: 13,
    color: '#c7254e',
    backgroundColor: '#f5f5f5',
    paddingHorizontal: 4,
    borderRadius: 3,
  },
  // 代码块容器：横向滚动、不自动断行
  code: {
    backgroundColor: '#f6f8fa',
    borderRadius: 6,
    paddingHorizontal: 10,
    paddingVertical: 8,
    marginVertical: 6,
  },
  blockquote: {
    backgroundColor: '#f9f9f9',
    borderLeftWidth: 3,
    borderLeftColor: '#d0d7de',
    paddingHorizontal: 10,
    paddingVertical: 6,
    marginVertical: 6,
  },
  link: { color: '#0969da', textDecorationLine: 'underline' },
  list: { marginVertical: 4 },
  li: { fontSize: 15, color: '#000', lineHeight: 22 },
}

interface MarkdownTextProps {
  text: string
}

/** Markdown 渲染：代码块横向滚动，正文自适应换行 */
export function MarkdownText({ text }: MarkdownTextProps) {
  if (!text || text.trim().length === 0) return null
  return (
    <View>
      <Markdown value={text} styles={markedStyles} />
    </View>
  )
}
