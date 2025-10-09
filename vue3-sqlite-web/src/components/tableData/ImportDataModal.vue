<template>
  <n-modal v-model:show="visible" preset="dialog" draggable title="Import Data" :showIcon="false">
    <template #default>
      <n-form ref="formRef" :model="formModel">
        <n-form-item label="File" path="fileList" :rule="{ required: true, message: 'Please select a file' }">
          <n-upload v-model:file-list="formModel.fileList" accept=".json,.csv" :max="1" :show-file-list="true" style="flex: 1; ">
            <n-button size="small" type="info">
              <template #icon>
                <n-icon>
                  <CloudUploadOutline />
                </n-icon>
              </template>
              Select
            </n-button>
          </n-upload>
        </n-form-item>
        <n-form-item label="NewColumn" path="newColumn">
          <template #label>
            <span>
              <span class="form-label">NewColumn</span>
              <n-tooltip trigger="hover">
                <template #trigger>
                  <n-icon class="pointer">
                    <svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 24 24">
                      <g fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <circle cx="12" cy="12" r="9"></circle>
                        <path d="M12 8h.01"></path>
                        <path d="M11 12h1v4h1"></path>
                      </g>
                    </svg>
                  </n-icon>
                </template>
                <div>If the imported data contains columns that do not exist in the current table, whether to create new columns.</div>
              </n-tooltip>
            </span>
          </template>
          <n-switch v-model:value="formModel.newColumn" />
        </n-form-item>
        <n-form-item label="Rollback" path="rollback">
          <template #label>
            <span>
              <span class="form-label">Rollback</span>
              <n-tooltip trigger="hover">
                <template #trigger>
                  <n-icon class="pointer">
                    <svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 24 24">
                      <g fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <circle cx="12" cy="12" r="9"></circle>
                        <path d="M12 8h.01"></path>
                        <path d="M11 12h1v4h1"></path>
                      </g>
                    </svg>
                  </n-icon>
                </template>
                <div>If the import fails, whether to roll back the changes.</div>
              </n-tooltip>
            </span>
          </template>
          <n-switch v-model:value="formModel.rollback" />
        </n-form-item>
        <div>{{ message }}</div>
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

<script setup name="ImportDataModal">
import { reactive, ref } from 'vue';
import { importTableDataRequest } from '@/services'
import { CloudUploadOutline, } from '@vicons/ionicons5';

const visible = ref(false);


const message = ref("")

const formModel = reactive({
  tableName: "",
  fileList: [],
  newColumn: false,
  rollback: false,
})

function setVisible(bool, info) {
  visible.value = bool;
  if (info) {
    formModel.tableName = info.name;
    formModel.fileList = []
    formModel.newColumn = false;
    formModel.rollback = false;
    message.value = ""
  }
}

defineExpose({
  setVisible,
});

function handleCancel() {
  visible.value = false;
}

const dialogLoading = ref(false);

const formRef = ref(null);
function handleConfirm() {
  formRef.value?.validate((errors) => {
    if (errors) {
      return;
    }
    const params = new FormData();
    params.append("file", formModel.fileList[0].file);
    params.append("createNewColumn", formModel.newColumn);
    params.append("rollback", formModel.rollback);

    dialogLoading.value = true;
    importTableDataRequest(formModel.tableName, params).then(({ data }) => {
      const msg = `${data.SuccessCount} rows imported successfully, ${data.FailedCount} rows failed.`;
      message.value = msg;
    }).finally(() => {
      dialogLoading.value = false;
    });
  })
}

</script>

<style lang="less" scoped>
.form-label {
  margin-right: 4px;
}
</style>
