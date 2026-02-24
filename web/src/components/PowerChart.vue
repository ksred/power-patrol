<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue'
import { use } from 'echarts/core'
import { LineChart } from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import type { PowerSample, TimePeriod } from '../types/api'

use([TitleComponent, TooltipComponent, GridComponent, LegendComponent, LineChart, CanvasRenderer])

interface SystemStatsSample {
  timestamp: string
  cpu_percent: number
  memory_percent: number
}

const props = defineProps<{
  data: PowerSample[]
  systemStats: SystemStatsSample[]
  period: TimePeriod
}>()

const chartRef = ref<InstanceType<typeof VChart> | null>(null)

const chartOption = computed(() => {
  const timestamps = props.data.map(d => new Date(d.timestamp).toLocaleTimeString())
  const cpuData = props.data.map(d => d.cpu_watts)
  const gpuData = props.data.map(d => d.gpu_watts)
  const aneData = props.data.map(d => d.ane_watts)

  // Create system stats data aligned with power timestamps
  const systemMap = new Map<string, { cpu: number; mem: number }>()
  props.systemStats.forEach(s => {
    systemMap.set(new Date(s.timestamp).toLocaleTimeString(), {
      cpu: s.cpu_percent,
      mem: s.memory_percent
    })
  })

  // Interpolate system stats to match power data timestamps
  const cpuPercentData: (number | null)[] = []
  const memoryPercentData: (number | null)[] = []
  let lastCpu: number | null = null
  let lastMem: number | null = null

  timestamps.forEach(t => {
    const stats = systemMap.get(t)
    if (stats) {
      lastCpu = stats.cpu
      lastMem = stats.mem
    }
    cpuPercentData.push(lastCpu)
    memoryPercentData.push(lastMem)
  })

  return {
    backgroundColor: 'transparent',
    grid: {
      left: 55,
      right: 55,
      top: 40,
      bottom: 30
    },
    tooltip: {
      trigger: 'axis',
      backgroundColor: '#1a1a1a',
      borderColor: 'rgba(255,255,255,0.1)',
      textStyle: {
        color: '#fafafa',
        fontSize: 12
      },
      formatter: (params: any) => {
        let html = `<div style="padding: 4px 8px;">`
        html += `<div style="color: #6b6b6b; margin-bottom: 4px;">${params[0].axisValue}</div>`
        params.forEach((p: any) => {
          if (p.value !== null && p.value !== undefined) {
            const unit = p.seriesName.includes('%') ? '%' : 'W'
            const value = p.seriesName.includes('%') ? p.value.toFixed(0) : p.value.toFixed(2)
            html += `<div style="display: flex; justify-content: space-between; gap: 16px;">
              <span style="color: ${p.color};">${p.seriesName}</span>
              <span style="font-weight: 500;">${value}${unit}</span>
            </div>`
          }
        })
        html += `</div>`
        return html
      }
    },
    legend: {
      show: true,
      top: 10,
      right: 60,
      textStyle: {
        color: '#6b6b6b',
        fontSize: 11
      },
      itemWidth: 12,
      itemHeight: 12
    },
    xAxis: {
      type: 'category',
      data: timestamps,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: {
        color: '#6b6b6b',
        fontSize: 10,
        interval: 'auto'
      }
    },
    yAxis: [
      {
        type: 'value',
        name: 'Watts',
        nameTextStyle: { color: '#6b6b6b', fontSize: 10 },
        position: 'left',
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: {
          lineStyle: {
            color: 'rgba(255,255,255,0.06)'
          }
        },
        axisLabel: {
          color: '#6b6b6b',
          fontSize: 10,
          formatter: '{value}W'
        }
      },
      {
        type: 'value',
        name: '%',
        nameTextStyle: { color: '#6b6b6b', fontSize: 10 },
        position: 'right',
        min: 0,
        max: 100,
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: { show: false },
        axisLabel: {
          color: '#6b6b6b',
          fontSize: 10,
          formatter: '{value}%'
        }
      }
    ],
    series: [
      {
        name: 'CPU Power',
        type: 'line',
        yAxisIndex: 0,
        data: cpuData,
        smooth: true,
        symbol: 'none',
        lineStyle: {
          width: 2,
          color: '#f97316'
        },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(249, 115, 22, 0.3)' },
              { offset: 1, color: 'rgba(249, 115, 22, 0)' }
            ]
          }
        }
      },
      {
        name: 'GPU',
        type: 'line',
        yAxisIndex: 0,
        data: gpuData,
        smooth: true,
        symbol: 'none',
        lineStyle: {
          width: 2,
          color: '#a78bfa'
        },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(167, 139, 250, 0.2)' },
              { offset: 1, color: 'rgba(167, 139, 250, 0)' }
            ]
          }
        }
      },
      {
        name: 'ANE',
        type: 'line',
        yAxisIndex: 0,
        data: aneData,
        smooth: true,
        symbol: 'none',
        lineStyle: {
          width: 2,
          color: '#22d3ee'
        },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(34, 211, 238, 0.15)' },
              { offset: 1, color: 'rgba(34, 211, 238, 0)' }
            ]
          }
        }
      },
      {
        name: 'CPU %',
        type: 'line',
        yAxisIndex: 1,
        data: cpuPercentData,
        smooth: true,
        symbol: 'none',
        connectNulls: true,
        lineStyle: {
          width: 2,
          color: '#10b981',
          type: 'dashed'
        }
      },
      {
        name: 'Memory %',
        type: 'line',
        yAxisIndex: 1,
        data: memoryPercentData,
        smooth: true,
        symbol: 'none',
        connectNulls: true,
        lineStyle: {
          width: 2,
          color: '#6366f1',
          type: 'dashed'
        }
      }
    ]
  }
})

// Auto-resize chart when data changes
watch(() => props.data.length, () => {
  chartRef.value?.resize()
})

onMounted(() => {
  window.addEventListener('resize', () => chartRef.value?.resize())
})
</script>

<template>
  <div class="card chart-card">
    <div class="card-header">Power & System History</div>
    <div class="chart-container">
      <VChart
        ref="chartRef"
        :option="chartOption"
        :autoresize="true"
        class="chart"
      />
    </div>
  </div>
</template>
