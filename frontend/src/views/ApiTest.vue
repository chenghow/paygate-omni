<template>
  <div class="api-test">
    <el-card class="box-card">
      <template #header>
        <div class="card-header">
          <span>商户下单网关测试 (API Test)</span>
        </div>
      </template>

      <el-form label-width="120px" :model="form" @submit.prevent>
        <el-form-item label="网关地址">
          <el-input v-model="gatewayUrl" disabled></el-input>
        </el-form-item>
        
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="AppID">
              <el-input v-model="form.appId" placeholder="请输入商户 AppID"></el-input>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="SecretKey">
              <el-input v-model="form.secretKey" placeholder="请输入商户 SecretKey" show-password></el-input>
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item label="请求体 (JSON)">
          <el-input
            v-model="form.body"
            type="textarea"
            :rows="8"
            placeholder="请输入 JSON 格式的下单请求"
          ></el-input>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="sendRequest" :loading="loading">发送请求 (模拟商户)</el-button>
          <el-button @click="resetForm">重置默认数据</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <br />

    <el-row :gutter="20" v-if="requestInfo || responseInfo">
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header"><span>请求信息 (Request)</span></div>
          </template>
          <pre class="code-block">{{ requestInfo }}</pre>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header"><span>响应结果 (Response)</span></div>
          </template>
          <pre class="code-block">{{ responseInfo }}</pre>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import CryptoJS from 'crypto-js'

const gatewayUrl = ref('/api/v1/pay/create')
const loading = ref(false)

const getMockTradeNo = () => 'TEST_' + new Date().getTime()

const defaultBody = () => JSON.stringify({
  merchant_trade_no: getMockTradeNo(),
  amount: 1,
  subject: "Test Item",
  channel_type: "alipay",
  notify_url: "https://httpbin.org/post"
}, null, 2)

const form = ref({
  appId: '',
  secretKey: '',
  body: defaultBody()
})

const requestInfo = ref('')
const responseInfo = ref('')

const resetForm = () => {
  form.value.body = defaultBody()
}

const generateNonce = (length = 16) => {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let result = ''
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return result
}

const sendRequest = async () => {
  if (!form.value.appId || !form.value.secretKey) {
    ElMessage.error('请填写 AppID 和 SecretKey')
    return
  }

  // 1. Minify JSON string (remove extra spaces for robust signing)
  let rawBody = form.value.body
  try {
    const parsed = JSON.parse(rawBody)
    rawBody = JSON.stringify(parsed)
  } catch (e) {
    ElMessage.error('请求体 JSON 格式不正确')
    return
  }

  // 2. Prepare headers values
  const timestamp = Math.floor(Date.now() / 1000).toString()
  const nonce = generateNonce(16)

  // 3. Calculate Signature per AGENTS.md §4.2
  // payload: timestamp + "\n" + nonce + "\n" + raw_body
  const sigPayload = `${timestamp}\n${nonce}\n${rawBody}`
  const signature = CryptoJS.HmacSHA256(sigPayload, form.value.secretKey).toString(CryptoJS.enc.Hex)

  const headers = {
    'Content-Type': 'application/json',
    'X-Pay-AppID': form.value.appId,
    'X-Pay-Timestamp': timestamp,
    'X-Pay-Nonce': nonce,
    'X-Pay-Signature': signature
  }

  requestInfo.value = JSON.stringify({
    method: 'POST',
    url: gatewayUrl.value,
    headers: headers,
    body: JSON.parse(rawBody)
  }, null, 2)

  loading.value = true
  responseInfo.value = 'Loading...'

  try {
    const response = await fetch(gatewayUrl.value, {
      method: 'POST',
      headers: headers,
      body: rawBody
    })
    
    const resData = await response.json()
    responseInfo.value = `Status: ${response.status} ${response.statusText}\n${JSON.stringify(resData, null, 2)}`
    
    if (response.ok && resData.code === 'SUCCESS') {
      ElMessage.success('下单成功！')
    } else {
      ElMessage.error(`请求失败: ${resData.message || 'Unknown error'}`)
    }
  } catch (err: any) {
    ElMessage.error('网络或未知错误')
    responseInfo.value = err.toString()
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.box-card {
  margin-bottom: 20px;
}
.card-header {
  font-weight: bold;
}
.code-block {
  background-color: #f5f7fa;
  padding: 10px;
  border-radius: 4px;
  max-height: 400px;
  overflow: auto;
  font-family: monospace;
  font-size: 13px;
  white-space: pre-wrap;
}
</style>
