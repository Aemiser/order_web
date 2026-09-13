import { reactive } from 'vue';

export const updateState = reactive({
  showDialog: false,
  version: '',
  name: '',
});

export function triggerUpdateDialog(version: string, name: string) {
  updateState.version = version;
  updateState.name = name || version;
  updateState.showDialog = true;
}
