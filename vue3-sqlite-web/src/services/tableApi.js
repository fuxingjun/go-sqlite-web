import { request } from '@/utils/request';

export function listColumnsRequest(table) {
  return request.get(`/table/${table}/columns`);
}

/**
 * Add a new column to the table
 * @param {string} table - The name of the table
 * @param {{name: string, type: string, notNull: boolean, default: string, pk: boolean}} column - The column definition object
 * @returns {Promise} - The response from the server
 */
export function addColumnRequest(table, column) {
  return request.post(`/table/${table}/columns`, column);
}

export function renameColumnRequest(table, oldName, newName) {
  return request.put(`/table/${table}/columns/${oldName}`, { newName });
}

export function deleteColumnRequest(table, column) {
  return request.delete(`/table/${table}/columns/${column}`);
}

export function listTableIndexesRequest(table) {
  return request.get(`/table/${table}/indexes`);
}

export function deleteTableIndexRequest(table, index) {
  return request.delete(`/table/${table}/indexes/${index}`);
}

/**
 * Add a new index to the table
 * @param {{ name: string, columns: Array<string>, unique: boolean }} params
 */
export function addTableIndexRequest(table, params) {
  return request.post(`/table/${table}/indexes`, params);
}

export function getTableDataRequest(table, params) {
  return request.get(`/table/${table}/rows`, params)
}

export function updateTableRowRequest(table, row) {
  return request.put(`/table/${table}/row`, row);
}

export function deleteTableRowRequest(table, row) {
  return request.delete(`/table/${table}/row`, row);
}

export function newTableRowRequest(table, row) {
  return request.post(`/table/${table}/row`, row);
}

/**
 * Export table data
 * @param {string} table - The name of the table
 * @param {{ columns: Array<string>, page: number, size: number, fileType: string }} params - The export parameters, e.g., { format: 'csv' }
 * @returns {Promise} - The response from the server
 */
export function exportTableDataRequest(table, params) {
  return request.postFile(`/table/${table}/export`, params);
}

export function importTableDataRequest(table, params) {
  return request.post(`/table/${table}/import`, params);
}