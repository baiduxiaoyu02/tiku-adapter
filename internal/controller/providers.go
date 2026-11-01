package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/itihey/tikuAdapter/pkg/logger"
	"github.com/spf13/viper"
)

// Provider 单个模型供应商配置
type Provider struct {
	Name        string `yaml:"name" json:"name"`
	BaseURL     string `yaml:"base_url" json:"base_url"`
	APIKey      string `yaml:"api_key" json:"api_key"`
	Model       string `yaml:"model" json:"model"`
	VisionModel string `yaml:"vision_model" json:"vision_model"`
}

// providerResponse 列表响应（密钥掩码）
type providerResponse struct {
	Active    string     `json:"active"`
	Providers []Provider `json:"providers"`
}

const providersPageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>模型供应商管理</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,"Segoe UI","Microsoft YaHei",sans-serif;background:#f0f2f5;min-height:100vh;padding:32px 16px;color:#1f2937}
.wrap{max-width:760px;margin:0 auto}
h1{font-size:22px;margin-bottom:4px}
.sub{color:#6b7280;font-size:13px;margin-bottom:20px}
.card{background:#fff;border-radius:12px;padding:20px;box-shadow:0 1px 4px rgba(0,0,0,.08);margin-bottom:16px}
.card h2{font-size:15px;margin-bottom:12px;color:#4f46e5}
label{display:block;font-size:13px;color:#555;margin:10px 0 4px}
input,select{width:100%;padding:9px 10px;border:1px solid #d1d5db;border-radius:8px;font-size:14px;outline:none}
input:focus{border-color:#4f46e5}
.row{display:flex;gap:10px}
.row>div{flex:1}
button{padding:9px 16px;border:none;border-radius:8px;font-size:14px;cursor:pointer;font-weight:600}
.btn-primary{background:#4f46e5;color:#fff}
.btn-ghost{background:#eef2ff;color:#4f46e5}
.btn-danger{background:#fef2f2;color:#dc2626}
.btn-sm{padding:6px 10px;font-size:12px}
.list-item{display:flex;justify-content:space-between;align-items:center;padding:10px 12px;border:1px solid #e5e7eb;border-radius:8px;margin-bottom:8px}
.list-item .name{font-weight:600}
.list-item .meta{font-size:12px;color:#6b7280}
.list-item.active{border-color:#4f46e5;background:#eef2ff}
.tag{font-size:11px;background:#4f46e5;color:#fff;border-radius:10px;padding:2px 8px;margin-left:6px}
.msg{padding:8px 10px;border-radius:6px;font-size:13px;margin:8px 0;display:none}
.msg.ok{background:#dcfce7;color:#16a34a;display:block}
.msg.err{background:#fef2f2;color:#dc2626;display:block}
.actions{display:flex;gap:6px;flex-wrap:wrap}
</style>
</head>
<body>
<div class="wrap">
<h1>模型供应商管理</h1>
<p class="sub">配置保存后需重启服务生效：托盘菜单 → Restart Service，或本页「保存并重启」。</p>

<div class="card">
<h2>当前使用</h2>
<div id="current" class="meta">加载中...</div>
</div>

<div class="card">
<h2>供应商列表</h2>
<div id="list">加载中...</div>
<div class="actions" style="margin-top:10px">
<button class="btn-ghost btn-sm" onclick="loadAll()">刷新</button>
</div>
</div>

<div class="card">
<h2 id="form-title">添加供应商</h2>
<div class="msg" id="msg"></div>
<label>名称</label><input id="f-name" placeholder="如：主力 / 备用">
<label>接口地址（base_url）</label><input id="f-url" placeholder="如 https://api.deepseek.com/v1">
<label>API Key</label><input id="f-key" placeholder="sk-...">
<div class="row">
<div><label>普通模型</label><input id="f-model" placeholder="如 glm-5.2"></div>
<div><label>图片视觉模型</label><input id="f-vision" placeholder="如 sensenova-6.8-flash-lite"></div>
</div>
<div style="display:flex;gap:8px;margin-top:14px">
<button class="btn-primary" style="flex:1" onclick="save()">保存并设为当前</button>
<button class="btn-ghost" onclick="testConn()">测试连通</button>
<button class="btn-ghost" onclick="resetForm()">清空表单</button>
</div>
</div>
</div>

<script>
const API = '/adapter-service/providers';
const $ = id => document.getElementById(id);
function mask(k){ if(!k) return '(未填)'; return k.length>12 ? k.slice(0,8)+'****'+k.slice(-4) : k; }
function show(m, ok){ const el=$('msg'); el.textContent=m; el.className='msg '+(ok?'ok':'err'); }

async function loadAll(){
 try{
  const r = await fetch(API);
  const d = await r.json();
  renderCurrent(d.current||{});
  renderList(d.providers||[], d.active||'');
 }catch(e){ $('current').textContent='加载失败：'+e.message; }
}
function renderCurrent(c){
 $('current').innerHTML = '接口：'+(c.base_url||'(未配置)')+'<br>模型：'+(c.model||'(未配置)')+'<br>密钥：'+mask(c.api_key);
}
function renderList(providers, active){
 const el=$('list'); el.innerHTML='';
 if(!providers.length){ el.innerHTML='<div class="meta">暂无供应商，请在下方添加</div>'; return; }
 providers.forEach(p=>{
  const div=document.createElement('div');
  div.className='list-item'+(p.name===active?' active':'');
  div.innerHTML='<div><span class="name">'+p.name+(p.name===active?'<span class="tag">当前</span>':'')+'</span>'
   +'<div class="meta">'+(p.base_url||'')+' · '+(p.model||'')+'</div></div>'
   +'<div class="actions">'
   +'<button class="btn-ghost btn-sm" onclick="useProvider(\''+p.name+'\')">设为当前</button>'
   +'<button class="btn-danger btn-sm" onclick="delProvider(\''+p.name+'\')">删除</button>'
   +'</div>';
  el.appendChild(div);
 });
}
async function apiPost(path, body){
 const r = await fetch(API+path, {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(body||{})});
 const d = await r.json();
 if(!r.ok) throw new Error(d.message || ('HTTP '+r.status));
 return d;
}
async function useProvider(name){
 try{ await apiPost('/switch',{name:name}); show('已切换：'+name,true); loadAll(); }
 catch(e){ show('切换失败：'+e.message,false); }
}
async function delProvider(name){
 if(!confirm('删除供应商「'+name+'」？')) return;
 try{ await apiPost('/delete',{name:name}); show('已删除',true); loadAll(); }
 catch(e){ show('删除失败：'+e.message,false); }
}
function resetForm(){ ['f-name','f-url','f-key','f-model','f-vision'].forEach(id=>$(id).value=''); $('form-title').textContent='添加供应商'; }
function formProvider(){
 return { name:$('f-name').value.trim(), base_url:$('f-url').value.trim(), api_key:$('f-key').value.trim(), model:$('f-model').value.trim(), vision_model:$('f-vision').value.trim() };
}
async function save(){
 const p = formProvider();
 if(!p.name || !p.base_url || !p.api_key){ show('名称/接口地址/密钥必填',false); return; }
 try{ await apiPost('/save', p); show('已保存并设为当前 ✓（需重启服务生效）',true); resetForm(); loadAll(); }
 catch(e){ show('保存失败：'+e.message,false); }
}
async function testConn(){
 const p = formProvider();
 if(!p.base_url || !p.api_key){ show('请先填写接口地址与密钥',false); return; }
 show('测试中...',true);
 try{ const d = await apiPost('/test', p); show(d.ok ? '✓ 连通成功（模型 '+d.model+'）' : '✗ '+d.message, d.ok); }
 catch(e){ show('测试失败：'+e.message,false); }
}
loadAll();
</script>
</body>
</html>`

// ProvidersPage 模型供应商管理页
func ProvidersPage(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(providersPageHTML))
}

// maskKey 掩码：保留前 8 后 4
func maskKey(k string) string {
	if k == "" {
		return ""
	}
	if len(k) <= 12 {
		return "****"
	}
	return k[:8] + "****" + k[len(k)-4:]
}

// getProviders 读 providers 列表（viper → 掩码响应）
func getProviders() (active Provider, list []Provider) {
	active = Provider{
		Name:        "当前使用",
		BaseURL:     viper.GetString("ai.base_url"),
		APIKey:      viper.GetString("ai.api_key"),
		Model:       viper.GetString("ai.model"),
		VisionModel: viper.GetString("ai.vision_model"),
	}
	var raw []Provider
	if err := viper.UnmarshalKey("providers", &raw); err != nil {
		logger.SysError("读取 providers 配置失败：" + err.Error())
		return active, raw
	}
	return active, raw
}

// ProvidersList GET /adapter-service/providers —— 当前配置 + 供应商列表（密钥掩码）
func ProvidersList(c *gin.Context) {
	active, list := getProviders()
	active.APIKey = maskKey(active.APIKey)
	for i := range list {
		list[i].APIKey = maskKey(list[i].APIKey)
	}
	c.JSON(http.StatusOK, gin.H{"active": active, "providers": list, "active_name": viper.GetString("ai.active")})
}

// ProvidersSave POST /adapter-service/providers/save —— 新增/更新供应商并设为当前，持久化 config.yaml
func ProvidersSave(c *gin.Context) {
	var in Provider
	if err := c.ShouldBindJSON(&in); err != nil || in.Name == "" || in.BaseURL == "" || in.APIKey == "" {
		c.JSON(400, gin.H{"message": "名称/接口地址/密钥必填"})
		return
	}
	var list []Provider
	_ = viper.UnmarshalKey("providers", &list)
	replaced := false
	for i := range list {
		if list[i].Name == in.Name {
			list[i] = in
			replaced = true
			break
		}
	}
	if !replaced {
		list = append(list, in)
	}
	viper.Set("providers", list)

	// 设为当前：写 ai.* 顶层字段（搜索流程读取的就是这一份）
	viper.Set("ai.enable", true)
	viper.Set("ai.base_url", in.BaseURL)
	viper.Set("ai.api_key", in.APIKey)
	viper.Set("ai.model", in.Model)
	viper.Set("ai.vision_model", in.VisionModel)
	viper.Set("ai.active", in.Name)

	if err := viper.WriteConfig(); err != nil {
		logger.SysError("写 config.yaml 失败：" + err.Error())
		c.JSON(500, gin.H{"message": "配置写入失败：" + err.Error()})
		return
	}
	logger.SysLog(fmt.Sprintf("供应商配置已保存并设为当前：%s（%s）", in.Name, in.Model))
	c.JSON(http.StatusOK, gin.H{"message": "已保存并设为当前，重启服务后生效"})
}

// ProvidersSwitch POST /adapter-service/providers/switch —— 切换当前供应商
func ProvidersSwitch(c *gin.Context) {
	var in struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Name == "" {
		c.JSON(400, gin.H{"message": "参数错误"})
		return
	}
	var list []Provider
	_ = viper.UnmarshalKey("providers", &list)
	for _, p := range list {
		if p.Name == in.Name {
			viper.Set("ai.base_url", p.BaseURL)
			viper.Set("ai.api_key", p.APIKey)
			viper.Set("ai.model", p.Model)
			viper.Set("ai.vision_model", p.VisionModel)
			viper.Set("ai.active", p.Name)
			if err := viper.WriteConfig(); err != nil {
				c.JSON(500, gin.H{"message": "配置写入失败：" + err.Error()})
				return
			}
			logger.SysLog("当前供应商已切换：" + p.Name)
			c.JSON(http.StatusOK, gin.H{"message": "已切换到 " + p.Name + "，重启服务后生效"})
			return
		}
	}
	c.JSON(404, gin.H{"message": "供应商不存在"})
}

// ProvidersDelete POST /adapter-service/providers/delete —— 删除供应商
func ProvidersDelete(c *gin.Context) {
	var in struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Name == "" {
		c.JSON(400, gin.H{"message": "参数错误"})
		return
	}
	var list []Provider
	_ = viper.UnmarshalKey("providers", &list)
	out := list[:0]
	for _, p := range list {
		if p.Name != in.Name {
			out = append(out, p)
		}
	}
	viper.Set("providers", out)
	if err := viper.WriteConfig(); err != nil {
		c.JSON(500, gin.H{"message": "配置写入失败：" + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除 " + in.Name})
}

// ProvidersTest POST /adapter-service/providers/test —— 连通性测试（真实请求）
func ProvidersTest(c *gin.Context) {
	var in Provider
	if err := c.ShouldBindJSON(&in); err != nil || in.BaseURL == "" || in.APIKey == "" {
		c.JSON(400, gin.H{"message": "接口地址与密钥必填"})
		return
	}
	client := &http.Client{Timeout: 20 * time.Second}
	body := fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"回复OK"}],"max_tokens":2000}`, in.Model)
	req, _ := http.NewRequest("POST", strings.TrimSuffix(in.BaseURL, "/")+"/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+in.APIKey)
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(200, gin.H{"ok": false, "message": "连接失败：" + err.Error()})
		return
	}
	defer resp.Body.Close()
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error any `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode != 200 {
		c.JSON(200, gin.H{"ok": false, "message": fmt.Sprintf("HTTP %d：%v", resp.StatusCode, out.Error)})
		return
	}
	if len(out.Choices) == 0 || strings.TrimSpace(out.Choices[0].Message.Content) == "" {
		c.JSON(200, gin.H{"ok": false, "message": "模型返回空正文（推理模型可能需要更大的 max_tokens）"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "model": in.Model, "message": "连通成功"})
}

// Auth 用户鉴权中间件
