import React, { useState, useEffect } from 'react'
import { Modal, View, Text, ScrollView, Pressable, ActivityIndicator, SafeAreaView } from 'react-native'

interface GitFileStatus {
  path: string
  status: string
  staged: boolean
}

interface GitDiffModalProps {
  visible: boolean
  onClose: () => void
  host: string
  port: string
  token: string
  useWss: boolean
}

export function GitDiffModal({ visible, onClose, host, port, token, useWss }: GitDiffModalProps) {
  const [files, setFiles] = useState<GitFileStatus[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedFile, setSelectedFile] = useState<string | null>(null)
  const [diffContent, setDiffContent] = useState<string | null>(null)
  const [loadingDiff, setLoadingDiff] = useState(false)
  const [cwd, setCwd] = useState<string>('')

  // 获取 cwd（工作目录）
  useEffect(() => {
    if (!visible) return
    const scheme = useWss ? 'https' : 'http'
    fetch(`${scheme}://${host}:${port}/version`)
      .then(res => res.ok ? res.json() : Promise.reject(new Error(`HTTP ${res.status}`)))
      .then((data: any) => {
        const resolvedCwd = data?.runtime?.cwd || ''
        setCwd(resolvedCwd)
        return resolvedCwd
      })
      .then((resolvedCwd: string) => {
        setLoading(true)
        const url = `${scheme}://${host}:${port}/api/v1/git/status?cwd=${encodeURIComponent(resolvedCwd)}`
        return fetch(url, { headers: { 'Authorization': `Bearer ${token}` } })
      })
      .then(res => res.ok ? res.json() : Promise.reject(new Error(`HTTP ${res.status}`)))
      .then((data: GitFileStatus[]) => {
        setFiles(data || [])
      })
      .catch(err => {
        console.warn('[GitDiff] 获取文件列表失败:', err?.message)
      })
      .finally(() => setLoading(false))
  }, [visible])

  // 获取 diff 内容
  const loadDiff = (filePath: string) => {
    setSelectedFile(filePath)
    setLoadingDiff(true)
    setDiffContent(null)
    const scheme = useWss ? 'https' : 'http'
    const url = `${scheme}://${host}:${port}/api/v1/git/diff?cwd=${encodeURIComponent(cwd)}&file=${encodeURIComponent(filePath)}`
    fetch(url, { headers: { 'Authorization': `Bearer ${token}` } })
      .then(res => res.ok ? res.json() : Promise.reject(new Error(`HTTP ${res.status}`)))
      .then((data: { diff: string }) => {
        setDiffContent(data.diff || '(无差异)')
      })
      .catch(err => {
        console.warn('[GitDiff] 获取 diff 失败:', err?.message)
        setDiffContent(`获取失败: ${err?.message}`)
      })
      .finally(() => setLoadingDiff(false))
  }

  // 重置状态
  const handleClose = () => {
    setSelectedFile(null)
    setDiffContent(null)
    onClose()
  }

  // 文件状态图标
  const statusIcon = (status: string, staged: boolean) => {
    if (staged) return '✓'
    if (status === 'M') return '●'
    if (status === 'D') return '✗'
    return '?'
  }

  return (
    <Modal visible={visible} animationType="slide" onRequestClose={handleClose}>
      <SafeAreaView style={{ flex: 1, backgroundColor: '#fff' }}>
        {/* 顶栏 */}
        <View style={{
          paddingHorizontal: 16, paddingVertical: 12,
          borderBottomWidth: 1, borderBottomColor: '#e5e5e5',
          flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center',
          backgroundColor: '#f7f7f7'
        }}>
          <Text style={{ fontSize: 17, fontWeight: '600', color: '#333' }}>
            Git Diff{files.length > 0 ? ` (${files.length})` : ''}
          </Text>
          <Pressable onPress={handleClose} style={({ pressed }) => ({
            padding: 8, borderRadius: 16, backgroundColor: pressed ? '#ddd' : 'transparent'
          })}>
            <Text style={{ fontSize: 20, color: '#666' }}>✕</Text>
          </Pressable>
        </View>

        {/* 主内容区 */}
        <View style={{ flex: 1, flexDirection: 'row' }}>
          {/* 左侧文件列表 */}
          <ScrollView style={{ width: '35%', backgroundColor: '#fafafa', borderRightWidth: 1, borderRightColor: '#e5e5e5' }}>
            {loading ? (
              <ActivityIndicator style={{ marginTop: 40 }} />
            ) : files.length === 0 ? (
              <Text style={{ padding: 16, color: '#999', fontSize: 13 }}>无变更文件</Text>
            ) : (
              files.map((file) => (
                <Pressable
                  key={file.path}
                  onPress={() => loadDiff(file.path)}
                  style={({ pressed }) => ({
                    paddingHorizontal: 12, paddingVertical: 10,
                    backgroundColor: selectedFile === file.path ? '#e3f2fd' : (pressed ? '#f0f0f0' : 'transparent'),
                    borderBottomWidth: 0.5, borderBottomColor: '#eee',
                    flexDirection: 'row', alignItems: 'center', gap: 6,
                  })}
                >
                  <Text style={{
                    fontSize: 13, color: file.staged ? '#2e7d32' : '#f57c00',
                    fontWeight: '600', fontFamily: 'monospace',
                  }}>
                    {statusIcon(file.status, file.staged)}
                  </Text>
                  <Text numberOfLines={1} style={{ fontSize: 13, color: '#333', flexShrink: 1 }}>
                    {file.path.split('/').pop()}
                  </Text>
                  <Text style={{ fontSize: 10, color: '#999', marginLeft: 'auto', fontFamily: 'monospace' }}>
                    {file.status}
                  </Text>
                </Pressable>
              ))
            )}
          </ScrollView>

          {/* 右侧 diff 内容 */}
          <ScrollView style={{ width: '65%', backgroundColor: '#fff' }} contentContainerStyle={{ padding: 12 }}>
            {selectedFile ? (
              <>
                <Text style={{ fontSize: 14, fontWeight: '600', marginBottom: 8, color: '#333' }}>
                  {selectedFile.split('/').pop()}
                </Text>
                {loadingDiff ? (
                  <ActivityIndicator style={{ marginTop: 30 }} />
                ) : (
                  <ScrollView horizontal>
                    <Text style={{ fontFamily: 'monospace', fontSize: 12, lineHeight: 18, color: '#333' }}>
                      {diffContent || '(无差异)'}
                    </Text>
                  </ScrollView>
                )}
              </>
            ) : (
              <View style={{ flex: 1, justifyContent: 'center', alignItems: 'center', marginTop: 80 }}>
                <Text style={{ fontSize: 14, color: '#999' }}>← 点击文件查看 diff</Text>
              </View>
            )}
          </ScrollView>
        </View>
      </SafeAreaView>
    </Modal>
  )
}
