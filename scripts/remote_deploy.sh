#!/usr/bin/env bash
# 生产部署脚本 —— 由 CI 通过 SSH 在服务器上以 root 执行。
# 本文件不含任何密钥、服务器地址或域名，可安全入库。
#
# 用法: remote_deploy.sh [部署包路径]
# 部署包内容: server(Linux 二进制) config/{aimap.json,systemPrompt.txt} web.tgz
set -euo pipefail

APP_DIR=/opt/order_web
STAGING_DIR=/opt/order_web_staging
WEB_ROOT=/www/wwwroot/order_web
SERVICE=order-web.service
BACKEND_PORT=8090
HEALTH_URL="http://127.0.0.1:${BACKEND_PORT}/healthz"
HEALTH_RETRIES=15
HEALTH_INTERVAL=2

BUNDLE="${1:-${STAGING_DIR}/bundle.tgz}"
WORK="${STAGING_DIR}/incoming"

log() { printf '[%s] %s\n' "$(date '+%F %T')" "$*"; }
die() { log "ERROR: $*"; exit 1; }

[ -f "$BUNDLE" ] || die "找不到部署包 ${BUNDLE}"

log "解包 ${BUNDLE}"
rm -rf "$WORK"
mkdir -p "$WORK"
tar -xzf "$BUNDLE" -C "$WORK"
[ -f "${WORK}/server" ]  || die "部署包缺少 server 二进制"
[ -f "${WORK}/web.tgz" ] || die "部署包缺少 web.tgz"

# ---------- 后端 ----------
# 保留上一版二进制用于回滚。用 mv(rename) 而非 cp 覆盖：正在运行的进程持有旧
# inode，rename 只替换目录项，既原子又不会触发 ETXTBSY。
if [ -f "${APP_DIR}/server" ]; then
  cp -a "${APP_DIR}/server" "${APP_DIR}/server.prev"
  log "已备份上一版二进制到 server.prev"
fi

install -m 750 -o root -g orderweb "${WORK}/server" "${APP_DIR}/server.new"
mv -f "${APP_DIR}/server.new" "${APP_DIR}/server"

# 只同步静态配置。settings.yaml 含密钥、productbelong.json 会被
# PUT /api/checkout/product-belong 在运行期回写，二者均不得被部署覆盖。
# systemPromptWuyin.txt 为吴饮品牌提示词，服务器上不存在时服务只告警不阻断，
# 但吴饮识别不可用，故随部署同步。
for f in aimap.json systemPrompt.txt systemPromptWuyin.txt; do
  if [ -f "${WORK}/config/${f}" ]; then
    install -m 644 -o root -g orderweb "${WORK}/config/${f}" "${APP_DIR}/go/config/${f}"
    log "同步 go/config/${f}"
  fi
done

for d in uploads logs; do
  if [ ! -d "${APP_DIR}/${d}" ]; then
    mkdir -p "${APP_DIR}/${d}"
    chown root:orderweb "${APP_DIR}/${d}"
    chmod 770 "${APP_DIR}/${d}"
    log "创建 ${APP_DIR}/${d}"
  fi
done

log "重启 ${SERVICE}"
systemctl restart "$SERVICE"

log "健康检查 ${HEALTH_URL}（最多 $((HEALTH_RETRIES * HEALTH_INTERVAL))s）"
healthy=0
for _ in $(seq 1 "$HEALTH_RETRIES"); do
  if curl -fsS -m 5 "$HEALTH_URL" >/dev/null 2>&1; then
    healthy=1
    break
  fi
  sleep "$HEALTH_INTERVAL"
done

if [ "$healthy" -ne 1 ]; then
  log "健康检查未通过，输出最近日志后回滚"
  journalctl -u "$SERVICE" --no-pager -n 40 || true
  if [ -f "${APP_DIR}/server.prev" ]; then
    mv -f "${APP_DIR}/server.prev" "${APP_DIR}/server"
    systemctl restart "$SERVICE"
    sleep 3
    if curl -fsS -m 5 "$HEALTH_URL" >/dev/null 2>&1; then
      die "已回滚至上一版本且服务恢复正常，本次发布判定失败"
    fi
  fi
  die "回滚后服务仍不可用，需人工介入"
fi
log "后端健康检查通过"

# ---------- 前端 ----------
# 先解到临时目录再整体换名，避免解包过程中 nginx 读到半成品文件。
WEB_NEW="${STAGING_DIR}/web_new"
rm -rf "$WEB_NEW"
mkdir -p "$WEB_NEW"
tar -xzf "${WORK}/web.tgz" -C "$WEB_NEW"
[ -f "${WEB_NEW}/index.html" ] || die "前端产物缺少 index.html，已中止（后端保持新版本）"

rm -rf "${WEB_ROOT}.prev"
if [ -d "$WEB_ROOT" ]; then
  mv "$WEB_ROOT" "${WEB_ROOT}.prev"
fi
mv "$WEB_NEW" "$WEB_ROOT"
chown -R root:root "$WEB_ROOT"
find "$WEB_ROOT" -type d -exec chmod 755 {} \;
find "$WEB_ROOT" -type f -exec chmod 644 {} \;
log "前端已发布到 ${WEB_ROOT}（上一版保留在 ${WEB_ROOT}.prev）"

rm -rf "$WORK" "$BUNDLE"
log "部署完成"
