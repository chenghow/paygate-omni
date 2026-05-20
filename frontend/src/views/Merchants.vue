<template>
  <div class="merchants-container">
    <el-card class="box-card">
      <template #header>
        <div class="card-header">
          <span>商户管理</span>
          <el-button type="primary" @click="openDialog()">新建商户</el-button>
        </div>
      </template>

      <el-table :data="tableData" style="width: 100%" v-loading="loading">
        <el-table-column type="index" label="序号" width="60" />
        <el-table-column prop="name" label="商户名称" />
        <el-table-column prop="app_id" label="App ID" />
        <el-table-column prop="total_volume" label="总流水(元)">
          <template #default="scope">
            ￥{{ (scope.row.total_volume / 100).toFixed(2) }}
          </template>
        </el-table-column>
        <el-table-column prop="is_active" label="状态">
          <template #default="scope">
            <el-tag :type="scope.row.is_active ? 'success' : 'info'">
              {{ scope.row.is_active ? '已启用' : '已禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间">
          <template #default="scope">
            {{ new Date(scope.row.created_at).toLocaleString() }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180">
          <template #default="scope">
            <el-button size="small" @click="openDialog(scope.row)">编辑</el-button>
            <el-button size="small" type="danger" @click="deleteMerchant(scope.row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog :title="dialogForm.id ? '编辑商户' : '新建商户'" v-model="dialogVisible">
      <el-form :model="dialogForm" label-width="120px">
        <el-form-item label="商户名称">
          <el-input v-model="dialogForm.name"></el-input>
        </el-form-item>
        <el-form-item label="App ID">
          <el-input v-model="dialogForm.app_id" :disabled="!!dialogForm.id"></el-input>
        </el-form-item>
        <el-form-item label="Secret Key">
          <el-input v-model="dialogForm.secret_key" type="password" show-password></el-input>
        </el-form-item>
        <el-form-item label="启用状态">
          <el-switch v-model="dialogForm.is_active"></el-switch>
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="saveMerchant">保存</el-button>
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
  name: '',
  app_id: '',
  secret_key: '',
  is_active: true
})

const fetchMerchants = async () => {
  loading.value = true
  try {
    const res = await fetch('/api/v1/admin/merchants', {
      headers: { 'Authorization': `Bearer ${localStorage.getItem('admin_token')}` }
    })
    const data = await res.json()
    if (data.code === 'SUCCESS') {
      tableData.value = data.data
    } else {
      ElMessage.error(data.message || '获取商户列表失败')
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
      name: '',
      app_id: 'app_' + Math.random().toString(36).substr(2, 9),
      secret_key: Math.random().toString(36).substr(2, 15) + Math.random().toString(36).substr(2, 15),
      is_active: true
    }
  }
  dialogVisible.value = true
}

const saveMerchant = async () => {
  const isEdit = !!dialogForm.value.id
  const url = isEdit ? `/api/v1/admin/merchants/${dialogForm.value.id}` : '/api/v1/admin/merchants'
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
      fetchMerchants()
    } else {
      ElMessage.error(data.message || '保存失败')
    }
  } catch (e) {
    ElMessage.error('网络错误')
  }
}

const deleteMerchant = (id: number) => {
  ElMessageBox.confirm('确定要删除此商户吗？', '提示', { type: 'warning' }).then(async () => {
    try {
      const res = await fetch(`/api/v1/admin/merchants/${id}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${localStorage.getItem('admin_token')}` }
      })
      const data = await res.json()
      if (data.code === 'SUCCESS') {
        ElMessage.success('删除成功')
        fetchMerchants()
      } else {
        ElMessage.error(data.message || '删除失败')
      }
    } catch (e) {
      ElMessage.error('网络错误')
    }
  }).catch(() => {})
}

onMounted(() => {
  fetchMerchants()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
