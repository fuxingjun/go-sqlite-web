<template>
  <n-modal v-model:show="visible" preset="dialog" draggable title="Export Data" :showIcon="false">
    <template #default>
      <n-form ref="formRef" :model="formModel">
        <n-form-item label="Columns" path="columns">
          <n-select v-model:value="formModel.columns" multiple :options="tableInfo.columns" placeholder="Select columns" />
        </n-form-item>
        <n-form-item label="Page" path="page" :rule="{ required: true, message: 'Page is required' }">
          <n-input-number v-model:value="formModel.page" :min="1" />
        </n-form-item>
        <n-form-item label="Size" path="size" :rule="{ required: true, message: 'Size is required' }">
          <n-input-number v-model:value="formModel.size" :min="1" />
        </n-form-item>
        <n-form-item label="File Type" path="fileType" :rule="{ required: true, message: 'File type is required' }">
          <n-radio-group v-model:value="formModel.fileType">
            <n-radio value="json">json</n-radio>
            <n-radio value="csv">csv</n-radio>
          </n-radio-group>
        </n-form-item>
      </n-form>
    </template>

    <template #action>
      <div class="flex justify-end">
        <n-button @click="handleCancel">Cancel</n-button>
        &nbsp;
        <n-button type="primary" @click="handleConfirm" :loading="dialogLoading">Confirm</n-button>
      </div>
    </template>
  </n-modal>
</template>

<script setup name="ExportDataModal">
import { reactive } from 'vue';
import { exportTableDataRequest } from '@/services'


const tableInfo = reactive({
  name: '',
  columns: [],
});

const visible = ref(false);

const formModel = reactive({
  columns: [],
  page: 1,
  size: 1000,
  fileType: "csv",
})

function setVisible(bool, info) {
  visible.value = bool;
  if (info) {
    tableInfo.name = info.name;
    tableInfo.columns = info.columns.map(col => ({ label: col.title, value: col.key }));
    formModel.columns = tableInfo.columns.map(col => col.value);
  }
}

defineExpose({
  setVisible,
});

function handleCancel() {
  visible.value = false;
}

const dialogLoading = ref(false);

function handleConfirm() {
  const params = {
    columns: formModel.columns,
    page: formModel.page,
    size: formModel.size,
    fileType: formModel.fileType,
  };
  dialogLoading.value = true;
  exportTableDataRequest(tableInfo.name, params).then(([blob, filename]) => {
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    // 从响应头取文件名称
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    visible.value = false;
  }).finally(() => {
    dialogLoading.value = false;
  });
}

</script>