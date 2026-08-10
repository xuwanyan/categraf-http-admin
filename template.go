package main

const pageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Categraf 拨测管理</title>
<link rel="icon" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'><text y='.9em' font-size='90'>📡</text></svg>">
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f5f5f5; color: #333; padding: 20px; }
.container { max-width: 1000px; margin: 0 auto; }
h1 { font-size: 22px; margin-bottom: 20px; }
h2 { font-size: 18px; margin: 24px 0 12px; }
.card { background: white; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); padding: 20px; margin-bottom: 20px; }
table { width: 100%; border-collapse: collapse; font-size: 14px; }
th, td { padding: 8px 10px; text-align: left; border-bottom: 1px solid #eee; }
th { background: #fafafa; font-weight: 600; color: #555; font-size: 13px; }
tr:hover { background: #f8f9ff; }
.badge { display: inline-block; background: #e8f0fe; color: #1967d2; border-radius: 4px; padding: 2px 8px; font-size: 12px; font-weight: 500; }
.btn { display: inline-block; padding: 6px 16px; border-radius: 6px; font-size: 14px; cursor: pointer; border: none; }
.btn-primary { background: #1a73e8; color: white; }
.btn-primary:hover { background: #1557b0; }
.btn-danger { background: #d93025; color: white; }
.btn-danger:hover { background: #b3261e; }
.btn-sm { padding: 4px 10px; font-size: 12px; }
.btn-outline { background: transparent; color: #1a73e8; border: 1px solid #dadce0; }
.btn-outline:hover { background: #f1f3f4; }
.form-row { display: flex; gap: 12px; flex-wrap: wrap; margin-bottom: 12px; }
.form-group { flex: 1; min-width: 150px; }
.form-group label { display: block; font-size: 13px; font-weight: 500; margin-bottom: 4px; color: #555; }
.form-group input, .form-group select, .form-group textarea { width: 100%; padding: 8px 10px; border: 1px solid #dadce0; border-radius: 6px; font-size: 14px; }
.form-group textarea { min-height: 60px; font-family: monospace; font-size: 13px; }
.form-group input:focus, .form-group select:focus { border-color: #1a73e8; outline: none; }
.form-group.checkbox { display: flex; align-items: center; min-width: auto; }
.form-group.checkbox label { margin: 0 0 0 6px; display: inline; cursor: pointer; }
.form-group.checkbox input[type=checkbox] { width: auto; cursor: pointer; }
.form-actions { margin-top: 12px; }
.empty { text-align: center; padding: 40px 20px; color: #888; }
.copy-area { background: #f8f9fa; border: 1px solid #eee; border-radius: 6px; padding: 12px; font-family: monospace; font-size: 12px; white-space: pre-wrap; max-height: 300px; overflow-y: auto; margin-top: 8px; }
.tag { display: inline-block; background: #e6f4ea; color: #1e8e3e; border-radius: 4px; padding: 2px 6px; font-size: 11px; margin: 1px; }
.alert { background: #fef7e0; border: 1px solid #f9d849; border-radius: 6px; padding: 12px 16px; margin-bottom: 16px; font-size: 14px; }
.alert code { background: #fff3cd; padding: 1px 4px; border-radius: 3px; }
.alert-error { background: #fce8e6; border-color: #d93025; color: #c5221f; }
.extra-fields { display: none; margin-top: 12px; padding-top: 12px; border-top: 1px dashed #eee; }
.show-extra { display: block; }
.tls-fields { display: none; margin-top: 12px; padding: 12px; background: #f8f9ff; border-radius: 6px; }
.show-tls { display: block; }
.edit-row { display: none; }
.edit-active { display: table-row; }
.edit-input { width: 100%; padding: 4px 6px; border: 1px solid #1a73e8; border-radius: 4px; font-size: 13px; }
.edit-select { padding: 4px 6px; border: 1px solid #1a73e8; border-radius: 4px; font-size: 13px; }
.edit-checkbox { width: 16px; height: 16px; cursor: pointer; }
.actions { white-space: nowrap; }
.actions form, .actions button { display: inline; }
</style>
</head>
<body>
<div class="container">
<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:20px">
	<h1 style="margin:0">📡 Categraf 拨测管理</h1>
	<a href="/logout" style="font-size:14px;color:#888;text-decoration:none">退出登录</a>
	</div>

{{if .Error}}
<div class="alert alert-error">{{.Error}}</div>
{{end}}

<div class="alert">
<strong>配置方式：</strong>
Categraf <code>conf/config.toml</code> →
<code>providers = ["local", "http"]</code> +
<code>[http_provider] remote_url = "http://你的IP:{{.Port}}/api/config/http_response"</code>（同时下发网站与端口拨测）
</div>


<div class="card">
<h2>添加目标</h2>
<form action="/api/targets" method="POST">
<div class="form-row">
<div class="form-group" style="max-width:150px">
<label>拨测类型</label>
<select name="kind" id="addKind" onchange="toggleKind()">
<option value="http">网站 HTTP(S)</option>
<option value="net">端口 TCP/UDP</option>
</select>
</div>
<div class="form-group" style="flex:3">
<label id="addUrlLabel">URL *</label>
<input type="url" name="url" id="addUrl" placeholder="https://example.com" required oninput="checkTLS('add')">
</div>
<div class="form-group">
<label>名称</label>
<input type="text" name="job" placeholder="myapp" list="jobList" required>
<datalist id="jobList">
{{range .Jobs}}<option value="{{.}}">{{end}}
</datalist>
</div>
</div>

<div id="addHttpFields">
<div class="form-row">
<div class="form-group">
<label>请求方法</label>
<select name="method" id="addMethod" onchange="toggleExtra('add')">
<option value="GET">GET</option>
<option value="POST">POST</option>
<option value="PUT">PUT</option>
<option value="DELETE">DELETE</option>
<option value="HEAD">HEAD</option>
</select>
</div>
<div class="form-group">
<label>期望状态码</label>
<input type="text" name="expected_status_codes" value="200" placeholder="200|301" required pattern="\d{3}(\|\d{3})*" title="三位数字，多值用|分隔，如 200 或 200|301">
</div>
			<div class="form-group">
			<label>超时时长</label>
			<select name="response_timeout">
			<option value="">默认</option>
			<option value="3s">3s</option>
			<option value="5s">5s</option>
			<option value="10s">10s</option>
			<option value="15s">15s</option>
			<option value="30s">30s</option>
			<option value="60s">60s</option>
			</select>
			</div>
</div>


<div class="extra-fields" id="addExtraFields">
<div class="form-row">
<div class="form-group">
<label>请求头 (JSON 数组)</label>
<input type="text" name="headers" placeholder='["X-Key","val"]'>
</div>
<div class="form-group" style="flex:2">
<label>请求 Body</label>
<textarea name="body" placeholder='{"key":"value"}'></textarea>
</div>
</div>
</div>

<div class="tls-fields" id="addTlsFields">
<div class="form-row">
<div class="form-group checkbox">
<input type="checkbox" name="insecure_skip_verify" id="addSkipVerify" value="true" onchange="toggleSkip('add')">
<label for="addSkipVerify" title="临时跳过证书校验（证书过期/自签名可用，不建议长期开启）">⚠️ 跳过证书校验</label>
</div>
</div>
<div style="margin-top:4px">
<a href="javascript:void(0)" id="addCaToggle" onclick="toggleCaAdvanced('add')" style="font-size:12px;color:#1a73e8;text-decoration:none">▸ 使用私有 CA 证书校验（高级，仅内网自建 CA 需要）</a>
</div>
<div class="form-row" id="addCaGroup" style="display:none;margin-top:8px">
<div class="form-group" style="flex:2">
<label>TLS CA 证书路径（categraf 服务器本地路径）</label>
<input type="text" name="tls_ca" id="addTlsCa" placeholder="/etc/categraf/ca.pem">
</div>
</div>
</div>
</div>

<div id="addNetFields" style="display:none">
<div class="form-row">
<div class="form-group" style="max-width:130px">
<label>协议</label>
<select name="protocol" id="addProtocol" disabled onchange="netProtoHint()">
<option value="tcp">TCP</option>
<option value="udp">UDP</option>
</select>
</div>
<div class="form-group" style="max-width:130px">
<label>连接超时</label>
<select name="response_timeout" disabled>
<option value="">默认(1s)</option>
<option value="3s">3s</option>
<option value="5s">5s</option>
<option value="10s">10s</option>
</select>
</div>
</div>
<div id="addUdpHint" style="display:none;font-size:12px;color:#b3261e;margin-bottom:8px">⚠️ UDP 无连接：不配置下方“发送与响应匹配”时判活不可靠，强烈建议配置</div>
<div style="margin-top:4px">
<a href="javascript:void(0)" id="addNetAdvToggle" onclick="toggleNetAdv()" style="font-size:12px;color:#1a73e8;text-decoration:none">▸ 发送与响应匹配（高级，验证服务真实可用，UDP 判活必需）</a>
</div>
<div class="form-row" id="addNetAdvGroup" style="display:none;margin-top:8px">
<div class="form-group">
<label>发送内容 (send，支持 \r \n \t 转义)</label>
<input type="text" name="send" disabled placeholder="\r\n">
</div>
<div class="form-group">
<label>期望响应包含 (expect)</label>
<input type="text" name="expect" disabled placeholder="PONG">
</div>
<div class="form-group" style="max-width:130px">
<label>读超时</label>
<select name="read_timeout" disabled>
<option value="">默认(3s)</option>
<option value="5s">5s</option>
<option value="10s">10s</option>
</select>
</div>
</div>
</div>

<div class="form-actions">
<button type="submit" class="btn btn-primary">➕ 添加</button>
</div>
</form>
</div>

<div class="card">
<h2>拨测目标 ({{.TargetCount}})</h2>
{{if .Targets}}
<table id="targetTable">
<thead>
<tr>
<th>类型</th>
<th style="width:26%">目标</th>
<th>方法/协议</th>
<th>名称</th>
<th>检查</th>
<th>TLS</th>
<th style="width:130px">操作</th>
</tr>
</thead>
<tbody>
{{range .Targets}}
<tr id="row-{{.ID}}">
<td>{{if eq .Kind "net"}}<span class="badge" style="background:#fef7e0;color:#b06000">端口</span>{{else}}<span class="badge">网站</span>{{end}}</td>
<td class="url-cell" style="word-break:break-all;font-family:monospace;font-size:13px">{{.URL}}</td>
<td><span class="badge">{{if eq .Kind "net"}}{{.Protocol}}{{else}}{{.Method}}{{end}}</span></td>
<td>{{.Job}}</td>
<td>{{if eq .Kind "net"}}{{if .Expect}}含“{{escCtl .Expect}}”{{else}}连通{{end}}{{else}}{{.ExpectedStatusCodes}}{{end}}</td>
<td>{{if eq .Kind "net"}}-{{else if .InsecureSkipVerify}}⚠️跳过校验{{else if .TLSCA}}🔒私有CA{{else}}-{{end}}</td>
<td class="actions">
<button class="btn btn-sm btn-outline" onclick="editRow('{{.ID}}')">编辑</button>
<form action="/api/targets/{{.ID}}/delete" method="POST" style="display:inline" onsubmit="return confirm('确定删除?')">
<button type="submit" class="btn btn-sm btn-danger">删除</button>
</form>
</td>
</tr>
{{if eq .Kind "net"}}
<tr id="edit-{{.ID}}" class="edit-row">
<td colspan="7">
<form class="edit-form" onsubmit="saveEdit('{{.ID}}');return false">
<div class="form-row">
<div class="form-group" style="flex:2">
<label>目标地址 (host:port)</label>
<input class="edit-input" name="url" value="{{.URL}}" required>
</div>
<div class="form-group">
<label>名称</label>
<input class="edit-input" name="job" value="{{.Job}}" required>
</div>
<div class="form-group">
<label>协议</label>
<select class="edit-select" name="protocol">
<option value="tcp" {{if eq .Protocol "tcp"}}selected{{end}}>TCP</option>
<option value="udp" {{if eq .Protocol "udp"}}selected{{end}}>UDP</option>
</select>
</div>
<div class="form-group">
<label>连接超时</label>
<select class="edit-select" name="response_timeout">
<option value="">默认(1s)</option>
<option value="3s" {{if eq .ResponseTimeout "3s"}}selected{{end}}>3s</option>
<option value="5s" {{if eq .ResponseTimeout "5s"}}selected{{end}}>5s</option>
<option value="10s" {{if eq .ResponseTimeout "10s"}}selected{{end}}>10s</option>
</select>
</div>
</div>
<div class="form-row">
<div class="form-group">
<label>发送内容 (send，支持 \r \n \t 转义)</label>
<input class="edit-input" name="send" value="{{escCtl .Send}}" placeholder="\r\n">
</div>
<div class="form-group">
<label>期望响应包含 (expect)</label>
<input class="edit-input" name="expect" value="{{escCtl .Expect}}" placeholder="PONG">
</div>
<div class="form-group" style="max-width:130px">
<label>读超时</label>
<select class="edit-select" name="read_timeout">
<option value="">默认(3s)</option>
<option value="5s" {{if eq .ReadTimeout "5s"}}selected{{end}}>5s</option>
<option value="10s" {{if eq .ReadTimeout "10s"}}selected{{end}}>10s</option>
</select>
</div>
</div>
<div class="form-actions">
<button type="submit" class="btn btn-sm btn-primary">保存</button>
<button type="button" class="btn btn-sm btn-outline" onclick="cancelEdit('{{.ID}}')">取消</button>
<span id="edit-status-{{.ID}}" style="font-size:13px;margin-left:10px"></span>
</div>
</form>
</td>
</tr>
{{else}}
<tr id="edit-{{.ID}}" class="edit-row">
<td colspan="7">
<form class="edit-form" onsubmit="saveEdit('{{.ID}}');return false">
<div class="form-row">
<div class="form-group" style="flex:3">
<label>URL</label>
<input class="edit-input" name="url" value="{{.URL}}" required oninput="checkTLSEdit('{{.ID}}')">
</div>
<div class="form-group">
<label>方法</label>
<select class="edit-select" name="method" onchange="toggleEditExtra('{{.ID}}')">
<option value="GET" {{if eq .Method "GET"}}selected{{end}}>GET</option>
<option value="POST" {{if eq .Method "POST"}}selected{{end}}>POST</option>
<option value="PUT" {{if eq .Method "PUT"}}selected{{end}}>PUT</option>
<option value="DELETE" {{if eq .Method "DELETE"}}selected{{end}}>DELETE</option>
<option value="HEAD" {{if eq .Method "HEAD"}}selected{{end}}>HEAD</option>
</select>
</div>
<div class="form-group">
<label>网站名称</label>
<input class="edit-input" name="job" value="{{.Job}}" required>
</div>
<div class="form-group">
<label>期望状态码</label>
<input class="edit-input" name="expected_status_codes" value="{{.ExpectedStatusCodes}}" required pattern="\d{3}(\|\d{3})*" title="三位数字，多值用|分隔，如 200 或 200|301">
t		</div>
			<div class="form-group">
			<label>超时时长</label>
			<select class="edit-select" name="response_timeout">
			<option value="">默认</option>
			<option value="3s" {{if eq .ResponseTimeout "3s"}}selected{{end}}>3s</option>
			<option value="5s" {{if eq .ResponseTimeout "5s"}}selected{{end}}>5s</option>
			<option value="10s" {{if eq .ResponseTimeout "10s"}}selected{{end}}>10s</option>
			<option value="15s" {{if eq .ResponseTimeout "15s"}}selected{{end}}>15s</option>
			<option value="30s" {{if eq .ResponseTimeout "30s"}}selected{{end}}>30s</option>
			<option value="60s" {{if eq .ResponseTimeout "60s"}}selected{{end}}>60s</option>
			</select>
			</div>
</div>
</div>
<div class="form-row" id="editExtra-{{.ID}}" style="{{if eq .Method "POST"}}display:flex{{else}}display:none{{end}};gap:12px">
<div class="form-group">
<label>请求头</label>
<input class="edit-input" name="headers" value='{{.HeadersJSON}}'>
</div>
<div class="form-group" style="flex:2">
<label>Body</label>
<textarea class="edit-input" name="body" style="min-height:40px">{{.Body}}</textarea>
</div>
</div>
<div class="tls-fields" id="editTls-{{.ID}}" style="{{if isHTTPS .URL}}display:block;margin-top:8px{{else}}display:none{{end}}">
<div class="form-row">
<div class="form-group checkbox">
<input type="checkbox" class="edit-checkbox" name="insecure_skip_verify" id="edit-skipv-{{.ID}}" value="true" {{if .InsecureSkipVerify}}checked{{end}} onchange="toggleSkipEdit('{{.ID}}')">
<label for="edit-skipv-{{.ID}}" title="临时跳过证书校验（证书过期/自签名可用，不建议长期开启）">⚠️ 跳过证书校验</label>
</div>
</div>
<div style="margin-top:4px">
<a href="javascript:void(0)" id="edit-catoggle-{{.ID}}" onclick="toggleCaAdvancedEdit('{{.ID}}')" style="font-size:12px;color:#1a73e8;text-decoration:none">▸ 使用私有 CA 证书校验（高级）</a>
</div>
<div class="form-row" id="edit-cagroup-{{.ID}}" style="{{if .TLSCA}}display:flex{{else}}display:none{{end}};margin-top:8px">
<div class="form-group" style="flex:2">
<label>TLS CA 证书路径（categraf 服务器本地路径）</label>
<input class="edit-input" name="tls_ca" id="edit-tlsca-{{.ID}}" value="{{.TLSCA}}" placeholder="/etc/categraf/ca.pem">
</div>
</div>
</div>
<div class="form-actions">
<button type="submit" class="btn btn-sm btn-primary">保存</button>
<button type="button" class="btn btn-sm btn-outline" onclick="cancelEdit('{{.ID}}')">取消</button>
<span id="edit-status-{{.ID}}" style="font-size:13px;margin-left:10px"></span>
</div>
</form>
</td>
</tr>
{{end}}
{{end}}
</tbody>
</table>
{{else}}
<div class="empty">暂无拨测目标，在上方添加</div>
{{end}}
</div>

<div class="card">
<h2>当前生成的 TOML 配置</h2>
<div style="font-size:13px;font-weight:600;margin-top:4px">http_response（网站拨测）</div>
<div class="copy-area" id="tomlHttp">{{.TOML}}</div>
<div style="font-size:13px;font-weight:600;margin-top:12px">net_response（端口拨测）</div>
<div class="copy-area" id="tomlNet">{{.NetTOML}}</div>
<div style="margin-top:8px;font-size:12px;color:#888">
version: {{.Version}}
<button class="btn btn-sm" style="background:#f1f3f4;margin-left:10px" onclick="copyToml('tomlHttp')">📋 复制 http</button>
<button class="btn btn-sm" style="background:#f1f3f4;margin-left:6px" onclick="copyToml('tomlNet')">📋 复制 net</button>
</div>
</div>

</div>

<script>
function isHTTPS(url) {
  return url.trim().toLowerCase().startsWith('https://');
}

// 添加表单：网站/端口两套字段切换（隐藏侧统一 disabled，避免提交和 required 校验）
function toggleKind() {
  var isNet = document.getElementById('addKind').value === 'net';
  var http = document.getElementById('addHttpFields');
  var net = document.getElementById('addNetFields');
  http.style.display = isNet ? 'none' : 'block';
  net.style.display = isNet ? 'block' : 'none';
  http.querySelectorAll('input,select,textarea').forEach(function(el) { el.disabled = isNet; });
  net.querySelectorAll('input,select,textarea').forEach(function(el) { el.disabled = !isNet; });
  var url = document.getElementById('addUrl');
  if (isNet) {
    url.type = 'text';
    url.placeholder = '10.0.0.1:22（host:port）';
    document.getElementById('addUrlLabel').textContent = '目标地址 *';
    netProtoHint();
  } else {
    url.type = 'url';
    url.placeholder = 'https://example.com';
    document.getElementById('addUrlLabel').textContent = 'URL *';
    checkTLS('add');
  }
}

function netProtoHint() {
  var isUdp = document.getElementById('addProtocol').value === 'udp';
  document.getElementById('addUdpHint').style.display = isUdp ? 'block' : 'none';
}

function toggleNetAdv() {
  var g = document.getElementById('addNetAdvGroup');
  var show = g.style.display === 'none';
  g.style.display = show ? 'flex' : 'none';
  document.getElementById('addNetAdvToggle').innerHTML =
    (show ? '▾' : '▸') + ' 发送与响应匹配（高级，验证服务真实可用，UDP 判活必需）';
}

function checkTLS(prefix) {
  if (prefix === 'add' && document.getElementById('addKind').value === 'net') return;
  var url = document.getElementById(prefix + 'Url').value;
  var el = document.getElementById(prefix + 'TlsFields');
  if (isHTTPS(url)) {
    el.style.display = 'block';
  } else {
    el.style.display = 'none';
    document.getElementById(prefix + 'SkipVerify').checked = false;
    document.getElementById(prefix + 'TlsCa').value = '';
    document.getElementById(prefix + 'CaGroup').style.display = 'none';
    document.getElementById(prefix + 'CaToggle').innerHTML = '▸ 使用私有 CA 证书校验（高级，仅内网自建 CA 需要）';
  }
}

function toggleExtra(prefix) {
  var method = document.getElementById(prefix + 'Method').value;
  var el = document.getElementById(prefix + 'ExtraFields');
  el.className = method === 'POST' ? 'extra-fields show-extra' : 'extra-fields';
}

function toggleEditExtra(id) {
  var method = document.querySelector('#edit-' + id + ' select[name="method"]').value;
  var el = document.getElementById('editExtra-' + id);
  el.style.display = method === 'POST' ? 'flex' : 'none';
}

// 跳过校验与 CA 互斥：勾跳过时置灰并清空 CA
function toggleSkip(prefix) {
  var skip = document.getElementById(prefix + 'SkipVerify').checked;
  var ca = document.getElementById(prefix + 'TlsCa');
  var toggle = document.getElementById(prefix + 'CaToggle');
  if (skip) {
    ca.value = '';
    ca.disabled = true;
    document.getElementById(prefix + 'CaGroup').style.display = 'none';
    toggle.style.pointerEvents = 'none';
    toggle.style.color = '#bbb';
  } else {
    ca.disabled = false;
    toggle.style.pointerEvents = 'auto';
    toggle.style.color = '#1a73e8';
  }
}

function toggleCaAdvanced(prefix) {
  var g = document.getElementById(prefix + 'CaGroup');
  var show = g.style.display === 'none';
  g.style.display = show ? 'flex' : 'none';
  document.getElementById(prefix + 'CaToggle').innerHTML =
    (show ? '▾' : '▸') + ' 使用私有 CA 证书校验（高级，仅内网自建 CA 需要）';
}

function checkTLSEdit(id) {
  var url = document.querySelector('#edit-' + id + ' input[name="url"]').value;
  var el = document.getElementById('editTls-' + id);
  el.style.display = isHTTPS(url) ? 'block' : 'none';
  if (!isHTTPS(url)) {
    document.getElementById('edit-skipv-' + id).checked = false;
    document.getElementById('edit-tlsca-' + id).value = '';
    document.getElementById('edit-cagroup-' + id).style.display = 'none';
  }
}

function toggleSkipEdit(id) {
  var skip = document.getElementById('edit-skipv-' + id).checked;
  var ca = document.getElementById('edit-tlsca-' + id);
  var toggle = document.getElementById('edit-catoggle-' + id);
  if (skip) {
    ca.value = '';
    ca.disabled = true;
    document.getElementById('edit-cagroup-' + id).style.display = 'none';
    toggle.style.pointerEvents = 'none';
    toggle.style.color = '#bbb';
  } else {
    ca.disabled = false;
    toggle.style.pointerEvents = 'auto';
    toggle.style.color = '#1a73e8';
  }
}

function toggleCaAdvancedEdit(id) {
  var g = document.getElementById('edit-cagroup-' + id);
  var show = g.style.display === 'none';
  g.style.display = show ? 'flex' : 'none';
  document.getElementById('edit-catoggle-' + id).innerHTML =
    (show ? '▾' : '▸') + ' 使用私有 CA 证书校验（高级）';
}

function editRow(id) {
  document.getElementById('row-' + id).style.display = 'none';
  document.getElementById('edit-' + id).className = 'edit-active';
  // 网站拨测行才有 TLS 区块；端口拨测行无需处理
  if (document.getElementById('editTls-' + id)) {
    checkTLSEdit(id);
    toggleSkipEdit(id);
  }
}

function copyToml(areaId) {
  var text = document.getElementById(areaId).textContent;
  if (navigator.clipboard && window.isSecureContext) {
    navigator.clipboard.writeText(text).then(function() { alert('已复制'); });
    return;
  }
  // 非 HTTPS/localhost 环境下 clipboard API 不可用，降级用 execCommand
  var ta = document.createElement('textarea');
  ta.value = text;
  ta.style.position = 'fixed';
  ta.style.opacity = '0';
  document.body.appendChild(ta);
  ta.select();
  try {
    document.execCommand('copy');
    alert('已复制');
  } catch (e) {
    alert('复制失败，请手动选择文本复制');
  }
  document.body.removeChild(ta);
}

function cancelEdit(id) {
  document.getElementById('row-' + id).style.display = 'table-row';
  document.getElementById('edit-' + id).className = 'edit-row';
}

function saveEdit(id) {
  var form = document.querySelector('#edit-' + id + ' form');
  var data = new FormData(form);
  var status = document.getElementById('edit-status-' + id);
  status.textContent = '保存中...';
  status.style.color = '#888';

  fetch('/api/targets/' + id + '/edit', {
    method: 'POST',
    body: new URLSearchParams(data)
  }).then(function(r) {
    if (r.ok) {
      status.textContent = '✅ 已保存';
      status.style.color = '#1e8e3e';
      setTimeout(function() { window.location.reload(); }, 600);
    } else {
      return r.text().then(function(t) {
        status.textContent = '❌ ' + t;
        status.style.color = '#d93025';
      });
    }
  }).catch(function(e) {
    status.textContent = '❌ 网络错误';
    status.style.color = '#d93025';
  });
}
</script>
</body>
</html>`
