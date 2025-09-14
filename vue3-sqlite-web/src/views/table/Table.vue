<template>
  <TableInfo :table="route.params.table || 'Data'" />
</template>

<script setup name="Table">
import { useRoute } from 'vue-router';
import TableInfo from '@/components/tableInfo/TableInfo.vue';
const route = useRoute();
</script>