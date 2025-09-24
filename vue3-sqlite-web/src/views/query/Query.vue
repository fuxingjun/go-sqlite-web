<template>
  <div>
    <n-form>
      <n-form-item label="SQL">
        <!-- <n-input v-model:value="sqlQuery" type="textarea" rows="10" placeholder="Enter your SQL query here..." /> -->
        <div class="editor-wrapper">
          <VAceEditor v-model:value="sqlQuery" @init="editorInit" lang="sql" theme="github" style="width: 100%; height:100%;" :options="editorOptions" />
        </div>
      </n-form-item>
      <n-form-item>
        <n-button type="primary" @click="handleExecute" :loading="loading">Execute</n-button>
      </n-form-item>

      <div :style="{ color: executeSuccess ? '' : '#d03050' }">{{ feedback }}</div>

      <!-- table -->
      <n-data-table :columns="columns" :data="tableData" :pagination="pagination" :loading="tableLoading" :bordered="true" remote v-if="columns.length" max-height="calc(100vh - 610px)" />

    </n-form>
  </div>
</template>

<script setup name="Query">
import { ref } from 'vue';
import { VAceEditor } from "vue3-ace-editor"
// cspell: ignore noconflict
import 'ace-builds/src-noconflict/mode-sql'
import 'ace-builds/src-noconflict/theme-github'
import 'ace-builds/src-noconflict/ext-language_tools'
import ace from "ace-builds"
import { executeQueryRequest } from "@/services"

ace.config.set("basePath", `/assets/ace-builds/src-noconflict/`)

const sqlQuery = ref('');

let codeEditor = null
function editorInit(editor) {
  codeEditor = editor
}

const editorOptions = {
  enableBasicAutocompletion: true, //启用基本自动完成
  enableSnippets: true, // 启用代码段
  enableLiveAutocompletion: true, // 启用实时自动完成
  fontSize: 18, //设置字号
  tabSize: 4, // 标签大小
  showPrintMargin: false, //去除编辑器里的竖线
  highlightActiveLine: true,
  // 显示行号区域
  showGutter: false,
  // 自动换行
  wrap: true,
}

const feedback = ref("")

const executeSuccess = ref(false)

const loading = ref(false)

const columns = ref([])
const tableData = ref([])

const tableLoading = ref(false)

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

function getTableData() {
  const params = {
    sql: sqlQuery.value,
    page: pagination.page,
    size: pagination.pageSize,
  }
  tableLoading.value = true;
  executeQueryRequest(params, { quite: true }).then(res => {
    const { data } = res
    feedback.value = res.code !== 0 ? (res.message || res.error) : ""
    feedback.value = data.error || data.message
    executeSuccess.value = res.code === 0 && !data.error
    if (data.columns && data.rows) {
      columns.value = data.columns.map(col => ({
        title: col, key: col,
        ellipsis: {
          tooltip: true,
        },
        resizable: true,
      }))
      tableData.value = data.rows
      pagination.itemCount = data.total
      pagination.page = data.page
      pagination.pageSize = data.size
    } else {
      columns.value = []
      tableData.value = []
      pagination.itemCount = 0
    }
  }).finally(() => {
    tableLoading.value = false;
  });
}

function handleExecute() {
  const params = {
    sql: sqlQuery.value,
    // page: 1,
    // size: 100,
  }
  console.log(params)
  feedback.value = ""
  loading.value = true
  executeSuccess.value = true
  executeQueryRequest(params, { quite: true }).then(res => {
    const { data } = res
    feedback.value = res.code !== 0 ? (res.message || res.error) : ""
    feedback.value = data.error || data.message
    executeSuccess.value = res.code === 0 && !data.error
    if (data.columns && data.rows) {
      columns.value = data.columns.map(col => ({ title: col, key: col }))
      tableData.value = data.rows
      pagination.itemCount = data.total
      pagination.page = data.page
      pagination.pageSize = data.size
    } else {
      columns.value = []
      tableData.value = []
      pagination.itemCount = 0
    }
  }).finally(() => {
    loading.value = false
  });
}

</script>

<style lang="less" scoped>
.editor-wrapper {
  width: 1000px;
  height: 300px;
  border: 1px solid rgb(224, 224, 230);
  border-radius: 3px;
}
</style>