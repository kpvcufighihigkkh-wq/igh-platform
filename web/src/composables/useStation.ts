import { ref } from 'vue'
import http from '../api/http'

export interface StationInfo {
  identified: boolean
  ip: string
  station?: { id: number; ip: string; station_type: string; line_code: string; description: string }
}

const stationInfo = ref<StationInfo | null>(null)

export function useStation() {
  async function identify() {
    if (stationInfo.value) return stationInfo.value
    try {
      const res: any = await http.get('/station/identify')
      stationInfo.value = res.data
    } catch {
      stationInfo.value = { identified: false, ip: '' }
    }
    return stationInfo.value!
  }
  return { stationInfo, identify }
}
