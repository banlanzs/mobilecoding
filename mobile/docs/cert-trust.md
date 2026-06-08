# 证书信任配置指南

## Android

### 开发环境

在 `android/app/src/main/res/xml/network_security_config.xml` 中添加：

```xml
<?xml version="1.0" encoding="utf-8"?>
<network-security-config>
  <base-config cleartextTrafficPermitted="false">
    <trust-anchors>
      <certificates src="system" />
      <certificates src="user" />
    </trust-anchors>
  </base-config>
</network-security-config>
```

然后在 `AndroidManifest.xml` 的 `<application>` 标签中添加：

```xml
android:networkSecurityConfig="@xml/network_security_config"
```

### 生产环境

1. 将 CA 证书安装到设备
2. 通过 `adb push` 推送到设备
3. 在设备设置中信任证书

## iOS

### 开发环境

1. 在 Safari 中打开服务器 URL
2. 下载并安装证书
3. 进入 设置 → 通用 → 关于本机 → 证书信任设置
4. 启用对自签名证书的完全信任

### ATS 配置

在 `Info.plist` 中添加例外域（开发环境）：

```xml
<key>NSAppTransportSecurity</key>
<dict>
  <key>NSExceptionDomains</key>
  <dict>
    <key>your-server-ip</key>
    <dict>
      <key>NSExceptionAllowsInsecureHTTPLoads</key>
      <true/>
      <key>NSIncludesSubdomains</key>
      <true/>
    </dict>
  </dict>
</dict>
```

## 测试

连接后检查：

```bash
# Android
adb logcat | grep -i "ssl"

# iOS
Console.app - 搜索 SSL/TLS 错误
```
