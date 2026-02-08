# 服务启动问题诊断

## 问题现象
服务启动后立即退出，日志显示：
```
mingyue-agent.service: Deactivated successfully.
```

## 诊断步骤

### 1. 查看完整的服务日志

```bash
# 查看最近的所有日志（包括错误）
sudo journalctl -u mingyue-agent -n 100 --no-pager

# 实时跟踪日志
sudo journalctl -u mingyue-agent -f
```

### 2. 手动运行程序查看详细错误

```bash
# 停止服务
sudo systemctl stop mingyue-agent

# 以 mingyue-agent 用户手动运行
sudo -u mingyue-agent /usr/local/bin/mingyue-agent start --config /etc/mingyue-agent/config.yaml

# 或者直接以 root 运行看错误
/usr/local/bin/mingyue-agent start --config /etc/mingyue-agent/config.yaml
```

### 3. 检查配置文件

```bash
# 查看配置文件内容
cat /etc/mingyue-agent/config.yaml

# 验证 YAML 语法
python3 -c "import yaml; yaml.safe_load(open('/etc/mingyue-agent/config.yaml'))" && echo "YAML syntax OK"
```

### 4. 验证所有必需目录

```bash
# 运行验证脚本
cd /path/to/mingyue-agent
sudo ./scripts/verify-setup.sh

# 或手动检查
ls -la /var/lib/mingyue-agent
ls -la /var/log/mingyue-agent
ls -la /var/run/mingyue-agent

# 检查目录权限
stat /var/lib/mingyue-agent
stat /var/lib/mingyue-agent/share-backups
```

### 5. 检查 systemd 服务配置

```bash
# 查看服务配置
systemctl cat mingyue-agent.service

# 检查服务状态
systemctl status mingyue-agent.service -l
```

## 常见问题和解决方案

### 问题 1：配置文件中的路径错误

**症状**：服务启动失败，日志中没有详细错误

**解决方案**：
```bash
# 检查配置文件中的所有路径是否存在
grep -E '(file:|dir:|path:)' /etc/mingyue-agent/config.yaml

# 确保使用正确的配置（从 config.example.yaml 复制）
sudo cp config.example.yaml /etc/mingyue-agent/config.yaml
sudo chown root:root /etc/mingyue-agent/config.yaml
sudo chmod 644 /etc/mingyue-agent/config.yaml
```

### 问题 2：目录权限不正确

**症状**：Cannot create/write to directory

**解决方案**：
```bash
# 重新设置所有权限
sudo mkdir -p /var/lib/mingyue-agent/share-backups
sudo chown -R mingyue-agent:mingyue-agent /var/lib/mingyue-agent
sudo chown -R mingyue-agent:mingyue-agent /var/log/mingyue-agent
sudo chown -R mingyue-agent:mingyue-agent /var/run/mingyue-agent
sudo chmod -R 755 /var/lib/mingyue-agent
sudo chmod -R 755 /var/log/mingyue-agent
sudo chmod -R 755 /var/run/mingyue-agent
```

### 问题 3：端口已被占用

**症状**：Address already in use

**解决方案**：
```bash
# 检查端口占用
sudo netstat -tlnp | grep 8080
sudo netstat -tlnp | grep 9090

# 如果被占用，修改配置文件中的端口
sudo vi /etc/mingyue-agent/config.yaml
# 修改 server.http_port 和 server.grpc_port
```

### 问题 4：用户不存在或权限不足

**症状**：User not found or permission denied

**解决方案**：
```bash
# 检查用户是否存在
id mingyue-agent

# 如果不存在，创建用户
sudo useradd --system --no-create-home --shell /usr/sbin/nologin mingyue-agent

# 重新设置目录权限
sudo chown -R mingyue-agent:mingyue-agent /var/lib/mingyue-agent /var/log/mingyue-agent /var/run/mingyue-agent
```

### 问题 5：SELinux 阻止访问

**症状**：Permission denied despite correct ownership

**解决方案**：
```bash
# 检查 SELinux 状态
getenforce

# 如果是 Enforcing，设置正确的 SELinux 上下文
sudo semanage fcontext -a -t var_lib_t "/var/lib/mingyue-agent(/.*)?"
sudo restorecon -R /var/lib/mingyue-agent

# 或者临时禁用 SELinux 测试（不推荐生产环境）
sudo setenforce 0
```

## 完整的重新安装步骤

如果以上步骤都无法解决，尝试完全重新安装：

```bash
# 1. 停止并禁用服务
sudo systemctl stop mingyue-agent
sudo systemctl disable mingyue-agent

# 2. 备份配置和数据
sudo cp /etc/mingyue-agent/config.yaml /tmp/config.yaml.bak
sudo tar czf /tmp/mingyue-agent-data.tar.gz /var/lib/mingyue-agent

# 3. 清理旧安装
sudo rm -f /usr/local/bin/mingyue-agent
sudo rm -f /etc/systemd/system/mingyue-agent.service

# 4. 清理目录（注意：会删除数据）
sudo rm -rf /var/lib/mingyue-agent
sudo rm -rf /var/log/mingyue-agent
sudo rm -rf /var/run/mingyue-agent

# 5. 重新安装
cd /path/to/mingyue-agent
make build
sudo ./scripts/install.sh

# 6. 恢复配置（如果需要）
sudo cp /tmp/config.yaml.bak /etc/mingyue-agent/config.yaml

# 7. 启动服务
sudo systemctl start mingyue-agent

# 8. 查看日志
sudo journalctl -u mingyue-agent -f
```

## 获取帮助

如果问题仍未解决，请收集以下信息：

```bash
# 收集诊断信息
{
    echo "=== System Info ==="
    uname -a
    cat /etc/os-release
    
    echo -e "\n=== User Info ==="
    id mingyue-agent
    
    echo -e "\n=== Service Status ==="
    systemctl status mingyue-agent.service -l
    
    echo -e "\n=== Recent Logs ==="
    journalctl -u mingyue-agent -n 50 --no-pager
    
    echo -e "\n=== Directory Permissions ==="
    ls -la /var/lib/mingyue-agent
    ls -la /var/log/mingyue-agent
    ls -la /var/run/mingyue-agent
    
    echo -e "\n=== Config File ==="
    cat /etc/mingyue-agent/config.yaml
    
    echo -e "\n=== Port Usage ==="
    sudo netstat -tlnp | grep -E '(8080|9090)'
} > /tmp/mingyue-agent-diagnostic.txt

# 查看诊断信息
cat /tmp/mingyue-agent-diagnostic.txt
```

将输出的诊断信息提供给开发者或在 GitHub 上创建 issue。
