<template>
  <div>
    <ul v-if="items && items.length" class="space-y-1">
      <li v-for="(b, i) in items" :key="i" class="flex items-center justify-between" style="border-bottom:1px solid #eee;padding:.5rem 0">
        <div>
          <div style="font-weight:600">{{ b.Title }}</div>
          <div style="font-size:.875rem;color:#6b7280">{{ b.Author }} · <span style="font-style:italic">{{ b.Source }}</span></div>
        </div>
        <button class="px-3 py-1" style="border:1px solid #ddd;border-radius:.375rem" @click="$emit('pick', b)">选择</button>
      </li>
    </ul>
    <div v-else style="color:#6b7280">无结果</div>
  </div>
</template>
<script setup lang="ts">
defineProps<{ items: any[] }>()
</script>
