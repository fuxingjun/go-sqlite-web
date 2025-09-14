<template>
  <div class="main-list" v-bind=$attrs>
    <div class="flex">
      <n-input v-model:value="pattern" placeholder="Search" clearable />
      &nbsp;
      <n-button type="primary" @click="handleNewTable">
        <n-icon>
          <Add />
        </n-icon>
      </n-button>
      &nbsp;
      <router-link :to="{ name: 'query' }">
        <n-button type="primary">
          SQL
        </n-button>
      </router-link>
    </div>
    <div style="height: 6px;"></div>
    <n-tree block-line :data="tableList" :pattern="pattern" :show-irrelevant-nodes="false" :default-expanded-keys="defaultExpandedKeys" selectable :selected-keys="selectedItem"
      :on-update:selected-keys="handleUpdateSelected" />

    <n-modal v-model:show="newTableModal.visible" preset="dialog" draggable title="New Table" :showIcon="false">
      <template #default>
        <n-form ref="formRef" :model="newTableModal.formModel">
          <n-form-item label="Table Name" path="name" :rule="{ required: true, message: 'Table name is required' }">
            <n-input v-model:value="newTableModal.formModel.name" placeholder="Enter new table name" />
          </n-form-item>
        </n-form>
      </template>
      <template #action>
        <div class="flex justify-end">
          <n-button @click="newTableModal.visible = false">Close</n-button>
          &nbsp;
          <n-button type="primary" @click="createTable" :loading="dialogLoading">Confirm</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<script setup name="MainTree">
import { ref, onMounted, watch, reactive, h } from "vue"
import { RouterLink } from "vue-router"
import { listTablesRequest, createTableRequest, deleteTableRequest } from "@/services"
import { Add } from '@vicons/ionicons5'
import { NButton, NPopconfirm } from 'naive-ui'

const props = defineProps({
  table: {
    type: String,
    default: '',
  },
})

const emits = defineEmits(['select'])

const pattern = ref('')

const tableList = ref([])

const selectedItem = ref([])

watch(() => props.table, (newVal) => {
  selectedItem.value = [newVal]
}, { immediate: true })

function handleUpdateSelected(keys) {
  selectedItem.value = keys
  emits('select', keys[0] || '')
}

const defaultExpandedKeys = ref([])

function getTableData() {
  listTablesRequest().then(({ data }) => {
    tableList.value = data.map(item => {
      return {
        key: item,
        label: item,
        suffix: () => [
          h(
            NPopconfirm,
            {
              'positive-button-props': { type: 'error' },
              onPositiveClick: (e) => handleDropTable(e, item)
            },
            {
              trigger: () => h(NButton, { text: true, type: 'error', onClick: e => e.stopPropagation() }, () => 'Drop'),
              default: () => 'Are you sure to drop this table?'
            }
          ),
        ]
      }
    })
  })
}

function handleDropTable(e, table) {
  e.preventDefault();
  e.stopPropagation()
  deleteTableRequest(table).then(() => {
    getTableData()
    if (selectedItem.value[0] === table) {
      selectedItem.value = []
      emits('select', '')
    }
  })
}

onMounted(() => {
  getTableData()
})

const newTableModal = reactive({
  visible: false,
  formModel: {
    name: '',
  },
})

const dialogLoading = ref(false)
const formRef = ref(null)

function handleNewTable() {
  newTableModal.visible = true
  newTableModal.formModel.name = ''
}

function createTable() {
  formRef.value.validate((errors) => {
    if (errors) {
      return
    }
    dialogLoading.value = true
    createTableRequest(newTableModal.formModel.name).then(() => {
      newTableModal.visible = false
      getTableData()
    }).finally(() => {
      dialogLoading.value = false
    })
  })
  return false
}

</script>

<style lang="less" scope>
.main-list {
  width: 100%;
  padding: 10px;
}
</style>