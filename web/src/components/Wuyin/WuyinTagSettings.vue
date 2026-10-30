<script setup lang="ts">
import { ref } from 'vue';
import { Plus, Trash2, Pencil, Tags } from 'lucide-vue-next';
import { ElMessage } from 'element-plus';
import * as wuyinApi from '../../api/wuyin';

// 吴饮标签设置：标签增删改 + 发送人关键词映射（用于识别自动打标）。
// 「未分配」为系统兜底标签，不出现在列表、不可创建同名。

const props = defineProps<{
  tags: wuyinApi.WuyinTagConfig[];
}>();
const emit = defineEmits<{
  (e: 'refreshTags'): void;
  (e: 'status', msg: string): void;
}>();

const editing = ref<{ id: number; name: string; keywords: string; sort: number } | null>(null);
const isCreating = ref(false);
const newName = ref('');
const newKeywords = ref('');
const newSort = ref(0);

const toId = (t: wuyinApi.WuyinTagConfig) => t.ID ?? t.id ?? 0;

const startCreate = () => {
  isCreating.value = true;
  newName.value = '';
  newKeywords.value = '';
  newSort.value = (props.tags.length + 1) * 10;
};

const submitCreate = async () => {
  if (!newName.value.trim()) {
    ElMessage.warning('标签名不能为空');
    return;
  }
  try {
    await wuyinApi.createTag({
      name: newName.value.trim(),
      keywords: newKeywords.value.split(/[,，\s]+/).filter(Boolean),
      sort: newSort.value,
    });
    ElMessage.success('标签已创建');
    isCreating.value = false;
    emit('refreshTags');
  } catch (error) {
    ElMessage.error((error as Error).message || '创建失败');
  }
};

const startEdit = (t: wuyinApi.WuyinTagConfig) => {
  editing.value = {
    id: toId(t),
    name: t.name,
    keywords: (t.keywords ?? []).join(', '),
    sort: t.sort ?? 0,
  };
};

const submitEdit = async () => {
  if (!editing.value || !editing.value.name.trim()) {
    ElMessage.warning('标签名不能为空');
    return;
  }
  try {
    await wuyinApi.updateTag(editing.value.id, {
      name: editing.value.name.trim(),
      keywords: editing.value.keywords.split(/[,，\s]+/).filter(Boolean),
      sort: editing.value.sort,
    });
    ElMessage.success('标签已更新');
    editing.value = null;
    emit('refreshTags');
  } catch (error) {
    ElMessage.error((error as Error).message || '更新失败');
  }
};

const removeTag = async (t: wuyinApi.WuyinTagConfig) => {
  if (!window.confirm(`确定删除标签「${t.name}」？已打该标签的历史订单不受影响。`)) return;
  try {
    await wuyinApi.deleteTag(toId(t));
    ElMessage.success('已删除');
    emit('refreshTags');
  } catch (error) {
    ElMessage.error((error as Error).message || '删除失败');
  }
};
</script>

<template>
  <div class="space-y-6 max-w-4xl">
    <div class="flex items-center justify-between">
      <div class="flex flex-col gap-2">
        <h2 class="text-2xl font-bold tracking-tight text-[var(--text-primary)]">吴饮标签管理</h2>
        <p class="text-[var(--text-muted)] text-sm">仓库标签与发送人关键词映射：识别时按发送人昵称包含关键词自动打标；匹配不到的订单落「未分配」。</p>
      </div>
      <button @click="startCreate" class="px-5 py-2.5 rounded-xl bg-[var(--accent)] text-[var(--text-on-accent)] text-sm font-bold flex items-center gap-2 shadow-lg shadow-[var(--accent-glow)]">
        <Plus class="w-4 h-4" /> 新增标签
      </button>
    </div>

    <!-- 新建表单 -->
    <section v-if="isCreating" class="settings-card rounded-3xl p-6 space-y-4">
      <h3 class="font-bold text-[var(--text-primary)] flex items-center gap-2"><Tags class="w-4 h-4 text-[var(--accent)]" /> 新建标签</h3>
      <div class="grid grid-cols-[1fr_2fr_6rem] gap-3">
        <input v-model="newName" placeholder="标签名，如：湖北仓" class="glass-input px-4 py-2.5 rounded-xl text-sm" />
        <input v-model="newKeywords" placeholder="发送人关键词，逗号分隔，如：湖北, 武汉" class="glass-input px-4 py-2.5 rounded-xl text-sm" />
        <input v-model.number="newSort" type="number" placeholder="排序" class="glass-input px-4 py-2.5 rounded-xl text-sm" />
      </div>
      <div class="flex justify-end gap-3">
        <button @click="isCreating = false" class="px-5 py-2 rounded-xl bg-[var(--bg-hover)] text-[var(--text-muted)] text-sm">取消</button>
        <button @click="submitCreate" class="px-6 py-2 rounded-xl bg-[var(--accent)] text-[var(--text-on-accent)] text-sm font-bold">创建</button>
      </div>
    </section>

    <!-- 列表 -->
    <section class="settings-card rounded-3xl overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left text-[var(--text-faint)] border-b border-[var(--border)]">
            <th class="px-6 py-4 font-semibold">标签名</th>
            <th class="px-6 py-4 font-semibold">匹配关键词（发送人昵称包含即命中）</th>
            <th class="px-6 py-4 font-semibold">排序</th>
            <th class="px-6 py-4 font-semibold text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="editing === null && props.tags.length === 0">
            <td colspan="4" class="px-6 py-12 text-center text-[var(--text-faint)]">还没有标签，点右上角「新增标签」创建</td>
          </tr>
          <tr v-for="t in props.tags" :key="toId(t)" class="border-b border-[var(--border)] last:border-0 hover:bg-[var(--bg-hover)] transition-colors">
            <template v-if="editing?.id === toId(t)">
              <td class="px-6 py-4"><input v-model="editing.name" class="glass-input px-3 py-2 rounded-lg text-sm w-32" /></td>
              <td class="px-6 py-4"><input v-model="editing.keywords" class="glass-input px-3 py-2 rounded-lg text-sm w-full" /></td>
              <td class="px-6 py-4"><input v-model.number="editing.sort" type="number" class="glass-input px-3 py-2 rounded-lg text-sm w-20" /></td>
              <td class="px-6 py-4 text-right space-x-2">
                <button @click="submitEdit" class="px-4 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--text-on-accent)] text-xs font-bold">保存</button>
                <button @click="editing = null" class="px-4 py-1.5 rounded-lg bg-[var(--bg-hover)] text-[var(--text-muted)] text-xs">取消</button>
              </td>
            </template>
            <template v-else>
              <td class="px-6 py-4">
                <span class="bg-[var(--accent-soft-bg)] text-[var(--accent-soft)] text-xs font-bold px-3 py-1 rounded-full">{{ t.name }}</span>
              </td>
              <td class="px-6 py-4 text-[var(--text-secondary)]">
                <span v-for="kw in t.keywords ?? []" :key="kw" class="inline-block bg-[var(--bg-hover-strong)] text-[var(--text-secondary)] text-xs px-2 py-0.5 rounded-md mr-1.5 font-mono">{{ kw }}</span>
                <span v-if="!t.keywords || t.keywords.length === 0" class="text-[var(--text-faint)] text-xs">（无关键词，仅人工指定）</span>
              </td>
              <td class="px-6 py-4 text-[var(--text-muted)]">{{ t.sort ?? 0 }}</td>
              <td class="px-6 py-4 text-right space-x-2">
                <button @click="startEdit(t)" class="px-3 py-1.5 rounded-lg bg-[var(--bg-hover)] hover:bg-[var(--bg-hover-strong)] text-[var(--text-muted)] text-xs inline-flex items-center gap-1"><Pencil class="w-3 h-3" /> 编辑</button>
                <button @click="removeTag(t)" class="px-3 py-1.5 rounded-lg bg-[var(--bg-hover)] hover:bg-red-500/10 hover:text-red-400 text-[var(--text-muted)] text-xs inline-flex items-center gap-1"><Trash2 class="w-3 h-3" /> 删除</button>
              </td>
            </template>
          </tr>
        </tbody>
      </table>
    </section>

    <p class="text-xs text-[var(--text-faint)]">提示：多个标签的关键词命中同一发送人时，按「排序」值小的优先；改关键词只影响之后的识别，历史订单标签可到「订单详情」页逐条修改。</p>
  </div>
</template>
