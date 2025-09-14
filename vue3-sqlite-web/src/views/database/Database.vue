<template>
  <div>
    <n-descriptions label-placement="left" :column="1" :bordered="true">
      <n-descriptions-item label="path">
        {{ database.path }}
      </n-descriptions-item>
      <n-descriptions-item label="size">
        {{ (database.size / 1024).toFixed(2) }} KB
      </n-descriptions-item>
      <n-descriptions-item label="created at">
        {{ formatDate(database.createdAt) }}
      </n-descriptions-item>
      <n-descriptions-item label="modified at">
        {{ formatDate(database.modifiedAt) }}
      </n-descriptions-item>
      <n-descriptions-item label="sqlite version">
        {{ database.sqliteVersion }}
      </n-descriptions-item>
    </n-descriptions>
  </div>
</template>

<script setup name="Database">
import { reactive, onMounted } from 'vue';
import { databaseInfoRequest } from '@/services';

const database = reactive({
  path: '',
  size: 0,
  createdAt: '',
  modifiedAt: '',
  sqliteVersion: '',
})

function requestDatabaseInfo() {
  databaseInfoRequest().then(res => {
    Object.assign(database, res.data)
  })
}

function formatDate(timestamp) {
  const date = new Date(timestamp)
  return date.toLocaleString()
}

onMounted(() => {
  requestDatabaseInfo()
})

</script>