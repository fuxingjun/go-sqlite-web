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
      <template v-if="columns.length > 0">
        <n-button size="small" type="primary" @click="handleNewRow">
          <template #icon>
            <n-icon>
              <Add />
            </n-icon>
          </template>
          Add
        </n-button>
        &nbsp;
        <n-button size="small" type="info" @click="handleImport">
          <template #icon>
            <n-icon>
              <CloudUploadOutline />
            </n-icon>
          </template>
          Import
        </n-button>
        &nbsp;
      </template>
      <n-button size="small" type="primary" @click="handleExport">
        <template #icon>
          <n-icon>
            <CloudDownloadOutline />
          </n-icon>
        </template>
        Export
      </n-button>
    </div>
    <n-data-table :columns="computeColumns" :data="tableData" :pagination="pagination" :loading="tableLoading" :bordered="true" resizable remote max-height="calc(100vh - 210px)" />

    <NewRowModal ref="newRowModalRef" @confirm="getTableData" />

    <ExportDataModal ref="exportDataModalRef" />

    <ImportDataModal ref="importDataModalRef" />
  </div>
</template>

<script setup name="TableData">
import { reactive, ref, onMounted, watch, computed } from "vue"
import { useMessage, NButton, NPopconfirm } from 'naive-ui';
import { Add, Refresh, CloudDownloadOutline, CloudUploadOutline } from '@vicons/ionicons5';
import { getTableDataRequest, listColumnsRequest, importTableDataRequest, deleteTableRowRequest } from '@/services'
import NewRowModal from './NewRowModal.vue';
import ExportDataModal from './ExportDataModal.vue';
import ImportDataModal from './ImportDataModal.vue';

const message = useMessage()

const props = defineProps({
  table: {
    type: String,
    default: '',
  },
})

watch(() => props.table, () => {
  pagination.page = 1;
  getTableData();
})

const columns = ref([])

const actionsColumn = {
  title: 'Actions', key: 'actions', width: 180, fixed: 'right', render(row) {
    return h('div', [
      h(NButton, { size: 'tiny', type: 'info', onClick: () => updateRow(row) }, () => 'Edit'),
      " ",
      h(NPopconfirm, {
        'positive-button-props': { type: 'error' },
        onPositiveClick: () => deleteRow(row)
      }, {
        trigger: () => h(NButton, { size: 'tiny', type: 'error' }, () => 'Delete'),
        default: () => 'Are you sure to delete this row?'
      })
    ]);
  }
}

const computeColumns = computed(() => {
  // 如果有主键, 支持修改和删除
  if (columns.value.find(col => col.primary)) {
    return [...columns.value, actionsColumn];
  }
  return columns.value;
})

const tableData = ref([])

const pagination = reactive({
  page: 1,
  pageSize: 50,
  pageSizes: [50, 100, 250, 500],
  itemCount: 0,
  showSizePicker: true,
  showQuickJumper: true,
  prefix(args) {
    return `Total is ${args.itemCount}.`
  },
  "onUpdate:page"(page) {
    pagination.page = page;
    getTableData();
  },
  "onUpdate:pageSize"(pageSize) {
    pagination.pageSize = pageSize;
    getTableData();
  }
})

const tableLoading = ref(false)

function getTableData() {
  const params = {
    page: pagination.page,
    limit: pagination.pageSize,
  }
  tableLoading.value = true;
  Promise.allSettled([getTableDataRequest(props.table, params), listColumnsRequest(props.table)]).then(([tableRes, columnsRes]) => {
    if (tableRes.status === "fulfilled") {
      tableData.value = tableRes.value.data.rows;
      pagination.itemCount = tableRes.value.data.total;
    }
    if (columnsRes.status === "fulfilled") {
      columns.value = columnsRes.value.data.map(item => {
        return {
          title: item.name,
          key: item.name,
          default: item.default,
          notNull: item.notNull,
          primary: item.primary,
          autoIncrement: item.autoIncrement,
          type: item.type,
          width: item.type === "TIMESTAMP" ? 240 : 180,
          ellipsis: {
            tooltip: true,
          },
          resizable: true,
        }
      });
    }
  }).finally(() => {
    tableLoading.value = false;
  })
}

function updateRow(row) {
  newRowModalRef.value.setVisible(true, "Edit", { name: props.table, columns: columns.value, row });
}

function deleteRow(row) {
  const messageReactive = message.loading('Deleting...', {
    duration: 0,
  });
  deleteTableRowRequest(props.table, row).then(() => {
    message.success('Row deleted successfully');
    getTableData();
  }).finally(() => {
    messageReactive.destroy();
  });
}

onMounted(() => {
  getTableData();
});

const newRowModalRef = ref(null)

function handleNewRow() {
  const params = {
    name: props.table,
    columns: columns.value
  }
  newRowModalRef.value.setVisible(true, "New", params);
}

const importDataModalRef = ref(null)

function handleImport() {
  const params = {
    name: props.table
  }
  importDataModalRef.value.setVisible(true, params);
}

const exportDataModalRef = ref(null)

function handleExport() {
  const params = {
    name: props.table,
    columns: columns.value
  }
  exportDataModalRef.value.setVisible(true, params);
}

</script>