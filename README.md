# aria2bango

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

一个用于自动检测和阻止吸血BT客户端的Go程序。通过aria2 RPC接口监控BT下载的peer连接，基于行为分析自动识别并屏蔽吸血客户端。

## 功能特性

- 🔍 **行为分析检测**
  - 基于分享率（上传/下载比例）检测吸血行为

- 🛡️ **智能屏蔽**
  - 使用nftables进行IP屏蔽
  - 支持自动过期（内核级别timeout）
  - 程序退出自动清理规则

- ⚖️ **累加惩罚机制**
  - 第1次：基础时长 × 1
  - 第2次：基础时长 × 2
  - 第3次：基础时长 × 3
  - 以此类推...

- 📝 **完整日志**
  - JSON格式结构化日志
  - 记录屏蔽原因、违规次数、分享率等

- ⚙️ **灵活配置**
  - YAML配置文件
  - 可自定义行为阈值
  - 可配置基础屏蔽时长

## 系统要求

- Linux系统（需要nftables支持）
- Go 1.21+（编译时）
- aria2（需要开启RPC接口）
- root权限或CAP_NET_ADMIN能力

## 快速开始

### 安装

```bash
# 克隆仓库
git clone https://github.com/lbl1m/aria2bango.git
cd aria2bango

# 编译
make build

# 安装到系统
sudo make install
```

### 配置

编辑配置文件 `/etc/aria2bango/config.yaml`：

```yaml
# aria2 RPC配置
aria2:
  host: "127.0.0.1"
  port: 6800
  secret: ""              # 如果aria2设置了RPC密钥，填写此处
  poll_interval: 10s      # 检测间隔

# 检测配置
detection:
  behavior:
    enabled: true
    min_share_ratio: 0.1  # 分享率阈值

# 屏蔽配置
blocking:
  base_duration: 5m       # 基础屏蔽时长
```

### 启动服务

```bash
# 启动服务
sudo systemctl start aria2bango

# 设置开机自启
sudo systemctl enable aria2bango

# 查看状态
sudo systemctl status aria2bango
```

### 手动运行

```bash
# 使用默认配置
sudo ./bin/aria2bango

# 指定配置文件
sudo ./bin/aria2bango -config /path/to/config.yaml

# 清理nftables规则
sudo ./bin/aria2bango -cleanup
```

## 配置说明

### aria2 RPC配置

| 字段 | 说明 | 默认值 |
|------|------|--------|
| host | aria2 RPC地址 | 127.0.0.1 |
| port | aria2 RPC端口 | 6800 |
| secret | RPC密钥 | 空 |
| poll_interval | 轮询间隔 | 10s |

### 行为分析配置

```yaml
detection:
  behavior:
    enabled: true               # 是否启用行为分析
    min_share_ratio: 0.1        # 最小分享率阈值
    min_data_threshold: 10485760 # 最小统计量（字节）
```

**分享率说明**：
- 分享率 = peer上传给我们的数据 / peer从我们下载的数据
- 低分享率意味着peer下载多但上传少，是典型的吸血行为
- 默认阈值0.1表示：peer每下载10字节只上传1字节

### 屏蔽配置

| 字段 | 说明 | 默认值 |
|------|------|--------|
| base_duration | 基础屏蔽时长 | 5m |
| nft_table | nftables表名 | aria2bango |

**累加惩罚说明**：
- 第1次检测到吸血：屏蔽 1 × base_duration
- 第2次检测到吸血：屏蔽 2 × base_duration
- 第3次检测到吸血：屏蔽 3 × base_duration
- 以此类推...

### 日志配置

| 字段 | 说明 | 默认值 |
|------|------|--------|
| level | 日志级别 | info |
| file | 日志文件路径 | /var/log/aria2bango/blocked.log |
| max_size | 单文件最大大小(MB) | 100 |
| max_backups | 保留旧文件数量 | 3 |
| max_age | 保留天数 | 30 |

## 日志格式

屏蔽事件以JSON格式记录：

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "event": "blocked",
  "ip": "192.168.1.100",
  "peer_id": "-XL0019-xxx",
  "client_name": "Unknown",
  "reason": "low_share_ratio",
  "duration": "15m0s",
  "download_speed": 1048576,
  "upload_speed": 1024,
  "share_ratio": 0.001
}
```

### 字段说明

| 字段 | 说明 |
|------|------|
| timestamp | 时间戳 |
| event | 事件类型（blocked/unblocked） |
| ip | 被屏蔽的IP地址 |
| peer_id | Peer ID |
| client_name | 客户端名称（行为分析时为Unknown） |
| reason | 屏蔽原因（low_share_ratio） |
| duration | 屏蔽时长 |
| download_speed | 下载速度 |
| upload_speed | 上传速度 |
| share_ratio | 分享率 |

## 工作原理

```
┌─────────────────────────────────────────────────────────┐
│                      aria2bango                         │
├─────────────────────────────────────────────────────────┤
│  ┌──────────────┐    ┌──────────────┐                   │
│  │ aria2 RPC    │───▶│ Peer检测器   │                   │
│  │ 客户端       │    │ (行为分析)   │                   │
│  └──────────────┘    └──────┬───────┘                   │
│                             │                           │
│                             ▼                           │
│                     ┌──────────────┐                    │
│                     │ 屏蔽决策器   │                    │
│                     │ (累加惩罚)   │                    │
│                     └──────┬───────┘                    │
│                             │                           │
│              ┌──────────────┴──────────────┐            │
│              ▼                             ▼            │
│     ┌──────────────┐              ┌──────────────┐      │
│     │ nftables     │              │ 日志记录器   │      │
│     │ (output链)   │              │              │      │
│     └──────────────┘              └──────────────┘      │
└─────────────────────────────────────────────────────────┘
```

1. 通过aria2 RPC获取活动下载任务的peer列表
2. 对每个peer进行行为分析：
   - 计算分享率 = peer上传 / peer下载
   - 如果分享率低于阈值，判定为吸血行为
3. 检测到吸血客户端后：
   - 增加违规次数
   - 计算屏蔽时长 = 违规次数 × 基础时长
   - 添加IP到nftables屏蔽集合（带超时）
   - 记录屏蔽事件到日志文件
4. nftables自动处理过期IP的释放

### 为什么只阻止往外发包？

- **不影响下载**：我们仍然可以从吸血客户端下载数据
- **阻止上传**：阻止他们从我们这里获取数据
- **惩罚与保护并重**：既惩罚吸血行为，又不影响我们的下载体验

### 为什么移除特征匹配？

- **误判风险**：即使是迅雷等客户端，也不一定都在吸血
- **特征过时**：客户端特征可能随版本更新而变化
- **规避容易**：吸血客户端可以轻易修改Peer ID来规避检测
- **行为分析更公平**：只看实际行为，不看出身

## 开发

```bash
# 下载依赖
make deps

# 编译
make build

# 运行测试
make test

# 代码格式化
make fmt

# 代码检查
make lint
```

## 注意事项

1. **权限要求**：程序需要root权限或CAP_NET_ADMIN能力来操作nftables
2. **aria2配置**：确保aria2开启了RPC接口，且允许本地连接
3. **防火墙冲突**：如果系统使用其他防火墙工具（如ufw、firewalld），请注意规则冲突
4. **IPv6支持**：程序同时支持IPv4和IPv6地址的屏蔽

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request！
