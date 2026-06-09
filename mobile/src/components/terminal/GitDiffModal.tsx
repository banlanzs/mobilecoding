import React, { useState, useEffect, useCallback } from 'react'
import { Modal, View, Text, FlatList, Pressable, ActivityIndicator, SafeAreaView } from 'react-native'

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
  const [diffContent, setDiffContent] = useState<string>('')
  const [loadingDiff, setLoadingDiff] = useState(false)
  const [cwd, setCwd] = useState<string>('')

  const scheme = useWss ? 'https' : 'http'
  const baseUrl = `${scheme}://${host}:${port}`
  const authHeader = { 'Authorization': `Bearer ${token}` }

  useEffect(() => {
    if (!visible) return
    let cancelled = false

    const loadData = async () => {
      setLoading(true)
      try {
        // 获取 cwd
        const verRes = await fetch(`${baseUrl}/version`)
        if (!verRes.ok) throw new Error(`version HTTP ${verRes.status}`)
        const verData = await verRes.json()
        if (cancelled) return
        const resolvedCwd = verData?.runtime?.cwd || ''
        setCwd(resolvedCwd)

        // 获取文件列表
        const statusUrl = `${baseUrl}/api/v1/git/status?cwd=${encodeURIComponent(resolvedCwd)}`
        const statusRes = await fetch(statusUrl, { headers: authHeader })
        if (!statusRes.ok) throw new Error(`git/status HTTP ${statusRes.status}`)
        const statusData = await statusRes.json()
        if (cancelled) return
        setFiles(Array.isArray(statusData) ? statusData : [])
      } catch (err: any) {
        if (!cancelled) {
          console.warn('[GitDiff] 加载失败:', err?.message)
          setFiles([])
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    loadData()
    return () => { cancelled = true }
  }, [visible, baseUrl, token])

  const loadDiff = useCallback(async (filePath: string) => {
    setSelectedFile(filePath)
    setLoadingDiff(true)
    setDiffContent('')
    try {
      const url = `${baseUrl}/api/v1/git/diff?cwd=${encodeURIComponent(cwd)}&file=${encodeURIComponent(filePath)}`
      const res = await fetch(url, { headers: authHeader })
      if (!res.ok) throw new Error(`git/diff HTTP ${res.status}`)
      const data = await res.json()
      setDiffContent(data?.diff || '(无差异)')
    } catch (err: any) {
      console.warn('[GitDiff] diff 失败:', err?.message)
      setDiffContent(`获取失败: ${err?.message}`)
    } finally {
      setLoadingDiff(false)
    }
  }, [cwd, baseUrl, token])

  const handleClose = useCallback(() => {
    setSelectedFile(null)
    setDiffContent('')
    onClose()
  }, [onClose])

  const statusIcon = (status: string, staged: boolean) => {
    if (staged) return '✓'
    if (status === 'M') return '●'
    if (status === 'D') return '✗'
    return '?'
  }

  const renderFileItem = ({ item }: { item: GitFileStatus }) => (
    <Pressable
      onPress={() => loadDiff(item.path)}
      style={{
        paddingHorizontal: 12, paddingVertical: 10,
        backgroundColor: selectedFile === item.path ? '#e3f2fd' : 'transparent',
        borderBottomWidth: 0.5, borderBottomColor: '#eee',
        flexDirection: 'row', alignItems: 'center', gap: 6,
      }}
    >
      <Text style={{
        fontSize: 13, color: item.staged ? '#2e7d32' : '#f57c00',
        fontWeight: '600', fontFamily: 'monospace',
      }}>
        {statusIcon(item.status, item.staged)}
      </Text>
      <Text numberOfLines={1} style={{ fontSize: 13, color: '#333', flex: 1 }}>
        {item.path.split('/').pop()}
      </Text>
      <Text style={{ fontSize: 10, color: '#999', fontFamily: 'monospace' }}>
        {item.status}
      </Text>
    </Pressable>
  )

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
          <Pressable onPress={handleClose} style={{ padding: 8 }}>
            <Text style={{ fontSize: 20, color: '#666' }}>✕</Text>
          </Pressable>
        </View>

        {/* 主内容区 */}
        <View style={{ flex: 1, flexDirection: 'row' }}>
          {/* 左侧文件列表 */}
          <View style={{ width: '35%', backgroundColor: '#fafafa', borderRightWidth: 1, borderRightColor: '#e5e5e5' }}>
            {loading ? (
              <ActivityIndicator style={{ marginTop: 40 }} />
            ) : files.length === 0 ? (
              <Text style={{ padding: 16, color: '#999', fontSize: 13 }}>无变更文件</Text>
            ) : (
              <FlatList
                data={files}
                keyExtractor={(item) => item.path}
                renderItem={renderFileItem}
              />
            )}
          </View>

          {/* 右侧 diff 内容 */}
          <View style={{ width: '65%', backgroundColor: '#fff' }}>
            {selectedFile ? (
              <View style={{ flex: 1 }}>
                <View style={{ paddingHorizontal: 12, paddingVertical: 8, borderBottomWidth: 0.5, borderBottomColor: '#eee' }}>
                  <Text style={{ fontSize: 13, fontWeight: '600', color: '#333' }}>
                    {selectedFile.split('/').pop()}
                  </Text>
                </View>
                {loadingDiff ? (
                  <ActivityIndicator style={{ marginTop: 30 }} />
                ) : (
                  <FlatList
                    data={diffContent.split('\n')}
                    keyExtractor={(_, i) => String(i)}
                    renderItem={({ item }) => (
                      <Text style={{ fontFamily: 'monospace', fontSize: 11, lineHeight: 16, color: '#333', paddingHorizontal: 12 }}>
                        {item}
                      </Text>
                    )}
                  />
                )}
              </View>
            ) : (
              <View style={{ flex: 1, justifyContent: 'center', alignItems: 'center' }}>
                <Text style={{ fontSize: 14, color: '#999' }}>← 点击文件查看 diff</Text>
              </View>
            )}
          </View>
        </View>
      </SafeAreaView>
    </Modal>
  )
}
