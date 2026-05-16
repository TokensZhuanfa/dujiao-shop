#!/usr/bin/env bash
# dujiao-shop release tarball 安装脚本
# 用法: 解压 tarball → 进入解压目录 → sudo ./install.sh
#
# 行为:
#   1. 创建 /opt/dujiao + 系统用户 dujiao
#   2. 把二进制 / config 模板 / systemd unit 复制到位
#   3. 如果是 headless 包,提示用户手动配 nginx (nginx/ 目录有模板)
#   4. systemctl enable --now dujiao
#
# 幂等: 重跑会更新二进制,但保留 config.yml 与数据库.

set -euo pipefail

INSTALL_DIR=/opt/dujiao
SERVICE_USER=dujiao
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 必须 root
if [[ $EUID -ne 0 ]]; then
    echo "[!] 需要 root 权限运行: sudo $0" >&2
    exit 1
fi

# 检测是 fullstack 还是 headless (看有没有 nginx/ 目录)
if [[ -d "$SCRIPT_DIR/nginx" ]]; then
    EDITION=headless
    BINARY=dujiao-api-headless
else
    EDITION=fullstack
    BINARY=dujiao-api
fi
echo "==> 检测到 ${EDITION} 版"

# 1) 用户 + 目录
if ! id -u "$SERVICE_USER" &>/dev/null; then
    echo "==> 创建系统用户 $SERVICE_USER"
    useradd --system --no-create-home --shell /usr/sbin/nologin "$SERVICE_USER"
fi

echo "==> 创建目录 $INSTALL_DIR/{db,uploads,logs,credentials}"
mkdir -p "$INSTALL_DIR"/{db,uploads,logs,credentials}

# 2) 二进制
echo "==> 安装二进制到 $INSTALL_DIR/"
install -m 0755 "$SCRIPT_DIR/$BINARY" "$INSTALL_DIR/dujiao-api"
install -m 0755 "$SCRIPT_DIR/admin-tool" "$INSTALL_DIR/admin-tool"

# 3) 配置模板 (仅当用户还没建 config.yml 时复制)
if [[ ! -f "$INSTALL_DIR/config.yml" ]]; then
    echo "==> 安装 config.yml 模板 (你必须改! 详见注释)"
    install -m 0640 "$SCRIPT_DIR/config.template.yml" "$INSTALL_DIR/config.yml"
    NEEDS_CONFIG=1
else
    echo "==> 已存在 config.yml,保留原值"
    NEEDS_CONFIG=0
fi

# 4) nginx 模板 (仅 headless)
if [[ $EDITION == headless ]]; then
    NGINX_DST=$INSTALL_DIR/nginx
    mkdir -p "$NGINX_DST"
    install -m 0644 "$SCRIPT_DIR/nginx/admin.conf" "$NGINX_DST/admin.conf"
    install -m 0644 "$SCRIPT_DIR/nginx/user.conf"  "$NGINX_DST/user.conf"
    echo "==> nginx 模板放在 $NGINX_DST/,需要你自己配上前端 dist 路径并加载到 nginx"
fi

# 5) systemd
echo "==> 安装 systemd unit"
install -m 0644 "$SCRIPT_DIR/systemd/dujiao.service" /etc/systemd/system/dujiao.service
chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR"
systemctl daemon-reload

# 6) 提示 / 启动
echo
if [[ $NEEDS_CONFIG -eq 1 ]]; then
    cat <<EOF
============================================================
首次安装完毕,但 **你必须先编辑 config.yml** 再启动:
    sudo vi $INSTALL_DIR/config.yml

至少检查:
  - jwt_secret      (生成: openssl rand -base64 32)
  - server.port     (默认 8080)
  - redis.address   (默认 127.0.0.1:6379,需要本机装 redis)
  - payment.*       (你接入的支付通道)

编辑完执行:
    sudo systemctl enable --now dujiao
    sudo journalctl -u dujiao -f

后台默认用户名 admin,首次登录请立刻改密码 + 启用 2FA。
忘了密码可以:
    sudo -u dujiao $INSTALL_DIR/admin-tool list-admins
============================================================
EOF
else
    echo "==> 二进制已升级,重启服务:"
    echo "    sudo systemctl restart dujiao"
fi
