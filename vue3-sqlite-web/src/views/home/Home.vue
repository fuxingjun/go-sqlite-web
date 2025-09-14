<template>
  <div class="flex">
    <MainTree class="main-tree" :table="tableSelected" @select="handleTableChange" />
    <div class="main-panel">
      <router-view></router-view>
    </div>
  </div>
</template>

<script setup name="Home">
import { ref } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import MainTree from '@/components/mainTree/MainTree.vue';

const router = useRouter();
const route = useRoute();

const tableSelected = ref(route.params.table || '');

function handleTableChange(val) {
  tableSelected.value = val;
  if (val) {
    const target = { name: "table", params: { table: val, action: route.params.action || "Data" } }
    router.replace(target);
  }
}

</script>

<style lang="less" scoped>
.main-tree {
  flex: 0 0 300px;
}

.main-panel {
  flex: 1 0 68.2%;
  width: 0;
  padding: 10px 10px 10px 0;
  box-sizing: border-box;
}
</style>