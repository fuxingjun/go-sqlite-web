<template>
  <div>
    <div class="button-wrapper flex justify-end">
      <n-button size="small" type="info" @click="getTableData">
        <template #icon>
          <n-icon>
            <Refresh />
          </n-icon>
        </template>
        Refresh
      </n-button>
      &nbsp;
      <n-button size="small" type="primary" @click="handleNewColumn">
        <template #icon>
          <n-icon>
            <Add />
          </n-icon>
        </template>
        New Column
      </n-button>
    </div>
    <n-spin :show="tableLoading">
      <n-table :bordered="false" :single-line="false">
        <thead>
          <tr>
            <th v-for="column in columns">{{ column.label }}</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, index) in tableData" :key="index">
            <td v-for="column in columns">{{ row[column.prop] }}</td>
            <td width="200">
              <n-button size="small" type="info" @click="handleEditItem(row)">Rename</n-button>
              &nbsp;
              <n-popconfirm :positive-button-props="{ type: 'error' }" @positive-click="handleDeleteItem(row)">
                <template #trigger>
                  <n-button size="small" type="error">Delete</n-button>
                </template>
                Are you sure to delete this column?
              </n-popconfirm>
            </td>
          </tr>
        </tbody>
      </n-table>
    </n-spin>

    <n-modal v-model:show="newColumnModal.visible" preset="dialog" draggable title="New Column" :showIcon="false">
      <template #default>
        <n-form ref="formRef" :model="newColumnModal.formModel">
          <n-form-item label="Column Name" path="name" :rule="{ required: true, message: 'Column name is required' }">
            <n-input v-model:value="newColumnModal.formModel.name" placeholder="Enter column name" />
          </n-form-item>
          <n-form-item label="Type" path="type" :rule="{ required: true, message: 'Column type is required' }">
            <n-select v-model:value="newColumnModal.formModel.type" :options="columnTypes" placeholder="Select column type" />
          </n-form-item>
          <n-form-item label="Not Null" path="notNull">
            <n-switch v-model:value="newColumnModal.formModel.notNull" />
          </n-form-item>
          <n-form-item label="Default" path="default">
            <n-input v-model:value="newColumnModal.formModel.default" placeholder="Enter default value" />
          </n-form-item>
        </n-form>
      </template>

      <template #action>
        <div class="flex justify-end">
          <n-button @click="newColumnCancel">Cancel</n-button>
          &nbsp;
          <n-button type="primary" @click="newColumnConfirm" :loading="dialogLoading">Confirm</n-button>
        </div>
      </template>

    </n-modal>

    <n-modal v-model:show="renameModal.visible" preset="dialog" draggable title="Rename Column" :showIcon="false">
      <template #default>
        <n-form ref="formRef" :model="renameModal.formModel">
          <n-form-item label="Column Name" path="name" :rule="{ required: true, message: 'Column name is required' }">
            <n-input v-model:value="renameModal.formModel.name" placeholder="Enter new column name" />
          </n-form-item>
        </n-form>
      </template>
      <template #action>
        <div class="flex justify-end">
          <n-button @click="renameCancel">Cancel</n-button>
          &nbsp;
          <n-button type="primary" @click="renameConfirm" :loading="dialogLoading">Confirm</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<script setup name="Columns">
import { ref, onMounted, watch, reactive } from 'vue';
import { Add, Refresh } from '@vicons/ionicons5';
import { useMessage, } from 'naive-ui';
import { listColumnsRequest, addColumnRequest, renameColumnRequest, deleteColumnRequest } from '@/services';


const props = defineProps({
  table: {
    type: String,
    default: '',
  },
});

const emits = defineEmits([]);
const columns = ref([
  { label: "Name", prop: "name", },
  { label: "Type", prop: "type", },
  { label: "Unique", prop: "unique", },
  { label: "NotNull", prop: "notNull", },
  { label: "Default", prop: "default", },
  { label: "PrimaryKey", prop: "primary", },
  { label: "AutoIncrement", prop: "autoIncrement", },
]);

const tableData = ref([]);

const tableLoading = ref(false);

function getTableData() {
  tableLoading.value = true;
  listColumnsRequest(props.table).then(({ data }) => {
    tableData.value = data;
  }).finally(() => {
    tableLoading.value = false;
  });
}

onMounted(() => {
  getTableData();
});

watch(() => props.table, () => {
  getTableData();
});

const columnTypes = [
  { label: 'INTEGER', value: 'INTEGER' },
  { label: 'TEXT', value: 'TEXT' },
  { label: 'BLOB', value: 'BLOB' },
  { label: 'REAL', value: 'REAL' },
  { label: 'NUMERIC', value: 'NUMERIC' },
];

const dialogLoading = ref(false);

const newColumnModal = reactive({
  visible: false,
  formModel: {
    name: '',
    type: '',
    notNull: false,
    default: '',
  },
});

function handleNewColumn() {
  newColumnModal.visible = true;
  newColumnModal.formModel = {
    name: '',
    type: '',
    notNull: false,
    default: '',
  };
}

function newColumnConfirm() {
  formRef.value?.validate((errors) => {
    if (errors) {
      return;
    }
    const params = {
      ...newColumnModal.formModel,
      notNull: newColumnModal.formModel.notNull,
    };
    dialogLoading.value = true;
    addColumnRequest(props.table, params).then(() => {
      message.success('Column created successfully');
      getTableData();
      newColumnModal.visible = false;
    }).finally(() => {
      dialogLoading.value = false;
    });
  });
  return false;
}

function newColumnCancel() {
  newColumnModal.visible = false;
}

const renameModal = reactive({
  visible: false,
  formModel: {
    name: '',
  },
});

function handleEditItem(row) {
  renameModal.visible = true;
  renameModal.formModel.oldName = row.name;
  renameModal.formModel.name = row.name;
}

const formRef = ref(null);
const message = useMessage();

function renameConfirm() {
  formRef.value?.validate((errors) => {
    if (errors) {
      return;
    }
    dialogLoading.value = true;
    renameColumnRequest(props.table, renameModal.formModel.oldName, renameModal.formModel.name).then(() => {
      message.success('Column renamed successfully');
      renameModal.visible = false;
      getTableData();
    }).finally(() => {
      dialogLoading.value = false;
    });
  });
  return false;
}

function renameCancel() {
  renameModal.visible = false;
}

function handleDeleteItem(row) {
  const messageReactive = message.loading('loading', {
    duration: 0
  });
  deleteColumnRequest(props.table, row.name).then(() => {
    message.success('Column deleted successfully');
    getTableData();
  }).finally(() =>
    messageReactive.destroy()
  );
}

</script>

<style lang="less" scope>
.button-wrapper {
  padding: 10px 0;
}
</style>