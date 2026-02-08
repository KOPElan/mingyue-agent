# 目录权限问题修复总结

## 问题描述

服务启动时报错：
```
Error: failed to create daemon: create server: create share manager: create backup directory: 
mkdir /var/lib/mingyue-agent: read-only file system
```

以及后续的：
```
Error: failed to create daemon: Required directories are not accessible:
  - network config: directory is not writable: /etc/mingyue-agent/network (read-only file system)
```

## 根本原因

1. **Systemd 安全配置**: 服务使用 `ProtectSystem=strict`，使整个文件系统只读
2. **缺少必需目录**: `/var/lib/mingyue-agent` 及其子目录未创建
3. **权限配置错误**: 即使目录存在，也可能没有正确的所有者和权限
4. **错误的设计思路**: 
   - 最初的回退机制将业务数据存储到临时目录，这会导致数据在系统重启后丢失
   - `/etc/mingyue-agent/network` 目录被配置为需要写入，但这违反了 `/etc` 只读的原则
   - 实际上 `netmanager` 的 `ConfigDir` 字段从未被使用，这是一个冗余配置

## 解决方案

### 1. 代码改进

#### 移除不当的回退机制
- ❌ **之前**: 失败时回退到 `/tmp` 或用户主目录
- ✅ **现在**: 明确报错，提供清晰的修复指引

#### 移除未使用的配置项
- 从 `netmanager` 中移除了从未使用的 `ConfigDir` 字段
- 移除了对 `/etc/mingyue-agent/network` 目录的创建和验证
- 简化了配置文件结构

#### 添加启动前预检查
在 `internal/daemon/daemon.go` 中添加了 `verifyDirectories()` 函数：
- 检查所有必需目录是否存在
- 验证目录是否可写
- 提供详细的错误信息和修复命令

#### 改进错误信息
所有模块现在都提供清晰的错误信息，包括：
- 失败的具体目录路径
- 修复命令示例
- 权限要求说明

### 2. 安装脚本改进

#### 完善目录创建 (`scripts/install.sh`)
```bash
# 创建所有必需的目录
mkdir -p /var/lib/mingyue-agent/share-backups
mkdir -p /etc/mingyue-agent/network

# 设置正确的所有者和权限
chown -R mingyue-agent:mingyue-agent /var/lib/mingyue-agent
chmod -R 755 /var/lib/mingyue-agent
```

#### 添加安装后验证
安装脚本现在会自动验证：
- 目录是否创建
- 权限是否正确
- 服务是否安装

### 3. 新增验证脚本

创建了 `scripts/verify-setup.sh`，可随时运行以验证：
- 用户和组是否存在
- 二进制文件是否安装
- 所有必需目录是否存在且权限正确
- 配置文件是否存在
- Systemd 服务是否安装和运行

### 4. 文档更新

在 `docs/DEPLOYMENT.md` 中添加：
- **目录结构章节**: 详细说明所有必需目录
- **权限要求**: 明确每个目录的所有者和权限
- **故障排查**: 添加常见权限问题的解决方案
- **验证步骤**: 如何验证安装是否正确

## 必需的目录结构

```
/etc/mingyue-agent/              # 配置文件 (root:root, 755)
└── config.yaml                  # 主配置文件 (644)

/var/log/mingyue-agent/          # 日志文件 (mingyue-agent:mingyue-agent, 755)

/var/run/mingyue-agent/          # 运行时文件 (mingyue-agent:mingyue-agent, 755)

/var/lib/mingyue-agent/          # 应用数据 (mingyue-agent:mingyue-agent, 755)
└── share-backups/              # 共享配置备份 (755)
```

## 使用指南

### 立即修复当前错误

在服务器上执行：

```bash
# 1. 创建所有必需的目录
sudo mkdir -p /var/lib/mingyue-agent/share-backups
sudo mkdir -p /var/log/mingyue-agent
sudo mkdir -p /var/run/mingyue-agent

# 2. 设置正确的所有者
sudo chown -R mingyue-agent:mingyue-agent /var/lib/mingyue-agent
sudo chown -R mingyue-agent:mingyue-agent /var/log/mingyue-agent
sudo chown -R mingyue-agent:mingyue-agent /var/run/mingyue-agent

# 3. 设置正确的权限
sudo chmod -R 755 /var/lib/mingyue-agent
sudo chmod -R 755 /var/log/mingyue-agent
sudo chmod -R 755 /var/run/mingyue-agent

# 4. 重启服务
sudo systemctl restart mingyue-agent

# 5. 查看日志确认正常运行
sudo journalctl -u mingyue-agent -f
```

### 新安装

```bash
# 1. 构建
make build

# 2. 运行安装脚本（已包含所有目录创建和验证）
sudo ./scripts/install.sh

# 3. 编辑配置
sudo vi /etc/mingyue-agent/config.yaml

# 4. 启动服务
sudo systemctl start mingyue-agent
```

### 验证现有安装

```bash
# 运行验证脚本
sudo ./scripts/verify-setup.sh
```

该脚本会检查所有必需的组件并提供修复建议。

## 代码变更清单

   - 移除对 `/etc/mingyue-agent/network` 的验证

2. **internal/netmanager/netmanager.go**
   - 移除未使用的 `configDir` 字段
   - 移除 `Config.ConfigDir` 字段
   - 简化 `New()` 函数

3. **internal/config/config.go**
   - 从 `NetworkConfig` 移除 `ConfigDir` 字段
   - 更新默认配置

4. **internal/server/server.go**
   - 移除创建 netmanager 时的 ConfigDir 参数

5. **internal/sharemanager/sharemanager.go**
   - 移除回退到临时目录的逻辑
   - 添加清晰的错误信息

6. **internal/netdisk/netdisk.go**
   - 改进 `saveState()` 错误处理
   - 添加清晰的错误信息

7. **internal/netmanager/netmanager.go**
   - 改进 `saveHistory()` 错误处理
   - 添加清晰的错误信息

8. **internal/auth/auth.go**
   - 移除回退机制
   - 添加清晰的错误信息

9. **internal/scheduler/scheduler.go**
   - 移除回退机制
   - 添加清晰的错误信息

10. **config.example.yaml**
    - 移除 `network.config_dir` 配置项

11. **scripts/install.sh**
    - 改进 `create_directories()` 函数
    - 添加 `verify_installation()` 函数
    - 创建所有必需的子目录
    - 移除对 `/et

2. **DIRECTORY_PERMISSIONS_FIX.md** (本文件)
   - 问题分析和修复总结c/mingyue-agent/network` 的创建

12. **scripts/verify-setup.sh**
    - 移除对 `/etc/mingyue-agent/network` 的检查

13. **docs/DEPLOYMENT.md**
    - 添加"目录结构和权限"章节
    - 扩展故障排查章节
    - 添加验证说明
    - 移除对 `/etc/mingyue-agent/network` 的引用的子目录

8. **docs/DEPLOYMENT.md**
   - 添加"目录结构和权限"章节
   - 扩展故障排查章节
   - 添加验证说明

### 新增的文件

1. **scripts/verify-setup.sh**
   - 完整的安装验证脚本
   - 检查所有组件和权限
   - 提供清晰的修复建议

## 设计原则

### ✅ 正确的做法

1. **持久化业务数据**: 所有业务数据必须存储在 `/var/lib/mingyue-agent`
2. **明确的错误信息**: 提供清晰的错误和修复指引
3. **启动前验证**: 在服务启动前检查所有前置条件
4. **安装时配置**: 在安装时创建所有必需的目录
5. **提供验证工具**: 让管理员可以随时验证配置

### ❌ 错误的做法

1. **回退到临时目录**: 业务数据不应存储在 `/tmp`（会在重启后丢失）
2. **静默失败**: 不要隐藏错误，应该明确报告
3. **假设目录存在**: 不要假设系统目录已正确配置
4. **延迟错误**: 应该在启动时就发现配置问题，而不是运行时

## 测试验证

所有修改的代码包都已通过编译验证：
- ✅ internal/sharemanager
- ✅ internal/netdisk
- ✅ internal/netmanager
- ✅ internal/auth
- ✅ internal/scheduler
- ✅ internal/daemon

## 后续建议

1. **在 CI/CD 中添加安装验证**: 自动运行 `verify-setup.sh`
2. **添加健康检查端点**: 包含目录权限检查
3. **监控目录空间**: 在磁盘空间不足时发出警告
4. **定期备份**: 自动备份 `/var/lib/mingyue-agent` 中的数据

## 参考资料

- [Linux Filesystem Hierarchy Standard](https://refspecs.linuxfoundation.org/FHS_3.0/fhs/index.html)
- [systemd Service Hardening](https://www.freedesktop.org/software/systemd/man/systemd.exec.html)
- [DEPLOYMENT.md](docs/DEPLOYMENT.md) - 完整部署指南
