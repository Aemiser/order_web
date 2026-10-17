import { BASE } from './request';

export interface StreamHandlers {
  onData: (data: string) => void;
  onDone?: () => void;
}

// EventSource 不支持 POST，改用 fetch + ReadableStream 手动解析 SSE
export async function streamRecognition(
  text: string,
  aimodel: number,
  previewType: number,
  handlers: StreamHandlers,
): Promise<void> {
  const res = await fetch(`${BASE}/api/order/recognition/stream`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text, aimodel, previewType }),
  });

  if (!res.ok || !res.body) {
    throw new Error(`HTTP ${res.status}`);
  }

  // 参数错误时后端在进入 SSE 前先写 JSON 错误响应（同样 HTTP 200），故按 content-type 区分
  const contentType = res.headers.get('content-type') ?? '';
  if (!contentType.includes('text/event-stream')) {
    const payload = await res.json().catch(() => null);
    throw new Error(payload?.msg || `HTTP ${res.status}`);
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });

    let idx: number;
    while ((idx = buffer.indexOf('\n\n')) >= 0) {
      const raw = buffer.slice(0, idx);
      buffer = buffer.slice(idx + 2);
      const data = raw
        .split('\n')
        .filter(line => line.startsWith('data: '))
        .map(line => line.slice(6))
        .join('\n');
      if (data === 'DONE') {
        handlers.onDone?.();
        return;
      }
      if (data) handlers.onData(data);
    }
  }

  // 后端正常结束时必发 DONE；走到这里说明连接被截断
  throw new Error('SSE 流未正常结束（未收到 DONE）');
}
