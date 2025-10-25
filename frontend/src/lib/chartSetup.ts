import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  TimeScale,
  BarElement,
  Filler,
} from 'chart.js'
import 'chartjs-adapter-date-fns'

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  TimeScale,
  BarElement,
  Filler
)

// Default chart options
export const defaultChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    mode: 'index' as const,
    intersect: false,
  },
  plugins: {
    legend: {
      position: 'top' as const,
    },
    tooltip: {
      backgroundColor: 'rgba(0, 0, 0, 0.8)',
      titleColor: 'white',
      bodyColor: 'white',
      borderColor: 'rgba(255, 255, 255, 0.2)',
      borderWidth: 1,
    },
  },
  scales: {
    x: {
      type: 'time' as const,
      time: {
        displayFormats: {
          hour: 'HH:mm',
          day: 'MMM dd',
        },
      },
      grid: {
        color: 'rgba(0, 0, 0, 0.1)',
      },
    },
  },
}

export const temperatureHumidityOptions = {
  ...defaultChartOptions,
  scales: {
    ...defaultChartOptions.scales,
    y: {
      type: 'linear' as const,
      display: true,
      position: 'left' as const,
      title: {
        display: true,
        text: 'Temperature (°C)',
        color: '#ef4444',
      },
      grid: {
        color: 'rgba(0, 0, 0, 0.1)',
      },
    },
    y1: {
      type: 'linear' as const,
      display: true,
      position: 'right' as const,
      title: {
        display: true,
        text: 'Humidity (%)',
        color: '#06b6d4',
      },
      grid: {
        drawOnChartArea: false,
      },
    },
  },
}

export const activityOptions = {
  ...defaultChartOptions,
  scales: {
    ...defaultChartOptions.scales,
    y: {
      type: 'linear' as const,
      display: true,
      title: {
        display: true,
        text: 'Events Count',
      },
      grid: {
        color: 'rgba(0, 0, 0, 0.1)',
      },
      beginAtZero: true,
    },
  },
  plugins: {
    ...defaultChartOptions.plugins,
    tooltip: {
      ...defaultChartOptions.plugins.tooltip,
      callbacks: {
        label: function(context: any) {
          const label = context.dataset.label || '';
          const value = context.parsed.y;
          return `${label}: ${value} events`;
        },
      },
    },
  },
}

export const dayNightOptions = {
  ...defaultChartOptions,
  scales: {
    ...defaultChartOptions.scales,
    y: {
      type: 'linear' as const,
      display: true,
      title: {
        display: true,
        text: 'Mode',
      },
      min: -0.5,
      max: 1.5,
      ticks: {
        stepSize: 1,
        callback: function(value: any) {
          return value === 0 ? 'Day' : value === 1 ? 'Night' : '';
        },
      },
      grid: {
        color: 'rgba(0, 0, 0, 0.1)',
      },
    },
  },
  plugins: {
    ...defaultChartOptions.plugins,
    tooltip: {
      ...defaultChartOptions.plugins.tooltip,
      callbacks: {
        label: function(context: any) {
          const value = context.parsed.y;
          return value === 0 ? 'Day Mode' : 'Night Mode';
        },
      },
    },
  },
}