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
      <n-button size="small" type="primary" @click="handleNewIndex">
        <template #icon>
          <n-icon>
            <Add />
          </n-icon>
        </template>
        New Index
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
              <n-popconfirm :positive-button-props="{ type: 'error' }" @positive-click="handleDeleteItem(row)">
                <template #trigger>
                  <n-button size="small" type="error">Drop</n-button>
                </template>
                Are you sure to delete this index?
              </n-popconfirm>
            </td>
          </tr>
        </tbody>
      </n-table>
    </n-spin>

    <n-modal v-model:show="newIndexModal.visible" preset="dialog" draggable title="New Column" :showIcon="false" :mask-closable="false">
      <template #default>
        <n-form ref="formRef" :model="newIndexModal.formModel">
          <n-form-item label="Index Name" path="name" :rule="{ required: true, message: 'Index name is required' }">
            <n-input v-model:value="newIndexModal.formModel.name" placeholder="Enter index name" />
          </n-form-item>
          <n-form-item label="Columns" path="columns" :rule="{ required: true, message: 'At least one column is required' }">
            <n-select v-model:value="newIndexModal.formModel.columns" multiple :options="columnOptions" placeholder="Select columns" />
          </n-form-item>
          <n-form-item label="Unique" path="unique">
            <n-switch v-model:value="newIndexModal.formModel.unique" />
          </n-form-item>
        </n-form>
      </template>

      <template #action>
        <div class="flex justify-end">
          <n-button @click="newIndexCancel">Cancel</n-button>
          &nbsp;
          <n-button type="primary" @click="newIndexConfirm" :loading="dialogLoading">Confirm</n-button>
        </div>
      </template>

    </n-modal>
  </div>
</template>

<script setup name="Indexed">
import { reactive, ref, onMounted, watch } from "vue"
import { Add, Refresh } from '@vicons/ionicons5';
import { listTableIndexesRequest, deleteTableIndexRequest, addTableIndexRequest, listColumnsRequest } from '@/services'
import { useMessage, } from 'naive-ui';

const message = useMessage()

const props = defineProps({
  table: {
    type: String,
    default: '',
  },
})

watch(() => props.table, () => {
  getTableData();
})

const columns = [
  { label: "Name", prop: "name" },
  { label: "Unique", prop: "unique" },
  { label: "SQL", prop: "sql" },
  { label: "Columns", prop: "columns" }
]

const tableLoading = ref(false);

const tableData = ref([])

function getTableData() {
  tableLoading.value = true;
  listTableIndexesRequest(props.table).then((res) => {
    tableData.value = res.data;
  }).finally(() => {
    tableLoading.value = false;
  });
}

onMounted(() => {
  getTableData();
})

const columnOptions = ref([])

function loadColumnOptions() {
  listColumnsRequest(props.table).then((res) => {
    columnOptions.value = res.data.map(col => ({
      label: col.name,
      value: col.name
    }))
  })
}

const newIndexModal = reactive({
  visible: false,
  formModel: {
    name: '',
    columns: [],
    unique: false,
  }
})

const formRef = ref(null);

const dialogLoading = ref(false);

function handleNewIndex() {
  newIndexModal.visible = true;
  newIndexModal.formModel.name = '';
  newIndexModal.formModel.columns = [];
  newIndexModal.formModel.unique = false;
  loadColumnOptions();
}

function newIndexCancel() {
  newIndexModal.visible = false;
}

function newIndexConfirm() {
  formRef.value.validate().then(() => {
    const params = { ...newIndexModal.formModel };
    dialogLoading.value = true;
    addTableIndexRequest(props.table, params).then(() => {
      message.success('Add index successfully');
      newIndexModal.visible = false;
      getTableData();
    }).finally(() => {
      dialogLoading.value = false;
    });
  })
}

function handleDeleteItem(row) {
  const messageReactive = message.loading('loading', {
    duration: 0
  });
  deleteTableIndexRequest(props.table, row.name).then(() => {
    message.success('Delete index successfully');
    getTableData();
  }).finally(() => {
    messageReactive.destroy();
  });
}
</script>