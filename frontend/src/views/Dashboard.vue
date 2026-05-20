<template>
  <div class="dashboard">
    <h2>系统总览</h2>
    <el-row :gutter="20">
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <template #header>今日流水(元)</template>
          <div class="stat-num amount">￥{{ (stats.today_amount / 100).toFixed(2) }}</div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <template #header>总流水(元)</template>
          <div class="stat-num amount">￥{{ (stats.total_amount / 100).toFixed(2) }}</div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <template #header>总订单数</template>
          <div class="stat-num">{{ stats.order_count }}</div>
        </el-card>
      </el-col>
    </el-row>
    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <template #header>接入商户数</template>
          <div class="stat-num">{{ stats.merchant_count }}</div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <template #header>开启渠道数</template>
          <div class="stat-num">{{ stats.channel_count }}</div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'

const stats = ref({ merchant_count: 0, channel_count: 0, order_count: 0, total_amount: 0, today_amount: 0 })

onMounted(async () => {
  try {
    const res = await fetch('/api/v1/admin/stats', {
      headers: { 'Authorization': 'Bearer ' + localStorage.getItem('admin_token') }
    })
    if (res.status === 401) {
      ElMessage.error('认证已过期，请重新登录')
      localStorage.removeItem('admin_token')
      window.location.href = '/login'
      return
    }
    const data = await res.json()
    if (data.code === 'SUCCESS') {
      stats.value = data.data
    }
  } catch (err: any) {
    ElMessage.error('获取统计失败: ' + err.message)
  }
})
</script>

<style scoped>
.stat-card {
  text-align: center;
}
.stat-num {
  font-size: 32px;
  font-weight: bold;
  color: #409eff;
}
.amount {
  color: #67c23a;
}
</style>
