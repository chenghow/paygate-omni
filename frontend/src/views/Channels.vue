<template>
  <div class="channels-container">
    <el-card class="box-card">
      <template #header>
        <div class="card-header">
          <span>支付渠道配置</span>
          <el-button type="primary" @click="openDialog()">新建渠道</el-button>
        </div>
      </template>

      <el-table :data="tableData" style="width: 100%" v-loading="loading">
        <el-table-column type="index" label="序号" width="60" />
        <el-table-column prop="merchant_app_id" label="商户 AppID" />
        <el-table-column prop="channel_type" label="渠道类型" />
        <el-table-column prop="app_id" label="渠道 AppID" />
        <el-table-column prop="is_sandbox" label="沙盒">
          <template #default="scope">
            <el-tag :type="scope.row.is_sandbox ? 'warning' : 'success'">
              {{ scope.row.is_sandbox ? '开发(沙盒)' : '生产' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="is_active" label="状态">
          <template #default="scope">
            <el-tag :type="scope.row.is_active ? 'success' : 'info'">
              {{ scope.row.is_active ? '已启用' : '已禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180">
          <template #default="scope">
            <el-button size="small" @click="openDialog(scope.row)">编辑</el-button>
            <el-button size="small" type="danger" @click="deleteChannel(scope.row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog :title="dialogForm.id ? '编辑渠道' : '新建渠道'" v-model="dialogVisible">
      <el-form :model="dialogForm" label-width="140px">
        <el-form-item label="归属商户 AppID">
          <el-input v-model="dialogForm.merchant_app_id" :disabled="!!dialogForm.id"></el-input>
        </el-form-item>
        <el-form-item label="渠道类型">
          <el-select v-model="dialogForm.channel_type" :disabled="!!dialogForm.id">
            <el-option label="微信支付" value="wechat"></el-option>
            <el-option label="支付宝" value="alipay"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="渠道 AppID">
          <el-input v-model="dialogForm.app_id"></el-input>
        </el-form-item>
        <el-form-item label="商户号 MchID" v-if="dialogForm.channel_type === 'wechat'">
          <el-input v-model="dialogForm.mch_id"></el-input>
        </el-form-item>
        <el-form-item label="证书序列号" v-if="dialogForm.channel_type === 'wechat'">
          <el-input v-model="dialogForm.serial_no"></el-input>
        </el-form-item>
        <el-form-item label="微信 APIv3 Key" v-if="dialogForm.channel_type === 'wechat'">
          <el-input v-model="dialogForm.api_v3_key" type="password"></el-input>
        </el-form-item>
        <el-form-item label="应用私钥 (PEM)" v-if="true">
          <el-input v-model="dialogForm.private_key" type="textarea" rows="4"></el-input>
        </el-form-item>
        <el-form-item label="支付宝公钥 (PEM)" v-if="dialogForm.channel_type === 'alipay'">
          <el-input v-model="dialogForm.alipay_public_key" type="textarea" rows="4"></el-input>
        </el-form-item>
        <el-form-item label="沙盒模式" v-if="dialogForm.channel_type === 'alipay'">
          <el-switch v-model="dialogForm.is_sandbox"></el-switch>
        </el-form-item>
        <el-form-item label="启用状态">
          <el-switch v-model="dialogForm.is_active"></el-switch>
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="saveChannel">保存</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const tableData = ref([])
const loading = ref(false)
const dialogVisible = ref(false)

const dialogForm = ref({
  id: 0,
  merchant_app_id: '',
  channel_type: 'wechat',
  app_id: '',
  mch_id: '',
  serial_no: '',
  api_v3_key: '',
  private_key: '',
  alipay_public_key: '',
  is_sandbox: false,
  is_active: true
})

const fetchChannels = async () => {
  loading.value = true
  try {
    const res = await fetch('/api/v1/admin/channels', {
      headers: { 'Authorization': `Bearer ${localStorage.getItem('admin_token')}` }
    })
    const data = await res.json()
    if (data.code === 'SUCCESS') {
      tableData.value = data.data
    } else {
      ElMessage.error(data.message || '获取渠道列表失败')
    }
  } catch (e) {
    ElMessage.error('网络错误')
  } finally {
    loading.value = false
  }
}

const openDialog = (row?: any) => {
  if (row) {
    dialogForm.value = { ...row }
  } else {
    dialogForm.value = {
      id: 0,
      merchant_app_id: '',
      channel_type: 'wechat',
      app_id: '',
      mch_id: '',
      serial_no: '',
      api_v3_key: '',
      private_key: '',
      alipay_public_key: '',
      is_sandbox: false,
      is_active: true
    }
  }
  dialogVisible.value = true
}

const saveChannel = async () => {
  const isEdit = !!dialogForm.value.id
  const url = isEdit ? `/api/v1/admin/channels/${dialogForm.value.id}` : '/api/v1/admin/channels'
  const method = isEdit ? 'PUT' : 'POST'

  try {
    const res = await fetch(url, {
      method,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('admin_token')}`
      },
      body: JSON.stringify(dialogForm.value)
    })
    const data = await res.json()
    if (data.code === 'SUCCESS') {
      ElMessage.success('保存成功')
      dialogVisible.value = false
      fetchChannels()
    } else {
      ElMessage.error(data.message || '保存失败')
    }
  } catch (e) {
    ElMessage.error('网络错误')
  }
}

const deleteChannel = (id: number) => {
  ElMessageBox.confirm('确定要删除此渠道配置吗？', '提示', { type: 'warning' }).then(async () => {
    try {
      const res = await fetch(`/api/v1/admin/channels/${id}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${localStorage.getItem('admin_token')}` }
      })
      const data = await res.json()
      if (data.code === 'SUCCESS') {
        ElMessage.success('删除成功')
        fetchChannels()
      } else {
        ElMessage.error(data.message || '删除失败')
      }
    } catch (e) {
      ElMessage.error('网络错误')
    }
  }).catch(() => {})
}

onMounted(() => {
  fetchChannels()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
