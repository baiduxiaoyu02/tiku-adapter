<template>
  <div style="width: 100%">
    <a-card title="当前使用" style="margin-bottom: 16px">
      <a-descriptions :column="2" size="small" bordered>
        <a-descriptions-item label="供应商">{{ current.name || '（未配置）' }}</a-descriptions-item>
        <a-descriptions-item label="接口地址">{{ current.base_url || '（未配置）' }}</a-descriptions-item>
        <a-descriptions-item label="普通模型">{{ current.model || '（未配置）' }}</a-descriptions-item>
        <a-descriptions-item label="视觉模型">{{ current.vision_model || '（未配置）' }}</a-descriptions-item>
        <a-descriptions-item label="密钥" :span="2">{{ current.api_key || '（未配置）' }}</a-descriptions-item>
      </a-descriptions>
      <a-alert style="margin-top: 12px" type="info" show-icon
        message="修改配置后需重启服务生效：托盘图标右键 → Restart Service" />
    </a-card>

    <a-card title="供应商列表" style="margin-bottom: 16px">
      <a-table bordered :data-source="providers" :columns="columns" :loading="loading" :pagination="false">
        <template #operation="{ record }">
          <a-space>
            <a-button type="primary" size="small" :disabled="record.name === activeName" @click="onUse(record)">设为当前</a-button>
            <a-button size="small" @click="onEdit(record)">编辑</a-button>
            <a-popconfirm title="确定删除该供应商吗?" ok-text="确定" cancel-text="取消" @confirm="onDelete(record.name)">
              <a style="color: red;">删除</a>
            </a-popconfirm>
          </a-space>
        </template>
      </a-table>
      <div v-if="!providers.length && !loading" style="color: #999; padding: 8px 0;">暂无供应商，请在下方添加</div>
    </a-card>

    <a-card :title="formTitle">
      <a-form :model="formState" :label-col="labelCol" :wrapper-col="wrapperCol">
        <a-form-item label="名称" required>
          <a-input v-model:value="formState.name" placeholder="如：主力 / 备用" />
        </a-form-item>
        <a-form-item label="接口地址" required>
          <a-input v-model:value="formState.base_url" placeholder="如 https://api.deepseek.com/v1" />
        </a-form-item>
        <a-form-item label="API Key" :required="!editing">
          <a-input-password v-model:value="formState.api_key" :placeholder="editing ? '留空 = 不修改密钥' : 'sk-...'" />
        </a-form-item>
        <a-form-item label="普通模型">
          <a-input v-model:value="formState.model" placeholder="如 glm-5.2" />
        </a-form-item>
        <a-form-item label="视觉模型">
          <a-input v-model:value="formState.vision_model" placeholder="如 sensenova-6.8-flash-lite（图片题用）" />
        </a-form-item>
        <a-form-item :wrapper-col="{ span: 12 }">
          <a-space>
            <a-button type="primary" :loading="saving" @click="onSave">保存并设为当前</a-button>
            <a-button :loading="testing" @click="onTest">测试连通</a-button>
            <a-button @click="resetForm">清空表单</a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-card>
  </div>
</template>

<script>
import { getProviders, saveProvider, switchProvider, deleteProvider, testProvider } from '@/api/TikuComponentApi';
import { message } from 'ant-design-vue';
import { defineComponent, ref } from 'vue';

const current = ref({});
const providers = ref([]);
const activeName = ref('');
const loading = ref(true);
const saving = ref(false);
const testing = ref(false);
const editing = ref(false);
const emptyForm = () => ({
  name: '',
  base_url: '',
  api_key: '',
  model: '',
  vision_model: '',
});
const formState = ref(emptyForm());

export default defineComponent({
  name: 'ProviderManage',
  setup() {
    return {
      labelCol: { style: { width: '150px' } },
      wrapperCol: { span: 14 },
      current,
      providers,
      activeName,
      loading,
      saving,
      testing,
      editing,
      formState,
      columns: [
        { title: '名称', dataIndex: 'name', key: 'name' },
        { title: '接口地址', dataIndex: 'base_url', key: 'base_url' },
        { title: '普通模型', dataIndex: 'model', key: 'model' },
        { title: '视觉模型', dataIndex: 'vision_model', key: 'vision_model' },
        { title: '操作', dataIndex: 'operation', key: 'operation', slots: { customRender: 'operation' } },
      ],
    };
  },
  computed: {
    formTitle() {
      return this.editing ? `编辑供应商：${this.formState.name}` : '添加供应商';
    },
  },
  mounted() {
    this.handleLoad();
  },
  methods: {
    handleLoad() {
      loading.value = true;
      getProviders().then(res => {
        if (res?.status == 200) {
          current.value = res.data?.active || {};
          providers.value = res.data?.providers || [];
          activeName.value = res.data?.active_name || '';
        }
        loading.value = false;
      }).catch(() => {
        loading.value = false;
      });
    },
    onUse(record) {
      switchProvider({ name: record.name }).then(res => {
        if (res?.status == 200) {
          message.success('已切换，重启服务后生效（托盘右键 → Restart Service）');
          this.handleLoad();
        } else {
          message.error(res?.message || '切换失败');
        }
      });
    },
    onEdit(record) {
      editing.value = true;
      formState.value = {
        name: record.name,
        base_url: record.base_url || '',
        api_key: '',
        model: record.model || '',
        vision_model: record.vision_model || '',
      };
    },
    onDelete(name) {
      deleteProvider({ name }).then(res => {
        if (res?.status == 200) {
          message.success('已删除');
          this.handleLoad();
        } else {
          message.error(res?.message || '删除失败');
        }
      });
    },
    onSave() {
      const f = formState.value;
      if (!f.name || !f.base_url) {
        message.error('名称与接口地址必填');
        return;
      }
      if (!this.editing && !f.api_key) {
        message.error('API Key 必填');
        return;
      }
      saving.value = true;
      saveProvider({ ...f }).then(res => {
        saving.value = false;
        if (res?.status == 200) {
          message.success('已保存并设为当前，重启服务后生效（托盘右键 → Restart Service）');
          this.resetForm();
          this.handleLoad();
        } else {
          message.error(res?.message || '保存失败');
        }
      }).catch(() => {
        saving.value = false;
        message.error('保存失败');
      });
    },
    onTest() {
      const f = formState.value;
      if (!f.base_url || !f.api_key) {
        message.error('请先填写接口地址与密钥');
        return;
      }
      testing.value = true;
      testProvider({ ...f }).then(res => {
        testing.value = false;
        if (res?.status == 200 && res.data?.ok) {
          message.success(`连通成功（模型 ${res.data.model || f.model}）`);
        } else {
          message.error(res.data?.message || res?.message || '连通失败');
        }
      }).catch(() => {
        testing.value = false;
        message.error('测试失败');
      });
    },
    resetForm() {
      editing.value = false;
      formState.value = emptyForm();
    },
  },
});
</script>

<style scoped></style>
