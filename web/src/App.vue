<template>
  <el-config-provider>
    <el-container style="min-height:100vh">
      <el-header height="56px" class="topbar">
        <div class="brand" @click="$router.push({name:'home'})">go-novel</div>
      </el-header>
      <el-main>
        <router-view v-slot="{ Component, route }">
          <keep-alive :include="['SearchPage']">
            <component v-if="route.meta.keepAlive" :is="Component" />
          </keep-alive>
          <component v-if="!route.meta.keepAlive" :is="Component" />
        </router-view>
      </el-main>
    </el-container>
  </el-config-provider>
</template>

<script setup lang="ts"></script>

<style scoped>
.topbar{ display:flex; align-items:center; }
.brand{ font-weight:700; font-size:20px; cursor:pointer; }
</style>
