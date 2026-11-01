import request from '../utils/request'


export function login(data) {
  return request({
    url: `/adapter-service/user/login`,
    method: 'post',
    data
  });
}


export function getPlat() {
  return request({
    url: `/adapter-service/plat`,
    method: 'GET',
  })
}

export function getCourses(params) {
  return request({
    url: `/adapter-service/courses`,
    method: 'GET',
    params
  })
}

export function getQuestions(data) {
  return request({
    url: `/adapter-service/questions/search`,
    method: 'post',
    data
  })
}

export function updateQuestions(data) {
  return request({
    url: `/adapter-service/questions/${data.id}`,
    method: 'put',
    data
  })
}

export function delQuestions(data) {
  return request({
    url: `/adapter-service/questions/${data.id}`,
    method: 'delete'
  })
}

export function createQuestions(data) {
  return request({
    url: `/adapter-service/questions`,
    method: 'post',
    data
  })
}
// 获取用户列表
export function getUserList() {
  return request({
    url: `/adapter-service/user`,
    method: 'GET'
  })
}
// 删除用户
export function delUser(data) {
  return request({
    url: `/adapter-service/user/${data.id}`,
    method: 'delete'
  })
}

// 新增用户
export function addUser(data) {
  return request({
    url: `/adapter-service/user`,
    method: 'post',
    data
  })
}

export function getLogList(params) {
  return request({
    url: `/adapter-service/logs`,
    method: 'GET',
    params
  })
}

// ===== 模型供应商管理 =====
// 获取当前配置与供应商列表（密钥掩码）
export function getProviders() {
  return request({
    url: `/adapter-service/providers`,
    method: 'GET'
  })
}
// 新增/更新供应商并设为当前（api_key 留空 = 沿用已有密钥）
export function saveProvider(data) {
  return request({
    url: `/adapter-service/providers/save`,
    method: 'post',
    data
  })
}
// 切换当前供应商
export function switchProvider(data) {
  return request({
    url: `/adapter-service/providers/switch`,
    method: 'post',
    data
  })
}
// 删除供应商
export function deleteProvider(data) {
  return request({
    url: `/adapter-service/providers/delete`,
    method: 'post',
    data
  })
}
// 连通性测试
export function testProvider(data) {
  return request({
    url: `/adapter-service/providers/test`,
    method: 'post',
    data,
    timeout: 30 * 1000
  })
}




