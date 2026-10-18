<template>
  <div v-if="show" class="fixed inset-0 bg-[var(--overlay)] flex items-center justify-center z-50 transition-opacity duration-300">
    <!-- Modal Container -->
    <div class="bg-[var(--bg-sidebar)] border border-[var(--border-strong)] rounded-2xl p-8 max-w-md w-full mx-4 transform transition-all duration-300 scale-100 flex flex-col">
      <!-- Modal Header -->
      <div class="mb-6">
        <h3 class="text-xl font-bold text-[var(--text-primary)] font-display">填写金额</h3>
        <p class="text-[var(--text-muted)] text-sm mt-2">请为商品填写价格信息</p>
      </div>
      
      <!-- Modal Body -->
      <div class="flex-1 space-y-4">
        <!-- 商品选择 -->
        <div>
          <label class="block text-sm font-medium text-[var(--text-secondary)] mb-2">商品</label>
          <select 
            v-model="selectedProduct" 
            class="w-full px-4 py-2 rounded-xl bg-[var(--bg-app)] border border-[var(--border-strong)] text-[var(--text-secondary)] focus:outline-none focus:border-[var(--accent)] transition-all"
          >
            <option value="">请选择商品</option>
            <option v-for="product in products" :key="product.name" :value="product.name">
              {{ product.name }}
            </option>
          </select>
        </div>
        
        <!-- 规格选择 -->
        <div>
          <label class="block text-sm font-medium text-[var(--text-secondary)] mb-2">规格</label>
          <select 
            v-model="selectedSpec" 
            class="w-full px-4 py-2 rounded-xl bg-[var(--bg-app)] border border-[var(--border-strong)] text-[var(--text-secondary)] focus:outline-none focus:border-[var(--accent)] transition-all"
            :disabled="!selectedProduct"
          >
            <option value="">请选择规格</option>
            <option v-for="spec in availableSpecs" :key="spec" :value="spec">
              {{ spec }}
            </option>
          </select>
        </div>
        
        <!-- 每斤价格 -->
        <div>
          <label class="block text-sm font-medium text-[var(--text-secondary)] mb-2">每斤价格 (元)</label>
          <input 
            type="number" 
            v-model.number="pricePerKg" 
            class="w-full px-4 py-2 rounded-xl bg-[var(--bg-app)] border border-[var(--border-strong)] text-[var(--text-secondary)] focus:outline-none focus:border-[var(--accent)] transition-all"
            placeholder="请输入每斤价格"
            step="0.01"
            min="0"
          >
          <p v-if="errors.pricePerKg" class="text-red-400 text-xs mt-1">{{ errors.pricePerKg }}</p>
        </div>
        
        <!-- 每件价格 -->
        <div>
          <label class="block text-sm font-medium text-[var(--text-secondary)] mb-2">每件价格 (元)</label>
          <input 
            type="number" 
            v-model.number="pricePerItem" 
            class="w-full px-4 py-2 rounded-xl bg-[var(--bg-app)] border border-[var(--border-strong)] text-[var(--text-secondary)] focus:outline-none focus:border-[var(--accent)] transition-all"
            placeholder="请输入每件价格（可选）"
            step="0.01"
            min="0"
          >
        </div>
      </div>
      
      <!-- Modal Footer -->
      <div class="mt-6 flex items-center justify-between">
        <button 
          class="px-6 py-2 rounded-lg bg-[var(--bg-hover)] hover:bg-[var(--bg-hover-strong)] text-[var(--text-muted)] hover:text-[var(--text-primary)] font-medium transition-all"
          @click="handleCancel"
        >
          取消
        </button>
        <button 
          class="px-6 py-2 rounded-lg bg-[var(--accent)] hover:bg-[var(--accent)] text-[var(--text-on-accent)] font-medium transition-all disabled:opacity-50 disabled:cursor-not-allowed"
          @click="handleSubmit"
          :disabled="!isFormValid"
        >
          确认
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';

// 定义组件属性
const props = defineProps<{
  show: boolean;
  products: Array<{
    name: string;
    specs: Array<{
      name: string;
    }>;
  }>;
}>();

// 定义组件事件
const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'save', data: {
    productName: string;
    spec: string;
    pricePerKg: number;
    pricePerItem: number;
  }): void;
}>();

// 响应式数据
const selectedProduct = ref('');
const selectedSpec = ref('');
const pricePerKg = ref(0);
const pricePerItem = ref(0);
const errors = ref<{
  pricePerKg: string;
}>({
  pricePerKg: ''
});

// 计算属性
const availableSpecs = computed(() => {
  if (!selectedProduct.value) return [];
  const product = props.products.find(p => p.name === selectedProduct.value);
  return product ? product.specs.map(s => s.name) : [];
});

const isFormValid = computed(() => {
  return selectedProduct.value && selectedSpec.value && pricePerKg.value > 0 && !errors.value.pricePerKg;
});

// 监听价格变化，进行验证
watch(pricePerKg, (newValue) => {
  if (newValue <= 0) {
    errors.value.pricePerKg = '每斤价格必须大于0';
  } else {
    errors.value.pricePerKg = '';
  }
});

// 方法
const handleCancel = () => {
  resetForm();
  emit('close');
};

const handleSubmit = () => {
  // 验证表单
  if (pricePerKg.value <= 0) {
    errors.value.pricePerKg = '每斤价格必须大于0';
    return;
  }
  
  // 提交数据
  emit('save', {
    productName: selectedProduct.value,
    spec: selectedSpec.value,
    pricePerKg: pricePerKg.value,
    pricePerItem: pricePerItem.value
  });
  
  resetForm();
  emit('close');
};

const resetForm = () => {
  selectedProduct.value = '';
  selectedSpec.value = '';
  pricePerKg.value = 0;
  pricePerItem.value = 0;
  errors.value = {
    pricePerKg: ''
  };
};
</script>

<style scoped>
@import url('https://api.fontshare.com/v2/css?f[]=satoshi@700,500,400&f[]=cabinet-grotesk@800&display=swap');

.font-display {
  font-family: 'Cabinet Grotesk', sans-serif;
}
</style>