#!/bin/bash
# http_response.toml → categraf-http-admin 批量导入脚本
# 在服务器上执行：bash batch_import.sh
# 前提：categraf-http-admin 已启动
#
# ⚠️ 本文件中的 URL / IP / job 名称均为脱敏示例（RFC5737 TEST-NET-1 保留段
#    + example.com）。导入前请替换成你环境里的真实地址，不要直接执行。

API="http://127.0.0.1:5000"

# ── 登录获取 session ──
echo "=== 登录 ==="
read -p "用户名 (默认 admin): " USERNAME
USERNAME=${USERNAME:-admin}
read -s -p "密码: " PASSWORD
echo ""

SESSION=$(curl -s -c - -X POST "$API/login" \
  -d "username=$USERNAME&password=$PASSWORD" 2>/dev/null | grep ccsid | awk '{print $NF}')

if [ -z "$SESSION" ]; then
  echo "❌ 登录失败，请检查用户名密码"
  exit 1
fi
echo "✅ 登录成功"
echo ""

add_one() {
  local payload="$1"
  local name="$2"
  result=$(curl -s -X POST "$API/api/targets" \
    -H "Content-Type: application/json" \
    -b "ccsid=$SESSION" \
    -d "$payload" 2>/dev/null)
  status=$(echo "$result" | python3 -c "import sys,json;d=json.load(sys.stdin);print('ok' if d.get('ok') else d.get('error',''))" 2>/dev/null)
  if [ "$status" = "ok" ]; then
    echo "  ✅ $name"
  else
    echo "  ❌ $name: $status"
  fi
}

echo "=== 第1组：38 个简单 GET 目标 ==="

add_one '{"url":"https://sts.example.com/prod-api/sts2-admin/info","job":"STS_2_系统-api接口"}' "STS_2_系统-api接口"
add_one '{"url":"http://192.0.2.20:51233/third/info","job":"示例第三方接口"}' "示例第三方接口"
add_one '{"url":"http://192.0.2.20:51258/info","job":"统一认证-api接口"}' "统一认证-api接口"
add_one '{"url":"https://bi.example.com/webroot/decision/login?origin=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx","job":"帆软系统"}' "帆软系统"
add_one '{"url":"http://192.0.2.201:3380/DMS_QUARTZ_FNSR/","job":"9-ssl-系统"}' "9-ssl-系统"
add_one '{"url":"https://career.example.com/login?redirect=%2Findex","job":"CAREER招聘系统"}' "CAREER招聘系统"
add_one '{"url":"http://192.0.2.20:51348/ping","job":"CIS-transformer系统"}' "CIS-transformer系统"
add_one '{"url":"http://192.0.2.214:8080/CIS","job":"CIS系统"}' "CIS系统"
add_one '{"url":"http://192.0.2.201:6080/CMS","job":"CMS系统"}' "CMS系统"
add_one '{"url":"http://192.0.2.161:8780/cos_rpt","job":"COS_RPT-windows主机"}' "COS_RPT-windows主机"
add_one '{"url":"http://192.0.2.161:5443/DMS_MSMQ/","job":"DMS_MSMQ-252主机"}' "DMS_MSMQ-252主机"
add_one '{"url":"http://192.0.2.201:3680/DMS_CRON_JOB/","job":"DMS-CRON-系统"}' "DMS-CRON-系统"
add_one '{"url":"http://dms.example.com/DMS/htmpage/common/welcome.htm","job":"DMS-系统"}' "DMS-系统"
add_one '{"url":"http://192.0.2.109:8080/SVC/","job":"ant-prd-192.0.2.109"}' "ant-prd-192.0.2.109"
add_one '{"url":"http://192.0.2.109:8090/SVC_INTF","job":"intf-tomcat-prd-192.0.2.109"}' "intf-tomcat-prd-192.0.2.109"
add_one '{"url":"http://jsonws.example.com:8880/JSONWS/","job":"JSONWS主机"}' "JSONWS主机"
add_one '{"url":"https://qms.example.com/QMS/login","job":"QMS系统"}' "QMS系统"
add_one '{"url":"https://scos.example.com/CAS/login?locale=zh_CN","job":"SCOS系统"}' "SCOS系统"
add_one '{"url":"https://sscd.example.com/login?redirect=%2Findex","job":"SSCD系统"}' "SSCD系统"
add_one '{"url":"https://sts.example.com/#/login","job":"STS_2 系统"}' "STS_2 系统"
add_one '{"url":"http://192.0.2.98:8780/nwms","job":"WMS-CRON"}' "WMS-CRON"
add_one '{"url":"http://wms.example.com/nwms","job":"WMS系统"}' "WMS系统"
add_one '{"url":"https://inspection.example.com/login?redirect=%2Findex","job":"查验管理 系统"}' "查验管理 系统"
add_one '{"url":"http://print.example.com/#/login","job":"打印管理系统"}' "打印管理系统"
add_one '{"url":"http://192.0.2.13:7780/cos_new","job":"老COS系统"}' "老COS系统"
add_one '{"url":"http://192.0.2.13:7080/COS","job":"新COS系统"}' "新COS系统"
add_one '{"url":"https://medical.example.com/home","job":"医疗官网"}' "医疗官网"
add_one '{"url":"http://192.0.2.20:51328/login?redirect=%2Findex","job":"医疗-后台系统"}' "医疗-后台系统"
add_one '{"url":"https://hap.example.com/prod-api/auth/detection","job":"docuai-auth"}' "docuai-auth"
add_one '{"url":"https://hap.example.com/prod-api/mail/detection","job":"docuai-mail"}' "docuai-mail"
add_one '{"url":"https://hap.example.com/prod-api/schedule/detection","job":"docuai-schedule"}' "docuai-schedule"
add_one '{"url":"https://hap.example.com/prod-api/system/detection","job":"docuai-system"}' "docuai-system"
add_one '{"url":"https://etms.example.com/frame/login","job":"TMS系统"}' "TMS系统"
add_one '{"url":"http://192.0.2.90:8081/#admin/repository/repositories","job":"maven-仓库"}' "maven-仓库"
add_one '{"url":"https://commonthird.example.com/prod-api/third/info","job":"commonthird.example.com后端"}' "commonthird.example.com后端"
add_one '{"url":"https://dms-quartz.example.com/","job":"dms-quartz.example.com 证书"}' "dms-quartz.example.com 证书"
add_one '{"url":"https://uploadapp.example.com/","job":"微信小程序upload"}' "微信小程序upload"
add_one '{"url":"https://wms-sit.example.com/nwms","job":"wms-sit测试环境"}' "wms-sit测试环境"

echo ""
echo "=== 第2组：外部 POST 目标（自定义配置示例） ==="

add_one '{"url":"https://gateway.example.com/api/example/external/direct/EXAMPLE","job":"外部接口-POST示例","method":"POST","expected_status_codes":"400","response_timeout":"15s","body":"{}","headers":["Content-Type","application/json"],"use_tls":true,"tls_ca":"/etc/categraf/ssl/example_ca.pem"}' "外部接口(POST)"

echo ""
echo "=== 全部导入完成 ==="
echo ""
echo "验证：curl -s http://127.0.0.1:5000/api/targets | python3 -m json.tool | grep -c '\"url\"'"
echo ""
echo "确认后停掉本地配置："
echo "  mv /etc/categraf/conf/input.http_response/http_response.toml \\"
echo "     /etc/categraf/conf/input.http_response/http_response.toml.bak"
echo "  systemctl restart categraf"
