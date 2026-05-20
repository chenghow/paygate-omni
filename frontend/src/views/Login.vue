<template>
  <div class="login-container">
    <el-card class="login-card">
      <template #header>
        <h2>PayGate Omni 管理后台</h2>
      </template>
      <el-form :model="form" @keyup.enter="handleLogin">
        <el-form-item>
          <el-input v-model="form.password" type="password" placeholder="请输入管理员密码" show-password />
        </el-form-item>
        <el-button type="primary" class="w-full" @click="handleLogin" :loading="loading">
          登 录
        </el-button>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'

const router = useRouter()
const form = ref({ password: '' })
const loading = ref(false)

const handleLogin = async () => {
  if (!form.value.password) {
    ElMessage.warning('密码不能为空')
    return
  }
  loading.value = true
  try {
    const res = await fetch('/api/v1/admin/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: form.value.password })
    })
    const data = await res.json()
    if (data.code === 'SUCCESS' && data.token) {
      localStorage.setItem('admin_token', data.token)
      ElMessage.success('登录成功')
      router.push('/')
    } else {
      ElMessage.error(data.message || '登录失败，密码错误')
    }
  } catch (err: any) {
    ElMessage.error('网络错误: ' + err.message)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
}
.login-card {
  width: 400px;
  text-align: center;
}
.w-full {
  width: 100%;
}
</style>
