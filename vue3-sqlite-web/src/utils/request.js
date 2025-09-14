import { tokenStore } from "@/stores/common.js";
import { message } from "@/utils/message.js";

function objToQueryString(obj) {
  const searchParams = new URLSearchParams();
  Object.entries(obj).forEach(([key, value]) => {
    searchParams.append(key, value);
  });
  return searchParams.toString();
}

class Request {
  constructor(options = {}) {
    this.baseURL = options.baseURL || "";
    this.timeout = options.timeout || 10000;
  }
  async request(url, data, options) {
    try {
      const newOptions = {
        ...options,
        headers: {
          rainbow: 'rainbow',
          token: tokenStore.getToken(),
          ...options.headers,
        },

      };
      if (typeof this.timeout === "number") {
        const controller = new AbortController();
        // 设置超时定时器
        this.timeoutId = setTimeout(() => controller.abort(), this.timeout);
        newOptions.signal = controller.signal; // 传入中断信号
      }
      const method = options.method || "GET";
      if (method === "POST" || method === "PUT") {
        if (data instanceof FormData) {
          newOptions.body = data;
        } else {
          newOptions.body = JSON.stringify(data);
          newOptions.headers["Content-Type"] = 'application/json;charset=utf-8';
        }
      } else if (method === "GET" || method === "DELETE") {
        if (data) {
          url = url + "?" + objToQueryString(data);
        }
      }
      let response = await fetch(this.baseURL + url, newOptions);

      // 清除定时器防止误触发
      clearTimeout(this.timeoutId);

      if (!response.ok) {
        console.warn(response);
        if (response.status === 403) {
          const token = prompt("请输入凭据", "");
          if (token) {
            tokenStore.setToken(token);
            return await this.request(url, data, options);
          }
        }
        throw new Error(response.statusText);
      } else {
        data = await response.json();
        if (data.code !== 0) {
          !options.quite && message?.error(data.message || data.error);
          return Promise.reject(data);
        }
        return data;
      }
    } catch (error) {
      clearTimeout(this.timeoutId);

      if (error.name === 'AbortError') {
        // throw new Error(`请求超时 (${timeout}ms)`)
        if (!options.quite) {
          message?.error("request timeout");
        }
        throw new Error("timeout");
      }

      if (!options.quite) {
        message?.error(error.message || "request failed");
      }
      throw error;
    }
  }
  async get(url, params, options) {
    return await this.request(url, params, { method: "GET", ...options });
  }
  async post(url, params, options) {
    return await this.request(url, params, { method: "POST", ...options });
  }
  async put(url, params, options) {
    return await this.request(url, params, { method: "PUT", ...options });
  }
  async delete(url, params, options) {
    return await this.request(url, params, { method: "DELETE", ...options });
  }

  // 新增：处理文件流响应的 GET 方法
  async getFile(url, params, options) {
    return await this.requestFile(url, params, { method: "GET", ...options });
  }

  // 新增：处理文件流响应的 POST 方法
  async postFile(url, params, options) {
    return await this.requestFile(url, params, { method: "POST", ...options });
  }

  // 新增：通用文件流请求方法
  async requestFile(url, data, options) {
    try {
      const newOptions = {
        ...options,
        headers: {
          rainbow: 'rainbow',
          token: tokenStore.getToken(),
          ...options?.headers,
        },
      };
      if (typeof this.timeout === "number") {
        const controller = new AbortController();
        this.timeoutId = setTimeout(() => controller.abort(), this.timeout);
        newOptions.signal = controller.signal;
      }
      const method = options.method || "GET";
      if (method === "POST" || method === "PUT") {
        if (data instanceof FormData) {
          newOptions.body = data;
        } else {
          newOptions.body = JSON.stringify(data);
          newOptions.headers["Content-Type"] = 'application/json;charset=utf-8';
        }
      } else if (method === "GET" || method === "DELETE") {
        if (data) {
          url = url + "?" + objToQueryString(data);
        }
      }
      let response = await fetch(this.baseURL + url, newOptions);

      clearTimeout(this.timeoutId);

      if (!response.ok) {
        throw new Error(response.statusText);
      } else {
        // 从响应头取文件名称
        const filename = response.headers.get('Content-Disposition').match(/filename="(.+)"/)[1];
        // 返回文件流
        return [await response.blob(), filename];
      }
    } catch (error) {
      clearTimeout(this.timeoutId);
      if (error.name === 'AbortError') {
        if (!options?.quite) {
          message?.error("request timeout");
        }
        throw new Error("timeout");
      }
      if (!options?.quite) {
        message?.error(error.message || "request failed");
      }
      throw error;
    }
  }
}

export const request = new Request({ baseURL: process.env.NODE_ENV === "development" ? "/api" : "" });