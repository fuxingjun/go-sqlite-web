import { request } from '@/utils/request';

/**
 * 请求数据库信息
 * @returns
 */
export function databaseInfoRequest() {
  return request.get("/db/info");
}

export function listTablesRequest() {
  return request.get("/db/tables");
}

export function createTableRequest(tableName) {
  return request.post("/db/table", { tableName });
}

export function deleteTableRequest(tableName) {
  return request.delete(`/db/table/${tableName}`);
}

export function executeQueryRequest(params, options) {
  return request.post("/db/query", params, options);
}