<template>
  <div class="orders-container">
    <el-card class="box-card">
      <template #header>
        <div class="card-header">
          <span>订单管理</span>
          <el-button type="primary" @click="fetchOrders">刷新</el-button>
        </div>
      </template>

      <el-table :data="tableData" style="width: 100%" v-loading="loading">
        <el-table-column type="index" label="序号" width="60" />
        <el-table-column prop="trade_no" label="系统单号" width="180" />
        <el-table-column prop="out_trade_no" label="渠道外部单号" width="180" />
        <el-table-column prop="merchant_app_id" label="商户 AppID" width="120" />
        <el-table-column prop="channel_type" label="支付渠道" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.channel_type === 'wechat' ? 'success' : 'primary'">
              {{ scope.row.channel_type === 'wechat' ? '微信支付' : '支付宝' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="amount" label="金额(元)">
          <template #default="scope">
            ￥{{ (scope.row.amount / 100).toFixed(2) }}
          </template>
        </el-table-column>
        <el-table-column prop="subject" label="商品描述" />
        <el-table-column prop="status" label="状态">
          <template #default="scope">
            <el-tag :type="getStatusType(scope.row.status)">
              {{ scope.row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="160">
          <template #default="scope">
            {{ new Date(scope.row.created_at).toLocaleString() }}
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'

const tableData = ref([])
const loading = ref(false)

const getStatusType = (status: string) => {
  switch (status) {
    case 'SUCCESS': return 'success'
    case 'PENDING': return 'warning'
    case 'FAILED': return 'danger'
    case 'CLOSED': return 'info'
    default: return ''
  }
}

const fetchOrders = async () => {
  loading.value = true
  try {
    const res = await fetch('/api/v1/admin/orders', {
      headers: { 'Authorization': `Bearer ${localStorage.getItem('admin_token')}` }
    })
    const data = await res.json()
    if (data.code === 'SUCCESS') {
      tableData.value = data.data
    } else {
      ElMessage.error(data.message || '获取订单列表失败')
    }
  } catch (e) {
    ElMessage.error('网络错误')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchOrders()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
