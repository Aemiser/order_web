<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import * as wuyinApi from '../../api/wuyin';
import WuyinInputView from './WuyinInputView.vue';
import WuyinDetailsView from './WuyinDetailsView.vue';
import WuyinCheckoutView from './WuyinCheckoutView.vue';
import WuyinTagSettings from './WuyinTagSettings.vue';

// 吴饮工作台：品牌内四页签（输入/详情/结账/标签设置）。
// 页签由 Sidebar 经 App 透传（wy-input / wy-details / wy-checkout / wy-settings），
// 本组件只负责承载视图与共享的标签列表。返回吴饮未配置等后端错误时给出引导。

const props = defineProps<{
  tab: string;
}>();
const emit = defineEmits<{
  (e: 'status', msg: string): void;
}>();

const tags = ref<wuyinApi.WuyinTagConfig[]>([]);
const tagNames = computed(() => {
  const names = tags.value.map(t => t.name);
  if (!names.includes('未分配')) names.unshift('未分配');
  return names;
});
const dbError = ref('');

const refreshTags = async () => {
  try {
    dbError.value = '';
    tags.value = (await wuyinApi.getTags()) ?? [];
  } catch (error) {
    dbError.value = (error as Error).message || '标签加载失败';
    tags.value = [];
  }
};

onMounted(refreshTags);
</script>

<template>
  <div class="space-y-6">
    <!-- 吴饮库未配置/后端错误引导 -->
    <div v-if="dbError" class="settings-card rounded-3xl p-6 flex items-start gap-4">
      <div class="flex-1">
        <h3 class="font-bold text-[var(--text-primary)] mb-1">吴饮功能暂不可用</h3>
        <p class="text-sm text-[var(--text-muted)]">{{ dbError }}</p>
        <p class="text-xs text-[var(--text-faint)] mt-2">首次使用：在服务器 settings.yaml 配置 dbWuyin 并建库，然后执行 go run main.go -dbwuyin 建表。</p>
      </div>
      <button @click="refreshTags" class="px-4 py-2 rounded-xl bg-[var(--bg-hover)] text-[var(--text-secondary)] text-sm">重试</button>
    </div>

    <WuyinInputView v-if="props.tab === 'wy-input'" :tags="tagNames" @status="msg => emit('status', msg)" />
    <WuyinDetailsView v-if="props.tab === 'wy-details'" :tags="tags" @status="msg => emit('status', msg)" />
    <WuyinCheckoutView v-if="props.tab === 'wy-checkout'" />
    <WuyinTagSettings v-if="props.tab === 'wy-settings'" :tags="tags" @refresh-tags="refreshTags" @status="msg => emit('status', msg)" />
  </div>
</template>
