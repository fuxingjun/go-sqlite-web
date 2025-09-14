<template>
  <div>
    <n-tabs type="card" v-model:value="activeTab" :on-update:value="handleClickTab">
      <n-tab v-for="tab in tabs" :key="tab" :name="tab">
        {{ tab }}
      </n-tab>
    </n-tabs>
    <div>
      <component :is="componentMap[activeTab]" :table="table"></component>
    </div>
  </div>
</template>

<script setup name="TableInfo">
import { ref } from "vue";
import { useRoute, useRouter } from 'vue-router';
import Columns from '../columns/Columns.vue';
import Indexes from '../indexes/Indexes.vue';
import TableData from '../tableData/TableData.vue';

const componentMap = {
  'Data': TableData,
  'Columns': Columns,
  'Indexes': Indexes,
};

const props = defineProps({
  table: {
    type: String,
    default: '',
  },
});

const router = useRouter();
const route = useRoute();

const activeTab = ref(route.params.action || 'Data');

const tabs = ref(["Data", "Columns", "Indexes"]);

function handleClickTab(tab) {
  activeTab.value = tab;
  router.replace({ params: { ...route.params, action: tab } });
}

</script>