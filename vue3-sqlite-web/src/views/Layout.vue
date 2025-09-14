<template>
  <header></header>
  <router-view></router-view>
</template>

<script setup name="Layout">
import { onMounted } from 'vue';
import { useMessage } from "naive-ui";
import { setupMessage } from "@/utils/message";

onMounted(() => {
  setupMessage(useMessage());
});
</script>