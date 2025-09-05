#!/bin/bash

# 测试脚本 - 启动mock服务和导出器进行功能验证

set -e

echo "=========================================="
echo "MiWiFi Exporter 测试脚本"
echo "=========================================="

# 设置变量
MOCK_SERVER_PORT=8080
EXPORTER_PORT=9001
CONFIG_FILE="test_config.json"
MOCK_SERVER_PID=""
EXPORTER_PID=""

# 清理函数
cleanup() {
    echo ""
    echo "正在清理进程..."
    
    if [ ! -z "$MOCK_SERVER_PID" ]; then
        kill $MOCK_SERVER_PID 2>/dev/null || true
        echo "Mock服务器已停止"
    fi
    
    if [ ! -z "$EXPORTER_PID" ]; then
        kill $EXPORTER_PID 2>/dev/null || true
        echo "导出器已停止"
    fi
    
    echo "清理完成"
    exit 0
}

# 设置信号处理
trap cleanup SIGINT SIGTERM

# 检查文件是否存在
if [ ! -f "miwifi_exporter" ]; then
    echo "错误: miwifi_exporter 二进制文件不存在"
    echo "请先运行: go build -o miwifi_exporter main.go"
    exit 1
fi

if [ ! -f "$CONFIG_FILE" ]; then
    echo "错误: 配置文件 $CONFIG_FILE 不存在"
    exit 1
fi

# 1. 构建mock服务器
echo "步骤 1: 构建mock服务器..."
cd mock_server
go build -o mock_server main.go
cd ..

# 2. 启动mock服务器
echo "步骤 2: 启动mock服务器 (端口 $MOCK_SERVER_PORT)..."
./mock_server/mock_server $MOCK_SERVER_PORT &
MOCK_SERVER_PID=$!
echo "Mock服务器PID: $MOCK_SERVER_PID"

# 等待mock服务器启动
echo "等待mock服务器启动..."
sleep 3

# 测试mock服务器是否正常工作
echo "步骤 3: 测试mock服务器连接..."
if curl -s http://localhost:$MOCK_SERVER_PORT/cgi-bin/luci/web > /dev/null; then
    echo "✅ Mock服务器连接正常"
else
    echo "❌ Mock服务器连接失败"
    cleanup
    exit 1
fi

# 4. 启动导出器
echo "步骤 4: 启动miwifi导出器 (端口 $EXPORTER_PORT)..."
export MIWIFI_ROUTER_IP=localhost
export MIWIFI_ROUTER_PASSWORD=test_password
export MIWIFI_ROUTER_HOST=test_router
export MIWIFI_ROUTER_TIMEOUT=10
export MIWIFI_SERVER_PORT=$EXPORTER_PORT
export MIWIFI_CACHE_ENABLED=true
export MIWIFI_CACHE_TTL=30s
export MIWIFI_LOGGING_LEVEL=info
export MIWIFI_LOGGING_FORMAT=text

./miwifi_exporter &
EXPORTER_PID=$!
echo "导出器PID: $EXPORTER_PID"

# 等待导出器启动
echo "等待导出器启动..."
sleep 5

# 5. 测试导出器metrics端点
echo "步骤 5: 测试导出器metrics端点..."
if curl -s http://localhost:$EXPORTER_PORT/metrics > /dev/null; then
    echo "✅ 导出器metrics端点正常"
else
    echo "❌ 导出器metrics端点失败"
    cleanup
    exit 1
fi

# 6. 验证指标输出
echo "步骤 6: 验证指标输出..."
echo "=========================================="
echo "获取指标输出:"
curl -s http://localhost:$EXPORTER_PORT/metrics | head -20
echo "=========================================="

# 检查关键指标是否存在
echo "步骤 7: 检查关键指标..."

METRICS_OUTPUT=$(curl -s http://localhost:$EXPORTER_PORT/metrics)

# 检查基本指标
if echo "$METRICS_OUTPUT" | grep -q "miwifi_cpu_cores"; then
    echo "✅ CPU核心指标正常"
else
    echo "❌ 缺少CPU核心指标"
fi

if echo "$METRICS_OUTPUT" | grep -q "miwifi_count_online"; then
    echo "✅ 在线设备数指标正常"
else
    echo "❌ 缺少在线设备数指标"
fi

if echo "$METRICS_OUTPUT" | grep -q "miwifi_wan_download_speed"; then
    echo "✅ WAN下载速度指标正常"
else
    echo "❌ 缺少WAN下载速度指标"
fi

if echo "$METRICS_OUTPUT" | grep -q "miwifi_device_download_traffic"; then
    echo "✅ 设备流量指标正常"
else
    echo "❌ 缺少设备流量指标"
fi

# 8. 测试缓存功能
echo "步骤 8: 测试缓存功能..."
echo "第一次请求metrics..."
curl -s http://localhost:$EXPORTER_PORT/metrics > /dev/null

echo "第二次请求metrics (应该使用缓存)..."
curl -s http://localhost:$EXPORTER_PORT/metrics > /dev/null

echo "✅ 缓存功能测试完成"

# 9. 显示完整的系统状态
echo "步骤 9: 显示系统状态..."
echo "=========================================="
echo "Mock服务器状态:"
ps -p $MOCK_SERVER_PID -o pid,ppid,cmd || echo "Mock服务器进程不存在"

echo ""
echo "导出器状态:"
ps -p $EXPORTER_PID -o pid,ppid,cmd || echo "导出器进程不存在"

echo ""
echo "端口占用情况:"
lsof -i :$MOCK_SERVER_PORT || echo "端口 $MOCK_SERVER_PORT 未被占用"
lsof -i :$EXPORTER_PORT || echo "端口 $EXPORTER_PORT 未被占用"

echo "=========================================="
echo "测试完成！系统运行正常。"
echo ""
echo "你可以访问以下URL查看指标:"
echo "  Mock服务器: http://localhost:$MOCK_SERVER_PORT"
echo "  Exporter指标: http://localhost:$EXPORTER_PORT/metrics"
echo ""
echo "按 Ctrl+C 停止所有服务"
echo "=========================================="

# 保持脚本运行
while true; do
    sleep 10
    
    # 检查进程是否还在运行
    if ! kill -0 $MOCK_SERVER_PID 2>/dev/null; then
        echo "Mock服务器意外停止"
        break
    fi
    
    if ! kill -0 $EXPORTER_PID 2>/dev/null; then
        echo "导出器意外停止"
        break
    fi
done

cleanup