<template>
  <n-modal v-model:show="newRowModal.visible" preset="dialog" draggable :title="newRowModal.type + ' Row'" :showIcon="false">
    <template #default>
      <n-form ref="formRef" :model="newRowModal.formModel">
        <n-form-item v-for="column in tableInfo.columns" :key="column.key" :label="column.title" :path="column.key"
          :rule="{ required: column.notNull && !column.autoIncrement, message: `${column.title} is required` }">
          <n-input-number v-model:value="newRowModal.formModel[column.key]" v-if="column.type === 'INTEGER'" style="width: 100%;" />
          <n-input v-model:value="newRowModal.formModel[column.key]" :placeholder="`Enter ${column.title}`" v-else />
        </n-form-item>
      </n-form>
    </template>

    <template #action>
      <div class="flex justify-end">
        <n-button @click="newRowCancel">Cancel</n-button>
        &nbsp;
        <n-button type="primary" @click="newRowConfirm" :loading="dialogLoading">Confirm</n-button>
      </div>
    </template>

  </n-modal>
</template>

<script setup name="NewRowModal">
import { reactive, ref } from "vue";
import { useMessage } from "naive-ui";
import { newTableRowRequest, updateTableRowRequest } from '@/services'

const emits = defineEmits(['confirm']);

const tableInfo = reactive({
  name: '',
  columns: [],
});

const message = useMessage();

const formRef = ref(null);

const newRowModal = reactive({
  visible: false,
  type: "New",
  formModel: {

  }
})

function setVisible(visible, type, data) {
  newRowModal.visible = visible;
  newRowModal.type = type || "New";
  if (data) {
    tableInfo.name = data.name;
    tableInfo.columns = data.columns;
    if (type === 'Edit') {
      newRowModal.formModel = { ...data.row };
    } else {
      newRowModal.formModel = {};
    }
  }
}

defineExpose({
  setVisible,
});

function newRowCancel() {
  newRowModal.visible = false;
}

const dialogLoading = ref(false);

function newRowConfirm() {
  formRef.value?.validate((errors) => {
    if (errors) {
      return;
    }
    const params = tableInfo.columns.reduce((acc, column) => {
      if (column.autoIncrement && !newRowModal.formModel[column.key]) {
        return acc;
      }
      acc[column.key] = newRowModal.formModel[column.key];
      return acc;
    }, {});
    dialogLoading.value = true;
    const service = newRowModal.type === 'Edit' ? updateTableRowRequest : newTableRowRequest;
    service(tableInfo.name, params).then(() => {
      newRowModal.visible = false;
      emits('confirm');
      message.success(newRowModal.type + ' row successfully');
    }).finally(() => {
      dialogLoading.value = false;
    });
  });
}

</script>