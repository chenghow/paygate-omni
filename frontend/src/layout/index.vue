<template>
  <el-container class="layout-container">
    <el-aside width="200px" class="aside">
      <div class="logo">PayGate Omni</div>
      <el-menu default-active="/dashboard" router background-color="#304156" text-color="#bfcbd9" active-text-color="#409eff">
        <el-menu-item index="/dashboard">
          <span>总览面板</span>
        </el-menu-item>
        <el-menu-item index="/merchants">
          <span>商户管理</span>
        </el-menu-item>
        <el-menu-item index="/orders">
          <span>订单管理</span>
        </el-menu-item>
        <el-menu-item index="/channels">
          <span>渠道配置</span>
        </el-menu-item>
        <el-menu-item index="/test">
          <span>接口测试收银台</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <div class="right">
          <el-button type="danger" link @click="handleLogout">退出登录</el-button>
        </div>
      </el-header>
      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
const router = useRouter()

const handleLogout = () => {
  localStorage.removeItem('admin_token')
  router.push('/login')
}
</script>

<style scoped>
.layout-container { height: 100vh; }
.aside { background-color: #304156; }
.logo { height: 60px; line-height: 60px; text-align: center; color: #fff; font-size: 20px; font-weight: bold; background: #2b3643; }
.header { background-color: #fff; box-shadow: 0 1px 4px rgba(0,21,41,.08); display: flex; align-items: center; justify-content: flex-end; }
.el-menu { border-right: none; }
</style>
