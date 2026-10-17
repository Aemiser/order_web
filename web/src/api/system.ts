import { get } from './request';

export function getAIModels() {
  return get<Record<string, number>>('/api/system/ai-models');
}
