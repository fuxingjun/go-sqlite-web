// 创建一个变量保存 message 实例
let globalMessage;

export function setupMessage(messageInstance) {
  // 在应用启动时传入 app 实例，获取 message
  globalMessage = messageInstance;
}

export const message = {
  success: (...args) => globalMessage?.success(...args),
  error: (...args) => globalMessage?.error(...args),
  warning: (...args) => globalMessage?.warning(...args),
  info: (...args) => globalMessage?.info(...args),
  loading: (...args) => globalMessage?.loading(...args)
}
