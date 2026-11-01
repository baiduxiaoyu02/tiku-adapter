<template>
  <div style="width: 100%">
    <a-card style="margin-bottom: 16px">
      <div class="toolbar">
        <span class="tip">配置自定义模型供应商，保存并设为当前后搜题即用该配置。</span>
        <a-button icon="reload" @click="handleLoad">刷新</a-button>
        <a-button type="primary" icon="plus" @click="onNew">添加供应商</a-button>
      </div>
    </a-card>

    <div class="layout">
      <!-- 左列：供应商列表 -->
      <a-card class="side" title="供应商">
        <div
          v-for="p in providers"
          :key="p.name"
          class="p-item"
          :class="{ active: p.name === selected.name, disabled: p.disabled }"
          @click="onSelect(p)"
        >
          <span class="dot" :class="dotClass(p)"></span>
          <span class="p-name">{{ p.name }}</span>
          <span v-if="p.name === activeName" class="cur-tag">当前</span>
        </div>
        <div v-if="!providers.length" class="empty-tip">暂无供应商，点右上「添加供应商」</div>
      </a-card>

      <!-- 右列：编辑表单 -->
      <a-card class="main" :title="isNew ? '新供应商' : selected.name">
        <template #extra>
          <span class="switch-label">{{ selected.disabled ? '已停用' : '启用中' }}</span>
          <a-switch :checked="!selected.disabled" @change="v => selected.disabled = !v" />
        </template>
        <a-form :label-col="labelCol" :wrapper-col="wrapperCol">
          <a-form-item label="名称" required>
            <a-input v-model:value="selected.name" placeholder="如：主力 / 备用" />
          </a-form-item>
          <a-form-item label="Base URL" required>
            <a-input v-model:value="selected.base_url" placeholder="https://api.example.com/v1" />
          </a-form-item>
          <a-form-item label="API 格式">
            <a-select v-model:value="selected.api_format">
              <a-select-option value="anthropic">Anthropic Messages (/v1/messages)</a-select-option>
              <a-select-option value="openai">Chat Completions (/chat/completions)</a-select-option>
              <a-select-option value="responses">Responses (/responses)</a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item label="API Key" :required="!hasStoredKey">
            <a-input-password v-model:value="selected.api_key" :placeholder="hasStoredKey ? '留空 = 不修改密钥' : '输入 API Key'" />
          </a-form-item>
          <a-form-item label="模型列表">
            <div v-for="(m, i) in selected.models" :key="i" class="model-row">
              <a-input v-model:value="selected.models[i]" placeholder="模型名称，如 glm-5.2" />
              <a-button icon="delete" @click="selected.models.splice(i, 1)" />
            </div>
            <a-button type="dashed" block icon="plus" @click="selected.models.push('')">添加模型</a-button>
          </a-form-item>
          <a-form-item label="文字搜题模型">
            <a-select v-model:value="selected.model" placeholder="从模型列表选择">
              <a-select-option v-for="m in modelOptions" :key="m" :value="m">{{ m }}</a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item label="图片搜题模型">
            <a-select v-model:value="selected.vision_model" placeholder="可选，图片题用">
              <a-select-option v-for="m in modelOptions" :key="m" :value="m">{{ m }}</a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item :wrapper-col="{ span: wrapperCol.span }">
            <a-space>
              <a-button type="primary" :loading="saving" @click="onSave">保存并设为当前</a-button>
              <a-button :loading="testing" @click="onTest">测试连通</a-button>
              <a-popconfirm v-if="!isNew" title="确定删除该供应商吗?" ok-text="确定" cancel-text="取消" @confirm="onDelete">
                <a-button type="danger">删除</a-button>
              </a-popconfirm>
            </a-space>
          </a-form-item>
        </a-form>
        <a-alert type="info" show-icon message="保存后需重启服务生效：托盘图标右键 → Restart Service" />
      </a-card>
    </div>
  </div>
</template>

<script>
import { getProviders, saveProvider, deleteProvider, testProvider } from '@/api/TikuComponentApi';
import { message } from 'ant-design-vue';
import { defineComponent, ref, computed } from 'vue';

const providers = ref([]);
const activeName = ref('');
const loading = ref(false);
const saving = ref(false);
const testing = ref(false);
const isNew = ref(false);
const emptyForm = () => ({
  name: '',
  base_url: '',
  api_format: 'openai',
  api_key: '',
  models: [],
  model: '',
  vision_model: '',
  disabled: false,
});
const selected = ref(emptyForm());

export default defineComponent({
  name: 'ProviderManage',
  setup() {
    const labelCol = { style: { width: '140px' } };
    const wrapperCol = { span: 14 };
    const modelOptions = computed(() => (selected.value.models || []).filter(m => m && m.trim()));
    const hasStoredKey = computed(() => {
      if (isNew.value) return false;
      const p = providers.value.find(x => x.name === selected.value.name);
      return !!(p && p.api_key);
    });
    return { labelCol, wrapperCol, providers, activeName, loading, saving, testing, isNew, selected, modelOptions, hasStoredKey };
  },
  mounted() {
    this.handleLoad();
  },
  methods: {
    handleLoad() {
      loading.value = true;
      getProviders().then(res => {
        loading.value = false;
        if (res?.status == 200) {
          providers.value = res.data?.providers || [];
          activeName.value = res.data?.active_name || '';
          // 选中当前供应商；没有则选中第一个；都没有则新建
          const cur = providers.value.find(p => p.name === activeName.value) || providers.value[0];
          if (cur) { this.onSelect(cur); } else { this.onNew(); }
        }
      }).catch(() => { loading.value = false; });
    },
    dotClass(p) {
      if (p.disabled) return 'gray';
      return p.name === activeName.value ? 'green' : 'orange';
    },
    onSelect(p) {
      isNew.value = false;
      selected.value = JSON.parse(JSON.stringify(p));
      selected.value.api_key = ''; // 编辑不回填密钥（列表是掩码）
      selected.value.models = (p.models || []).slice();
      if (!selected.value.api_format) selected.value.api_format = 'openai';
      if (selected.value.disabled === undefined) selected.value.disabled = false;
    },
    onNew() {
      isNew.value = true;
      selected.value = emptyForm();
    },
    onSave() {
      const f = selected.value;
      if (!f.name || !f.base_url) { message.error('名称与 Base URL 必填'); return; }
      if (!this.hasStoredKey && !f.api_key) { message.error('API Key 必填'); return; }
      const models = (f.models || []).map(m => m.trim()).filter(Boolean);
      if (f.model && !models.includes(f.model)) models.unshift(f.model);
      if (!f.model && models.length) f.model = models[0];
      saving.value = true;
      saveProvider({ ...f, models }).then(res => {
        saving.value = false;
        if (res?.status == 200) {
          message.success('已保存并设为当前，重启服务后生效（托盘右键 → Restart Service）');
          this.handleLoad();
        } else {
          message.error(res?.message || '保存失败');
        }
      }).catch(() => { saving.value = false; message.error('保存失败'); });
    },
    onTest() {
      const f = selected.value;
      if (!f.base_url || (!f.api_key && !this.hasStoredKey)) { message.error('请先填写 Base URL 与 API Key'); return; }
      if (!f.model) { message.error('请先填写或选择模型'); return; }
      testing.value = true;
      testProvider({ ...f }).then(res => {
        testing.value = false;
        if (res?.status == 200 && res.data?.ok) {
          message.success(`连通成功（${f.api_format} · ${f.model}）`);
        } else {
          message.error(res.data?.message || res?.message || '连通失败');
        }
      }).catch(() => { testing.value = false; message.error('测试失败'); });
    },
    onDelete() {
      deleteProvider({ name: selected.value.name }).then(res => {
        if (res?.status == 200) {
          message.success('已删除');
          this.handleLoad();
        } else {
          message.error(res?.message || '删除失败');
        }
      });
    },
  },
});
</script>

<style scoped>
.toolbar { display: flex; align-items: center; gap: 8px; }
.toolbar .tip { flex: 1; color: #6b7280; font-size: 13px; }
.layout { display: flex; gap: 16px; align-items: flex-start; }
.side { width: 230px; flex-shrink: 0; }
.main { flex: 1; }
.p-item { display: flex; align-items: center; gap: 8px; padding: 9px 10px; border-radius: 8px; cursor: pointer; margin-bottom: 4px; }
.p-item:hover { background: #f5f6fa; }
.p-item.active { background: #eef2ff; }
.p-item.disabled .p-name { color: #9ca3af; text-decoration: line-through; }
.p-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.dot.green { background: #22c55e; }
.dot.orange { background: #f59e0b; }
.dot.gray { background: #d1d5db; }
.cur-tag { font-size: 11px; background: #22c55e; color: #fff; border-radius: 10px; padding: 1px 8px; }
.empty-tip { color: #9ca3af; font-size: 13px; padding: 8px 0; }
.switch-label { margin-right: 8px; font-size: 13px; color: #6b7280; }
.model-row { display: flex; gap: 8px; margin-bottom: 8px; }
</style>
