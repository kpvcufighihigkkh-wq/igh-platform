const statusMap: Record<string, string> = {
  active: '活跃', pending: '待处理', in_progress: '进行中', completed: '已完成',
  loaded: '已装载', empty: '空闲', occupied: '占用', reserved: '预留', disabled: '禁用',
  full: '满车', partial: '部分', packed: '已包装',
  running: '运行中', stopped: '已停机', offline: '离线', online: '在线', unknown: '未检测',
  created: '已创建', acknowledged: '已确认', resolved: '已解决',
  maintenance: '维护中', synced: '已同步', failed: '失败',
  in_use: '在用', deactivated: '停用', fault: '故障',
  idle: '空闲', transit: '运输中', charging: '充电',
}

const levelMap: Record<string, string> = {
  urgent: '紧急', important: '重要', normal: '一般', warning: '警告',
}

const gradeMap: Record<string, string> = {
  AAA: '特等品', AA: '优等品', AA1: '一等品', AA2: '二等品', A1: '合格品', A: '次品',
}

const stationTypeMap: Record<string, string> = {
  winding: '卷绕工位', sorting: '分拣工位', packaging: '包装工位', warehouse: '仓库工位',
}

export function t(key: string): string {
  return statusMap[key] || levelMap[key] || gradeMap[key] || stationTypeMap[key] || key
}

export function tStatus(s: string): string { return statusMap[s] || s }
export function tLevel(l: string): string { return levelMap[l] || l }
export function tGrade(g: string): string { return gradeMap[g] || g }
export function tStation(s: string): string { return stationTypeMap[s] || s }

const sourceMap: Record<string, string> = { manual: '手动', erp: 'ERP', plc: 'PLC' }
export function tSource(s: string): string { return sourceMap[s] || s }

export function formatTime(ts: string | undefined): string {
  if (!ts) return '-'
  return new Date(ts).toLocaleString('zh-CN', { hour12: false })
}
